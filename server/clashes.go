package main

import (
	"fmt"
	"sort"
)

// A clash is one group of students entered in two events that are on at the same time,
// so they can't be at both. We only know when each event and age group starts, so each
// is taken to last `heat_minutes` for every heat it has.

type ClashBlock struct {
	EventID   int64  `json:"event_id"`
	EventName string `json:"event_name"`
	AgeGroup  string `json:"age_group"`
	Time      string `json:"time"`
	Until     string `json:"until"` // estimated finish
}

type Clash struct {
	A        ClashBlock    `json:"a"`
	B        ClashBlock    `json:"b"`
	Count    int           `json:"count"`    // students in both
	Students []HeatStudent `json:"students"` // the first few
}

type ClashSummary struct {
	Pairs    int     `json:"pairs"`    // pairs of events that overlap with shared students
	Students int     `json:"students"` // different students affected
	List     []Clash `json:"list"`     // biggest first, limited
}

const (
	maxClashList     = 60 // pairs returned
	maxClashStudents = 12 // names shown per pair
)

type timedBlock struct {
	ClashBlock
	start, end int
	students   map[int64]HeatStudent
}

func clashesOf(heats []Heat, perHeat int) ClashSummary {
	index := map[string]*timedBlock{}
	var order []*timedBlock
	counts := map[*timedBlock]int{}
	for _, h := range heats {
		key := fmt.Sprintf("%d|%s", h.EventID, h.AgeGroup)
		b, ok := index[key]
		if !ok {
			b = &timedBlock{ClashBlock: ClashBlock{EventID: h.EventID, EventName: h.EventName, AgeGroup: h.AgeGroup, Time: h.Time}, students: map[int64]HeatStudent{}}
			index[key] = b
			order = append(order, b)
		}
		counts[b]++
		for _, st := range h.Students {
			b.students[st.ID] = st
		}
	}

	var timed []*timedBlock
	for _, b := range order {
		start, ok := parseClock(b.Time)
		if !ok {
			continue
		}
		b.start = start
		b.end = start + counts[b]*perHeat
		b.Until = fmtClock(b.end % (24 * 60))
		timed = append(timed, b)
	}

	var out ClashSummary
	affected := map[int64]bool{}
	for i := 0; i < len(timed); i++ {
		for j := i + 1; j < len(timed); j++ {
			a, b := timed[i], timed[j]
			if a.EventID == b.EventID || a.start >= b.end || b.start >= a.end {
				continue
			}
			var shared []HeatStudent
			for id, st := range a.students {
				if _, both := b.students[id]; both {
					shared = append(shared, st)
					affected[id] = true
				}
			}
			if len(shared) == 0 {
				continue
			}
			sort.Slice(shared, func(x, y int) bool {
				if shared[x].LastName != shared[y].LastName {
					return shared[x].LastName < shared[y].LastName
				}
				return shared[x].FirstName < shared[y].FirstName
			})
			c := Clash{A: a.ClashBlock, B: b.ClashBlock, Count: len(shared), Students: shared}
			if len(c.Students) > maxClashStudents {
				c.Students = c.Students[:maxClashStudents]
			}
			out.Pairs++
			out.List = append(out.List, c)
		}
	}
	sort.SliceStable(out.List, func(x, y int) bool { return out.List[x].Count > out.List[y].Count })
	if len(out.List) > maxClashList {
		out.List = out.List[:maxClashList]
	}
	if out.List == nil {
		out.List = []Clash{}
	}
	out.Students = len(affected)
	return out
}
