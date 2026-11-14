package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// organiserClient signs in as the first organiser and returns a request helper.
func organiserClient(t *testing.T) (*Server, func(method, path, body string) (int, []byte)) {
	t.Helper()
	db, err := openDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := &Server{db: db, limit: newLoginLimiter()}
	t.Setenv("ADMIN_PASSWORD", "test-password-123")
	if err := s.bootstrap(); err != nil {
		t.Fatal(err)
	}
	h := s.routes()
	do := func(method, path, body string) (int, []byte) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if path == "/api/auth/login" {
			cookies = rec.Result().Cookies()
		}
		return rec.Code, rec.Body.Bytes()
	}
	cookies = nil
	if code, body := do("POST", "/api/auth/login", `{"email":"organiser@school.test","password":"test-password-123"}`); code != 200 {
		t.Fatalf("login: %d %s", code, body)
	}
	return s, do
}

var cookies []*http.Cookie

func TestTimesBelongToEventAndAgeGroup(t *testing.T) {
	s, do := organiserClient(t)

	// 14 runners aged 13 and 9 aged 14, all in 100m and 200m: several heats of each (heat size 6).
	for i := 0; i < 23; i++ {
		age := 13
		if i >= 14 {
			age = 14
		}
		res, _ := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
			fmt.Sprintf("S%d", i), fmt.Sprintf("L%02d", i), fmt.Sprintf("%d-01-15", 2026-age-1), "SSH", now(), now())
		id, _ := res.LastInsertId()
		s.db.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,1),(?,2)`, id, id)
	}
	if code, body := do("POST", "/api/schedule/generate", `{"randomize":false}`); code != 200 {
		t.Fatalf("generate: %d %s", code, body)
	}
	if code, body := do("POST", "/api/schedule/times", `{"start":"09:00","end":"12:00"}`); code != 200 {
		t.Fatalf("times: %d %s", code, body)
	}

	type sched struct {
		Heats []Heat `json:"heats"`
	}
	load := func() []Heat {
		code, body := do("GET", "/api/schedule", "")
		var d sched
		if code != 200 || json.Unmarshal(body, &d) != nil {
			t.Fatalf("schedule: %d %s", code, body)
		}
		return d.Heats
	}
	heats := load()
	if len(heats) < 6 {
		t.Fatalf("expected several heats, got %d", len(heats))
	}

	// Every heat in one event + age group shares one time, and times only move forward through the day.
	blockTime := map[string]string{}
	var order []string
	prev := -1
	for _, h := range heats {
		key := fmt.Sprintf("%d|%s", h.EventID, h.AgeGroup)
		if h.Time == "" {
			t.Fatalf("heat in %s has no time", key)
		}
		if old, ok := blockTime[key]; ok && old != h.Time {
			t.Errorf("%s has two times: %s and %s", key, old, h.Time)
		}
		if _, ok := blockTime[key]; !ok {
			order = append(order, key)
			m, _ := parseClock(h.Time)
			if m < prev {
				t.Errorf("time went backwards at %s (%s)", key, h.Time)
			}
			prev = m
		}
		blockTime[key] = h.Time
		if m, _ := parseClock(h.Time); m < 9*60 || m >= 12*60 {
			t.Errorf("%s at %s is outside 09:00-12:00", key, h.Time)
		}
	}
	if blockTime[order[0]] != "09:00" {
		t.Errorf("first block at %s, want 09:00", blockTime[order[0]])
	}
	if len(blockTime) != 4 {
		t.Errorf("expected 4 blocks (2 events x 2 ages), got %d", len(blockTime))
	}

	// Edit one block by hand.
	first := heats[0]
	body := fmt.Sprintf(`{"event_id":%d,"age_group":%q,"time":"10:45"}`, first.EventID, first.AgeGroup)
	if code, out := do("PATCH", "/api/schedule/block-time", body); code != 200 {
		t.Fatalf("set block time: %d %s", code, out)
	}
	for _, h := range load() {
		if h.EventID == first.EventID && h.AgeGroup == first.AgeGroup && h.Time != "10:45" {
			t.Errorf("heat %d still at %s after edit", h.ID, h.Time)
		}
	}
	if code, _ := do("PATCH", "/api/schedule/block-time", fmt.Sprintf(`{"event_id":%d,"age_group":%q,"time":"9am"}`, first.EventID, first.AgeGroup)); code != 400 {
		t.Errorf("bad time accepted: %d", code)
	}
	if code, _ := do("PATCH", "/api/schedule/block-time", `{"event_id":999,"age_group":"Under 14","time":"09:00"}`); code != 404 {
		t.Errorf("unknown block: %d", code)
	}

	// Times survive a re-shuffle of that event, and clearing removes them all.
	do("POST", "/api/schedule/generate", fmt.Sprintf(`{"randomize":true,"event_id":%d}`, first.EventID))
	for _, h := range load() {
		if h.EventID == first.EventID && h.AgeGroup == first.AgeGroup && h.Time != "10:45" {
			t.Errorf("time lost after reshuffle: %s", h.Time)
		}
	}
	do("POST", "/api/schedule/times", `{"clear":true}`)
	for _, h := range load() {
		if h.Time != "" {
			t.Fatalf("time left after clear: %s", h.Time)
		}
	}
}

func TestOldHeatTimeColumnIsDropped(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`SELECT time FROM heats`); err == nil {
		t.Error("heats still has a time column")
	}
}

func TestTrackAndFieldRunTogether(t *testing.T) {
	s, do := organiserClient(t)

	// Ages are measured on the event date, so fix it. 100m (track, id 1) and Discus (field, id 7):
	// 8 students aged 13 (Under 14) and 8 aged 14 (Under 15), half doing only the 100m and half only
	// the discus, so track and field have no student in common and can start together.
	do("PUT", "/api/settings", `{"event_date":"2026-06-01"}`)
	for i := 0; i < 16; i++ {
		age := 13
		if i >= 8 {
			age = 14
		}
		res, _ := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
			fmt.Sprintf("S%d", i), fmt.Sprintf("L%02d", i), fmt.Sprintf("%d-12-31", 2026-age-1), "SSH", now(), now())
		id, _ := res.LastInsertId()
		event := 1
		if i%2 == 1 {
			event = 7
		}
		s.db.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,?)`, id, event)
	}
	do("POST", "/api/schedule/generate", `{"randomize":false}`)
	if code, out := do("POST", "/api/schedule/times", `{"start":"09:00","end":"12:00"}`); code != 200 {
		t.Fatalf("times: %d %s", code, out)
	}
	load := func() map[string]Heat { // first heat of each event + age group
		code, body := do("GET", "/api/schedule", "")
		var d struct {
			Heats []Heat `json:"heats"`
		}
		if code != 200 || json.Unmarshal(body, &d) != nil {
			t.Fatalf("schedule: %d %s", code, body)
		}
		out := map[string]Heat{}
		for _, h := range d.Heats {
			k := fmt.Sprintf("%s|%s", h.Kind, h.AgeGroup)
			if _, ok := out[k]; !ok {
				out[k] = h
			}
		}
		return out
	}

	// Both timelines start at 09:00: the first field block does not wait for the track to finish.
	b := load()
	if b["track|Under 14"].Time != "09:00" || b["field|Under 14"].Time != "09:00" {
		t.Errorf("track starts %s and field starts %s, both should start at 09:00", b["track|Under 14"].Time, b["field|Under 14"].Time)
	}
	for k, h := range b {
		if m, _ := parseClock(h.Time); m < 9*60 || m >= 12*60 {
			t.Errorf("%s at %s is outside 09:00-12:00", k, h.Time)
		}
	}

	// A field time typed in by hand is fixed: automatic times keep it and flag it.
	field := b["field|Under 14"]
	patch := fmt.Sprintf(`{"event_id":%d,"age_group":%q,"time":"10:30"}`, field.EventID, field.AgeGroup)
	if code, out := do("PATCH", "/api/schedule/block-time", patch); code != 200 {
		t.Fatalf("set fixed: %d %s", code, out)
	}
	do("POST", "/api/schedule/times", `{"start":"09:00","end":"12:00"}`)
	b = load()
	if h := b["field|Under 14"]; h.Time != "10:30" || !h.TimeFixed {
		t.Errorf("fixed field time changed or lost its flag: %s fixed=%v", h.Time, h.TimeFixed)
	}
	if h := b["field|Under 15"]; h.TimeFixed || h.Time != "09:00" {
		t.Errorf("the other field block should be spread from 09:00, got %s fixed=%v", h.Time, h.TimeFixed)
	}
	if h := b["track|Under 14"]; h.TimeFixed || h.Time != "09:00" {
		t.Errorf("track should be untouched by the field time: %s fixed=%v", h.Time, h.TimeFixed)
	}

	// Clearing removes every time, fixed ones too.
	do("POST", "/api/schedule/times", `{"clear":true}`)
	for k, h := range load() {
		if h.Time != "" || h.TimeFixed {
			t.Errorf("%s still has a time after clearing: %s", k, h.Time)
		}
	}
}
