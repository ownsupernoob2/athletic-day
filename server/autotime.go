package main

import "math"

// Automatic start times.
//
// Each event and age group (a "block") lasts its number of heats times the minutes per heat.
// Blocks are put on the day so that:
//   - nobody is entered in two events at the same time,
//   - nothing in the same place overlaps: all track events share one track, and each field
//     event is its own place (so javelin can run beside discus, but not beside itself).
//
// Times typed in by hand are fixed: they stay where they are and everything else works around them.
// Each place has a queue of blocks in schedule order, and a queue never runs out of order. We repeatedly
// take the queue whose next block can start earliest, which is how track and field end up side by side.

type planBlock struct {
	event    int64
	age      string
	kind     string
	heats    int
	fixed    bool
	students map[int64]struct{}
	dur      int // minutes
	start    int
	end      int
}

func (b *planBlock) sameVenue(o *planBlock) bool {
	if b.kind == "track" && o.kind == "track" {
		return true
	}
	return b.kind == "field" && o.kind == "field" && b.event == o.event
}

// durations says how long an event and age group lasts. Each event can have its own time,
// either for every heat or for the whole thing; otherwise it is the default minutes per heat times its heats.
type durations struct {
	perHeat float64
	event   map[int64]eventLength
}

type eventLength struct {
	minutes float64
	per     string // "heat" or "event"
}

func (d durations) of(event int64, heats int) int {
	minutes := float64(heats) * d.perHeat
	if e, ok := d.event[event]; ok && e.minutes > 0 {
		if e.per == "event" {
			minutes = e.minutes
		} else {
			minutes = float64(heats) * e.minutes
		}
	}
	return max(1, int(math.Ceil(minutes)))
}

func (s *Server) durations() durations {
	d := durations{perHeat: heatMinutes(s.db), event: map[int64]eventLength{}}
	if evs, err := s.listEvents(false); err == nil {
		for _, e := range evs {
			if e.Duration > 0 {
				d.event[e.ID] = eventLength{e.Duration, e.DurationPer}
			}
		}
	}
	return d
}

// planTimes sets start/end on every block that is not fixed and returns the minute the last block finishes.
func planTimes(blocks []*planBlock, dayStart int) (finish int) {
	n := len(blocks)
	shares := make([][]bool, n) // do these two blocks have a student in common?
	for i := range shares {
		shares[i] = make([]bool, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			a, b := blocks[i].students, blocks[j].students
			if len(a) > len(b) {
				a, b = b, a
			}
			for id := range a {
				if _, ok := b[id]; ok {
					shares[i][j], shares[j][i] = true, true
					break
				}
			}
		}
	}

	var placed []int // indexes of blocks that have a time
	for i, b := range blocks {
		if b.fixed {
			placed = append(placed, i)
			finish = max(finish, b.end)
		}
	}

	// One queue for the track, one per field event.
	type queue struct {
		items []int
		after int // the previous block in this queue finished here: the next one starts no earlier
	}
	var queues []*queue
	field := map[int64]*queue{}
	var track *queue
	for i, b := range blocks {
		if b.fixed {
			continue
		}
		if b.kind == "track" {
			if track == nil {
				track = &queue{after: dayStart}
				queues = append(queues, track)
			}
			track.items = append(track.items, i)
			continue
		}
		q := field[b.event]
		if q == nil {
			q = &queue{after: dayStart}
			field[b.event] = q
			queues = append(queues, q)
		}
		q.items = append(q.items, i)
	}

	// earliest minute block i can start without clashing with anything already placed.
	earliest := func(i, from int) int {
		b := blocks[i]
		t := from
		for {
			next := -1
			for _, p := range placed {
				o := blocks[p]
				if t < o.end && o.start < t+b.dur && (b.sameVenue(o) || shares[i][p]) {
					if next == -1 || o.end < next {
						next = o.end
					}
				}
			}
			if next == -1 {
				return t
			}
			t = next
		}
	}

	for {
		bestQ, bestStart := -1, 0
		for qi, q := range queues {
			if len(q.items) == 0 {
				continue
			}
			if s := earliest(q.items[0], q.after); bestQ == -1 || s < bestStart {
				bestQ, bestStart = qi, s
			}
		}
		if bestQ == -1 {
			break
		}
		q := queues[bestQ]
		i := q.items[0]
		q.items = q.items[1:]
		blocks[i].start = bestStart
		blocks[i].end = bestStart + blocks[i].dur
		q.after = blocks[i].end
		placed = append(placed, i)
		finish = max(finish, blocks[i].end)
	}
	return finish
}
