package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// seedDemo fills an empty database with fake students, staff and a few entries
// so the app can be tried out. Enabled with SEED_DEMO=1.
func (s *Server) seedDemo() error {
	first := strings.Fields("James Amir Juan Hiroshi Mohammed Chen Raj Ahmed Lucas Rafael Olivia Amelia Isla Ava Mia Noah Oliver Leo Ella Grace Freya Ivy Zara Ethan Mason Jack Ruby Lily Sophie Daniel Aiden Priya Mei Yuki Sara Omar Hana Tom Ben Chloe")
	last := strings.Fields("O'Connor Khan Garcia Tanaka Al-Farsi Li Patel Al-Mansoor Pereira Silva Smith Jones Taylor Brown Wilson Evans Thomas Roberts Walker Wright Green Hall Wood Clarke Jackson Lewis Young King Scott Adams Baker Nguyen Kim Singh Murphy Kelly Campbell")
	groups := []string{"SSH", "SHV", "STW", "TDR", "TLK", "TMN", "WND", "WBR", "WSL", "HAN", "HKT", "HLM"}
	staffNames := strings.Fields("Mr.Adams Ms.Brooks Mr.Carter Ms.Dixon Mr.Ellis Ms.Foster Mr.Grant Ms.Hughes Mr.Irwin Ms.Jenkins Mr.Knight Ms.Lowe")
	events, err := s.listEvents(true)
	if err != nil {
		return err
	}
	rng := rand.New(rand.NewPCG(7, 42))
	today := time.Now()
	for gi, g := range groups {
		email := "staff." + strings.ToLower(g) + "@school.test"
		if _, err := s.createUser(strings.ReplaceAll(staffNames[gi], ".", ". "), email, "staffpass123", roleStaff, g); err != nil {
			return err
		}
		for i := 0; i < 8+rng.IntN(5); i++ {
			age := 12 + rng.IntN(7)
			dob := today.AddDate(-age, -rng.IntN(12), -rng.IntN(28))
			res, err := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
				first[rng.IntN(len(first))], last[rng.IntN(len(last))], dob.Format("2006-01-02"), g, now(), now())
			if err != nil {
				return err
			}
			// Half the groups have been filled in already.
			if gi%2 == 0 {
				id, _ := res.LastInsertId()
				for _, k := range rng.Perm(len(events))[:3+rng.IntN(2)] {
					s.db.Exec(`INSERT OR IGNORE INTO entries(student_id,event_id) VALUES(?,?)`, id, events[k].ID)
				}
			}
		}
	}
	touchEntries(s.db)
	fmt.Println("Demo data loaded. Staff logins: staff.ssh@school.test ... password staffpass123")
	return nil
}
