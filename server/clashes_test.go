package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func heatOf(id, event int64, name, age, time string, students ...int64) Heat {
	h := Heat{ID: id, EventID: event, EventName: name, AgeGroup: age, Number: 1, Time: time, Students: []HeatStudent{}}
	for _, s := range students {
		h.Students = append(h.Students, HeatStudent{ID: s, FirstName: fmt.Sprintf("S%d", s), LastName: "Test", AgeGroup: age})
	}
	return h
}

func TestClashesOf(t *testing.T) {
	d := durations{perHeat: 3}
	cases := []struct {
		name  string
		heats []Heat
		pairs int
		count int // students in the biggest clash
	}{
		{"same start, one shared student", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1, 2), heatOf(2, 7, "Discus", "Under 14", "09:00", 2, 3)}, 1, 1},
		{"same start, nobody in both", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1), heatOf(2, 7, "Discus", "Under 14", "09:00", 2)}, 0, 0},
		{"second starts as the first ends", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1), heatOf(2, 7, "Discus", "Under 14", "09:03", 1)}, 0, 0},
		{"second starts a minute before the first ends", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1), heatOf(2, 7, "Discus", "Under 14", "09:02", 1)}, 1, 1},
		{"more heats last longer", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1), heatOf(3, 1, "100m", "Under 14", "09:00", 4), heatOf(2, 7, "Discus", "Under 14", "09:05", 1)}, 1, 1}, // 2 heats = 6 min
		{"a time that is not set is ignored", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1), heatOf(2, 7, "Discus", "Under 14", "", 1)}, 0, 0},
		{"one event on twice is not a clash", []Heat{heatOf(1, 1, "100m", "Under 14", "09:00", 1), heatOf(2, 1, "100m", "Under 15", "09:00", 1)}, 0, 0},
	}
	for _, c := range cases {
		got := clashesOf(c.heats, d)
		if got.Pairs != c.pairs {
			t.Errorf("%s: %d pairs, want %d", c.name, got.Pairs, c.pairs)
		}
		if c.pairs > 0 && (len(got.List) == 0 || got.List[0].Count != c.count) {
			t.Errorf("%s: biggest clash %+v, want %d students", c.name, got.List, c.count)
		}
		if got.List == nil {
			t.Errorf("%s: list should be empty, not null", c.name)
		}
	}

	// Students are counted once however many clashes they are in.
	h := []Heat{
		heatOf(1, 1, "100m", "Under 14", "09:00", 1, 2),
		heatOf(2, 7, "Discus", "Under 14", "09:00", 1, 2),
		heatOf(3, 8, "High Jump", "Under 14", "09:00", 1),
	}
	if got := clashesOf(h, d); got.Pairs != 3 || got.Students != 2 {
		t.Errorf("3 events at once: %d pairs, %d students; want 3 and 2", got.Pairs, got.Students)
	}
}

func TestScheduleReportsClashes(t *testing.T) {
	s, do := organiserClient(t)
	do("PUT", "/api/settings", `{"event_date":"2026-06-01"}`)
	for i := 0; i < 8; i++ { // all Under 14, all in 100m (track) and Discus (field)
		res, _ := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
			fmt.Sprintf("S%d", i), fmt.Sprintf("L%02d", i), "2012-12-31", "SSH", now(), now())
		id, _ := res.LastInsertId()
		s.db.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,1),(?,7)`, id, id)
	}
	do("POST", "/api/schedule/generate", `{"randomize":false}`)

	clashes := func() ClashSummary {
		code, body := do("GET", "/api/schedule", "")
		var d struct {
			Clashes ClashSummary `json:"clashes"`
		}
		if code != 200 || json.Unmarshal(body, &d) != nil {
			t.Fatalf("schedule: %d %s", code, body)
		}
		return d.Clashes
	}
	if c := clashes(); c.Pairs != 0 {
		t.Errorf("no times set yet, but %d clashes", c.Pairs)
	}

	// Put the 100m and the discus both at 09:00 by hand, so everyone in both is in two places.
	do("PATCH", "/api/schedule/block-time", `{"event_id":1,"age_group":"Under 14","time":"09:00"}`)
	do("PATCH", "/api/schedule/block-time", `{"event_id":7,"age_group":"Under 14","time":"09:00"}`)
	if c := clashes(); c.Pairs != 1 || c.Students != 8 {
		t.Fatalf("both start at 09:00: %d pairs, %d students; want 1 and 8", c.Pairs, c.Students)
	}

	// Move the discus to the afternoon and the clash goes.
	do("PATCH", "/api/schedule/block-time", `{"event_id":7,"age_group":"Under 14","time":"14:00"}`)
	if c := clashes(); c.Pairs != 0 {
		t.Errorf("discus moved to 14:00 but %d clashes remain", c.Pairs)
	}

	// How long a heat takes is a setting, and is checked.
	if code, _ := do("PUT", "/api/settings", `{"heat_minutes":0}`); code != 400 {
		t.Errorf("0 minutes per heat accepted: %d", code)
	}
	if code, _ := do("PUT", "/api/settings", `{"heat_minutes":600}`); code != 400 {
		t.Errorf("600 minutes per heat accepted: %d", code)
	}
	if code, _ := do("PUT", "/api/settings", `{"heat_minutes":1.5}`); code != 200 {
		t.Errorf("1.5 minutes per heat refused: %d", code)
	}
	do("PUT", "/api/settings", `{"heat_minutes":60}`)
	if c := clashes(); c.Pairs != 0 {
		t.Errorf("14:00 discus vs a 100m that lasts at most 60 min from 09:00: %d clashes", c.Pairs)
	}
}
