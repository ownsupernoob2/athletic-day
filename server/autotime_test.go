package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func pb(event int64, kind string, heats int, fixedAt int, students ...int64) *planBlock {
	b := &planBlock{event: event, age: "Under 14", kind: kind, heats: heats, students: map[int64]struct{}{}, dur: heats * 3}
	for _, s := range students {
		b.students[s] = struct{}{}
	}
	if fixedAt > 0 {
		b.fixed, b.start, b.end = true, fixedAt, fixedAt+b.dur
	}
	return b
}

func TestPlanTimes(t *testing.T) {
	const day = 9 * 60

	// Track events share one track, so they go one after another.
	a, b := pb(1, "track", 2, 0, 1), pb(2, "track", 1, 0, 2)
	if planTimes([]*planBlock{a, b}, day); a.start != day || b.start != day+6 {
		t.Errorf("track not sequential: %d and %d", a.start, b.start)
	}

	// Track and field with nobody in common run side by side.
	tr, f := pb(1, "track", 2, 0, 1), pb(7, "field", 2, 0, 2)
	planTimes([]*planBlock{tr, f}, day)
	if tr.start != day || f.start != day {
		t.Errorf("no shared students, both should start at the beginning: %d and %d", tr.start, f.start)
	}

	// With a student in both, the field event waits for the track event.
	tr, f = pb(1, "track", 2, 0, 1), pb(7, "field", 2, 0, 1, 2)
	planTimes([]*planBlock{tr, f}, day)
	if f.start != tr.end {
		t.Errorf("field should start when track ends (%d), got %d", tr.end, f.start)
	}

	// Different field events are different places, but one event is only in one place at a time.
	d1, d2, j := pb(7, "field", 2, 0, 1), pb(7, "field", 2, 0, 2), pb(8, "field", 2, 0, 3)
	d2.age = "Under 15"
	planTimes([]*planBlock{d1, d2, j}, day)
	if d1.start != day || j.start != day || d2.start != d1.end {
		t.Errorf("discus U14 %d, javelin %d, discus U15 %d (want %d, %d, %d)", d1.start, j.start, d2.start, day, day, d1.end)
	}

	// A time fixed by hand stays, and the track works around a student who is busy then.
	fixed := pb(7, "field", 1, 10*60, 1) // discus 10:00-10:03 with student 1
	t1, t2 := pb(1, "track", 1, 0, 1), pb(2, "track", 1, 0, 5)
	t1.dur, t2.dur = 120, 5 // the first track block would run from 09:00 to 11:00, over the discus
	planTimes([]*planBlock{fixed, t1, t2}, day)
	if fixed.start != 10*60 {
		t.Errorf("fixed block moved to %d", fixed.start)
	}
	if t1.start != fixed.end {
		t.Errorf("student 1 is at the discus 10:00-10:03, so the track block starts at %d, got %d", fixed.end, t1.start)
	}
	if t2.start < t1.end {
		t.Errorf("second track block overlaps the first: %d < %d", t2.start, t1.end)
	}
}

func TestAutomaticTimesAvoidClashes(t *testing.T) {
	s, do := organiserClient(t)
	do("PUT", "/api/settings", `{"event_date":"2026-06-01","heat_minutes":2}`)

	// 24 students aged 13, every one in 100m, 200m, Discus and High Jump: lots of
	// track and field pairs that would clash if they both started at 09:00.
	for i := 0; i < 24; i++ {
		res, _ := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
			fmt.Sprintf("S%d", i), fmt.Sprintf("L%02d", i), "2012-12-31", "SSH", now(), now())
		id, _ := res.LastInsertId()
		s.db.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,1),(?,2),(?,7),(?,8)`, id, id, id, id)
	}
	do("POST", "/api/schedule/generate", `{"randomize":false}`)
	code, body := do("POST", "/api/schedule/times", `{"start":"09:00","end":"17:00"}`)
	if code != 200 {
		t.Fatalf("times: %d %s", code, body)
	}
	var d struct {
		Clashes ClashSummary      `json:"clashes"`
		Timing  map[string]string `json:"timing"`
		Heats   []Heat            `json:"heats"`
	}
	if err := json.Unmarshal(body, &d); err != nil {
		t.Fatal(err)
	}
	if d.Clashes.Pairs != 0 {
		t.Errorf("automatic times left %d clashes: %+v", d.Clashes.Pairs, d.Clashes.List)
	}
	if d.Timing["first"] != "09:00" || d.Timing["finish"] == "" {
		t.Errorf("timing = %v", d.Timing)
	}

	// One track: no two track events overlap.
	type span struct{ from, to int }
	heats := map[string]int{}
	for _, h := range d.Heats {
		if h.Kind == "track" {
			heats[h.EventName]++
		}
	}
	var track []span
	for _, h := range d.Heats {
		if h.Kind == "track" && h.Number == 1 {
			m, _ := parseClock(h.Time)
			track = append(track, span{m, m + heats[h.EventName]*2})
		}
	}
	for i := range track {
		for j := i + 1; j < len(track); j++ {
			if track[i].from < track[j].to && track[j].from < track[i].to {
				t.Errorf("two track events overlap: %+v and %+v", track[i], track[j])
			}
		}
	}

	// A typed-in time is kept and the rest still has no clashes.
	do("PATCH", "/api/schedule/block-time", `{"event_id":7,"age_group":"Under 14","time":"09:00"}`)
	code, body = do("POST", "/api/schedule/times", `{"start":"09:00","end":"17:00"}`)
	if err := json.Unmarshal(body, &d); code != 200 || err != nil {
		t.Fatalf("times again: %d %v", code, err)
	}
	for _, h := range d.Heats {
		if h.EventID == 7 && (h.Time != "09:00" || !h.TimeFixed) {
			t.Errorf("fixed discus time changed: %s fixed=%v", h.Time, h.TimeFixed)
		}
	}
	if d.Clashes.Pairs != 0 {
		t.Errorf("%d clashes after a fixed time: %+v", d.Clashes.Pairs, d.Clashes.List)
	}
}

func TestDurationOf(t *testing.T) {
	d := durations{perHeat: 2, event: map[int64]eventLength{
		7: {minutes: 90, per: "event"}, // discus: 90 minutes for each age group, however many heats
		8: {minutes: 4, per: "heat"},   // high jump: 4 minutes a heat
	}}
	for _, c := range []struct {
		event int64
		heats int
		want  int
	}{
		{1, 3, 6},  // no setting: default 2 min x 3 heats
		{7, 1, 90}, // whole event
		{7, 9, 90}, // still 90 with more heats
		{8, 3, 12}, // own time per heat
		{1, 0, 1},  // never less than a minute
	} {
		if got := d.of(c.event, c.heats); got != c.want {
			t.Errorf("event %d, %d heats: %d min, want %d", c.event, c.heats, got, c.want)
		}
	}
	if got := (durations{perHeat: 1.5}).of(1, 3); got != 5 { // 4.5 rounds up
		t.Errorf("1.5 x 3 = %d, want 5", got)
	}
}

func TestEventLengthsSetTheSchedule(t *testing.T) {
	s, do := organiserClient(t)
	do("PUT", "/api/settings", `{"event_date":"2026-06-01","heat_minutes":2}`)
	// Discus (field, id 7) for 6 students aged 13 and 6 aged 14: two age groups, one heat each.
	for i := 0; i < 12; i++ {
		age := 13
		if i >= 6 {
			age = 14
		}
		res, _ := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
			fmt.Sprintf("S%d", i), fmt.Sprintf("L%02d", i), fmt.Sprintf("%d-12-31", 2026-age-1), "SSH", now(), now())
		id, _ := res.LastInsertId()
		s.db.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,7)`, id)
	}
	do("POST", "/api/schedule/generate", `{"randomize":false}`)

	gap := func() int { // minutes between the Under 14 and Under 15 discus start times
		code, body := do("POST", "/api/schedule/times", `{"start":"10:00","end":"14:00"}`)
		var d struct {
			Heats []Heat `json:"heats"`
		}
		if code != 200 || json.Unmarshal(body, &d) != nil {
			t.Fatalf("times: %d %s", code, body)
		}
		at := map[string]int{}
		for _, h := range d.Heats {
			if h.EventID == 7 {
				at[h.AgeGroup], _ = parseClock(h.Time)
			}
		}
		return at["Under 15"] - at["Under 14"]
	}
	if g := gap(); g != 2 {
		t.Errorf("default: one heat of 2 minutes, next age group %d minutes later, want 2", g)
	}

	// 90 minutes for the whole event (each age group).
	if code, out := do("PATCH", "/api/events/7", `{"duration":90,"duration_per":"event"}`); code != 200 {
		t.Fatalf("set duration: %d %s", code, out)
	}
	if g := gap(); g != 90 {
		t.Errorf("90 minutes per event: next age group %d minutes later, want 90", g)
	}
	// 10 minutes for each heat.
	do("PATCH", "/api/events/7", `{"duration":10,"duration_per":"heat"}`)
	if g := gap(); g != 10 {
		t.Errorf("10 minutes per heat, one heat: %d, want 10", g)
	}

	// The event list reports it, bad values are refused, and 0 goes back to the default.
	_, list := do("GET", "/api/events", "")
	var evs []Event
	json.Unmarshal(list, &evs)
	for _, e := range evs {
		if e.ID == 7 && (e.Duration != 10 || e.DurationPer != "heat") {
			t.Errorf("event 7 reported as %v per %q", e.Duration, e.DurationPer)
		}
	}
	for _, bad := range []string{`{"duration":0.1}`, `{"duration":2000}`, `{"duration":-5}`, `{"duration_per":"day"}`} {
		if code, _ := do("PATCH", "/api/events/7", bad); code != 400 {
			t.Errorf("%s accepted: %d", bad, code)
		}
	}
	do("PATCH", "/api/events/7", `{"duration":0}`)
	if g := gap(); g != 2 {
		t.Errorf("after resetting, next age group %d minutes later, want 2", g)
	}
}
