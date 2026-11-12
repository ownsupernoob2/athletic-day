package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// loginAs returns a request helper with its own cookie jar.
func loginAs(t *testing.T, s *Server, email, password string) func(method, path, body string) (int, string) {
	t.Helper()
	h := s.routes()
	var jar []*http.Cookie
	do := func(method, path, body string) (int, string) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for _, c := range jar {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if path == "/api/auth/login" {
			jar = rec.Result().Cookies()
		}
		return rec.Code, rec.Body.String()
	}
	if code, out := do("POST", "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)); code != 200 {
		t.Fatalf("login %s: %d %s", email, code, out)
	}
	return do
}

func TestEventLimits(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	t.Setenv("ADMIN_PASSWORD", "organiser-pass-1")
	s := &Server{db: db, limit: newLoginLimiter()}
	if err := s.bootstrap(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.createUser("Staff", "staff@kingshigh.school.nz", "staff-pass-123", roleStaff, "SSH"); err != nil {
		t.Fatal(err)
	}
	res, _ := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES('Sam','Ng','2012-03-04','SSH',?,?)`, now(), now())
	sid, _ := res.LastInsertId()

	// Seeded events: ids 1-6 and 13-14 are track, 7-12 and 15-16 are field.
	tick := func(do func(string, string, string) (int, string), eid int, on bool) int {
		code, _ := do("PUT", fmt.Sprintf("/api/students/%d/events/%d", sid, eid), fmt.Sprintf(`{"on":%v}`, on))
		return code
	}
	staff := loginAs(t, s, "staff@kingshigh.school.nz", "staff-pass-123")

	// 3 track is fine, a 4th track is not.
	for _, e := range []int{1, 2, 3} {
		if c := tick(staff, e, true); c != 200 {
			t.Fatalf("track %d: %d", e, c)
		}
	}
	code, body := do2(staff, sid, 4)
	if code != 409 || !strings.Contains(body, "3 track") {
		t.Errorf("4th track: %d %s", code, body)
	}
	// Ticking something already ticked is not "one more".
	if c := tick(staff, 1, true); c != 200 {
		t.Errorf("re-tick: %d", c)
	}
	// 2 field makes 5 in total, which is the most.
	for _, e := range []int{7, 8} {
		if c := tick(staff, e, true); c != 200 {
			t.Fatalf("field %d: %d", e, c)
		}
	}
	code, body = do2(staff, sid, 9)
	if code != 409 || !strings.Contains(body, "5 events") {
		t.Errorf("6th event: %d %s", code, body)
	}
	// Taking one off is always allowed, and then another fits.
	if c := tick(staff, 1, false); c != 200 {
		t.Errorf("untick: %d", c)
	}
	if c := tick(staff, 9, true); c != 200 {
		t.Errorf("field after untick: %d", c)
	}
	if n := count(t, s, `1=1`); n != 1 {
		t.Fatalf("unexpected students: %d", n)
	}

	// The organiser can make an exception.
	org := loginAs(t, s, "organiser@school.test", "organiser-pass-1")
	if c := tick(org, 4, true); c != 200 {
		t.Errorf("organiser override: %d", c)
	}

	// Limits are settings: raising the per-type limit lets staff add a 4th track event after room is made.
	if c, out := org("PUT", "/api/settings", `{"max_events":6,"max_per_type":4}`); c != 200 || !strings.Contains(out, `"max_per_type":4`) {
		t.Fatalf("settings: %d %s", c, out)
	}
	if c, _ := org("PUT", "/api/settings", `{"max_events":2}`); c != 400 {
		t.Errorf("max below min accepted: %d", c)
	}
	if c, _ := org("PUT", "/api/settings", `{"max_per_type":0}`); c != 400 {
		t.Errorf("zero limit accepted: %d", c)
	}
}

func do2(do func(string, string, string) (int, string), sid int64, eid int) (int, string) {
	return do("PUT", fmt.Sprintf("/api/students/%d/events/%d", sid, eid), `{"on":true}`)
}
