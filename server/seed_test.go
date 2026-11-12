package main

import (
	"path/filepath"
	"testing"
)

func TestDemoSeed(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, limit: newLoginLimiter()}
	if err := s.bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := s.seedDemo(); err != nil {
		t.Fatal(err)
	}
	groups, err := s.groupStatuses()
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, g := range groups {
		total += g.Students
		if g.Students < 24 || g.Students > 29 {
			t.Errorf("group %s has %d students", g.Code, g.Students)
		}
		if demoTestGroups[g.Code] {
			if g.Started != 0 || len(g.Staff) != 0 {
				t.Errorf("test group %s should be empty and have no dummy staff: %+v", g.Code, g)
			}
		} else if g.Complete != g.Students || len(g.Staff) != 1 {
			t.Errorf("dummy group %s should be fully filled in with one staff: %+v", g.Code, g)
		}
	}
	if total != demoStudentTotal || len(groups) != 31 {
		t.Errorf("got %d students in %d groups", total, len(groups))
	}
	checkPickRules(t, s, 600)
}

// checkPickRules: 3 to 5 events each, never more than 3 track or 3 field, and mostly exactly 3.
func checkPickRules(t *testing.T, s *Server, minStudents int) {
	t.Helper()
	rows, err := s.db.Query(`SELECT e.student_id,
		SUM(v.kind='track'), SUM(v.kind='field') FROM entries e JOIN events v ON v.id=e.event_id GROUP BY e.student_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n, threes := 0, 0
	for rows.Next() {
		var id int64
		var track, field int
		rows.Scan(&id, &track, &field)
		n++
		if total := track + field; total < 3 || total > 5 {
			t.Errorf("student %d has %d events", id, total)
		} else if total == 3 {
			threes++
		}
		if track > 3 || field > 3 {
			t.Errorf("student %d has %d track and %d field", id, track, field)
		}
	}
	if n < minStudents {
		t.Fatalf("only %d students have events", n)
	}
	if share := float64(threes) / float64(n); share < 0.6 || share > 0.85 {
		t.Errorf("%.0f%% of students do exactly 3 events, want most (60-85%%)", share*100)
	}
}

func TestRerollKeepsRules(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, limit: newLoginLimiter()}
	if err := s.bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := s.seedDemo(); err != nil {
		t.Fatal(err)
	}
	// A real account's group, with a hand-made pick, must be left alone.
	s.db.Exec(`INSERT INTO users(name,email,password_hash,role,group_code,created_at) VALUES('Real','real@kingshigh.school.nz','x','staff','SSH',?)`, now())
	var sid int64
	s.db.QueryRow(`SELECT id FROM students WHERE group_code='SSH' LIMIT 1`).Scan(&sid)
	s.db.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,1)`, sid)

	n, err := s.rerollDemoEntries()
	if err != nil || n != demoStudentTotal-104 {
		t.Fatalf("rerolled %d students (err %v), want %d", n, err, demoStudentTotal-104)
	}
	var picks int
	s.db.QueryRow(`SELECT COUNT(*) FROM entries WHERE student_id=?`, sid).Scan(&picks)
	if picks != 1 {
		t.Errorf("a real account's student was changed: %d picks", picks)
	}
	s.db.Exec(`DELETE FROM entries WHERE student_id=?`, sid)
	checkPickRules(t, s, 600)
}
