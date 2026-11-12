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
