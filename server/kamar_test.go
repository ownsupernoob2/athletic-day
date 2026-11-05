package main

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func kamarServer(t *testing.T) *Server {
	t.Helper()
	db, err := openDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &Server{db: db, limit: newLoginLimiter(), kamarUser: "kamar", kamarPass: "s3cret"}
}

// post sends a body to the KAMAR endpoint and returns the status and decoded reply.
func post(t *testing.T, s *Server, user, pass, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/kamar", strings.NewReader(body))
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func msg(sync, students string) string {
	s := `{"SMSDirectoryData":{"datetime":20260101000000,"sync":"` + sync + `","sms":"KAMAR"`
	if students != "" {
		s += `,"students":{"count":1,"data":[` + students + `]}`
	}
	return s + `}}`
}

func count(t *testing.T, s *Server, where string) (n int) {
	t.Helper()
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM students WHERE ` + where).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return
}

func TestKamarRefusesBadLogin(t *testing.T) {
	s := kamarServer(t)
	for name, c := range map[string][2]string{"none": {"", ""}, "wrong password": {"kamar", "nope"}, "wrong user": {"x", "s3cret"}} {
		code, out := post(t, s, c[0], c[1], msg("check", ""))
		if code != 403 {
			t.Errorf("%s: got %d, want 403", name, code)
		}
		if d, _ := out["SMSDirectoryData"].(map[string]any); d["result"] != "Authentication Failed" || d["error"] != float64(403) {
			t.Errorf("%s: wrong error body %v", name, out)
		}
	}
	// With no login configured everything is refused, even a "matching" empty login.
	s.kamarUser, s.kamarPass = "", ""
	if code, _ := post(t, s, "", "", msg("check", "")); code != 403 {
		t.Errorf("unset login: got %d, want 403", code)
	}
}

func TestKamarCheckReply(t *testing.T) {
	s := kamarServer(t)
	code, out := post(t, s, "kamar", "s3cret", msg("check", ""))
	if code != 200 {
		t.Fatalf("got %d", code)
	}
	d := out["SMSDirectoryData"].(map[string]any)
	if d["error"] != float64(0) || d["result"] != "OK" || d["countryDataStored"] == "" {
		t.Errorf("bad reply %v", d)
	}
	students := d["options"].(map[string]any)["students"].(map[string]any)
	if students["details"] != true || students["photos"] != false || students["timetables"] != false {
		t.Errorf("students options wrong: %v", students)
	}
	if req := students["fields"].(map[string]any)["required"]; req != kamarFields {
		t.Errorf("fields = %v", req)
	}
	if staff := d["options"].(map[string]any)["staff"].(map[string]any); staff["details"] != false {
		t.Errorf("staff should be off: %v", staff)
	}
}

func TestKamarRejectsXML(t *testing.T) {
	s := kamarServer(t)
	code, _ := post(t, s, "kamar", "s3cret", `<?xml version="1.0"?><SMSDirectoryData><sync>check</sync></SMSDirectoryData>`)
	if code != 415 {
		t.Errorf("got %d, want 415", code)
	}
}

func TestKamarStudentSync(t *testing.T) {
	s := kamarServer(t)
	user := func(body string) int { c, _ := post(t, s, "kamar", "s3cret", body); return c }

	// A student staff already added by hand should be linked, not duplicated.
	s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES('Sam','Ng','2011-03-04','SSH',?,?)`, now(), now())

	full := msg("full", `{"id":101,"firstname":"Sam","lastname":"Ng","tutor":"ssh","datebirth":"20110304"},
		{"id":102,"firstname":"Tui","lastname":"Park","tutor":"TDR","datebirth":20120815},
		{"id":103,"firstname":"No","lastname":"Group","tutor":"10X","datebirth":"20120101"},
		{"firstname":"No","lastname":"Id","tutor":"WND","datebirth":"20120101"},
		{"id":105,"firstname":"Too","lastname":"Old","tutor":"HAN","datebirth":"19700101"}`)
	if c := user(full); c != 200 {
		t.Fatalf("full sync: %d", c)
	}
	if n := count(t, s, `kamar_id IS NOT NULL`); n != 2 {
		t.Errorf("linked students = %d, want 2", n)
	}
	if n := count(t, s, `last_name='Ng'`); n != 1 {
		t.Errorf("Sam Ng duplicated: %d", n)
	}
	var group, dob string
	s.db.QueryRow(`SELECT group_code,dob FROM students WHERE kamar_id='102'`).Scan(&group, &dob)
	if group != "TDR" || dob != "2012-08-15" {
		t.Errorf("Tui stored as %s %s", group, dob)
	}

	// Part sync with only the changed field must keep everything else.
	if c := user(msg("part", `{"id":102,"tutor":"WND"}`)); c != 200 {
		t.Fatalf("part sync: %d", c)
	}
	var first string
	s.db.QueryRow(`SELECT first_name,group_code FROM students WHERE kamar_id='102'`).Scan(&first, &group)
	if first != "Tui" || group != "WND" {
		t.Errorf("after part sync: %s %s", first, group)
	}

	// Unknown message types are acknowledged, not errors.
	if c := user(msg("studenttimetables", "")); c != 200 {
		t.Errorf("unknown sync: %d", c)
	}

	// Status reflects the last sync.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/kamar/status", nil)
	s.handleKamarStatus(rec, req)
	if !strings.Contains(rec.Body.String(), `"linked":2`) {
		t.Errorf("status: %s", rec.Body.String())
	}

	// Delete notice removes KAMAR students but not hand-added ones.
	s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES('Hand','Added','2011-01-01','TLK',?,?)`, now(), now())
	if c := user(msg("deletenotice", "")); c != 200 {
		t.Fatalf("deletenotice: %d", c)
	}
	if n := count(t, s, `kamar_id IS NOT NULL`); n != 0 {
		t.Errorf("kamar students left: %d", n)
	}
	if n := count(t, s, `last_name='Added'`); n != 1 {
		t.Errorf("hand-added student was deleted")
	}
}
