package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// Demo school: 31 tutor groups and 831 students, 24 to 29 per group.
// These groups belong to real test accounts, so they get students but no staff
// account and no events picked: the testers fill them in themselves.
var demoTestGroups = map[string]bool{"SSH": true, "HTR": true, "SWM": true, "WST": true}

var demoGroups = []string{
	// Stuart
	"SSH", "SWM", "SHV", "STW", "SKR", "SJP", "STE", "SKN",
	// Tudor
	"TDR", "TLK", "TMN", "TNQ", "TVN", "TBZ", "TTT", "TRN",
	// Windsor
	"WST", "WND", "WBR", "WSL", "WRD", "WHL", "WCK", "WPR",
	// Hanover
	"HTR", "HAN", "HKT", "HLM", "HPN", "HMD", "HWL",
}

const demoStudentTotal = 831

// demoSizes returns one size (24..29) per group that adds up to exactly `total`.
func demoSizes(rng *rand.Rand, groups, total int) []int {
	sizes := make([]int, groups)
	sum := 0
	for i := range sizes {
		sizes[i] = 24 + rng.IntN(6)
		sum += sizes[i]
	}
	for sum != total {
		i := rng.IntN(groups)
		if sum > total && sizes[i] > 24 {
			sizes[i]--
			sum--
		} else if sum < total && sizes[i] < 29 {
			sizes[i]++
			sum++
		}
	}
	return sizes
}

// demoPicks chooses a believable set of events for one student. The real rules:
// at least 3 events, at most 5, and never more than 3 track or 3 field. Most students do exactly 3.
func demoPicks(rng *rand.Rand, events []Event) []int64 {
	var track, field []Event
	for _, e := range events {
		if e.Kind == "track" {
			track = append(track, e)
		} else {
			field = append(field, e)
		}
	}
	// (track, field) counts; repeated pairs are more likely.
	var pairs [][2]int
	switch r := rng.IntN(100); {
	case r < 72: // exactly 3
		pairs = [][2]int{{3, 0}, {2, 1}, {2, 1}, {1, 2}, {1, 2}, {0, 3}}
	case r < 90: // 4
		pairs = [][2]int{{3, 1}, {2, 2}, {2, 2}, {1, 3}}
	default: // 5: 3 of one type and 2 of the other
		pairs = [][2]int{{3, 2}, {2, 3}}
	}
	p := pairs[rng.IntN(len(pairs))]
	var ids []int64
	for _, k := range rng.Perm(len(track))[:p[0]] {
		ids = append(ids, track[k].ID)
	}
	for _, k := range rng.Perm(len(field))[:p[1]] {
		ids = append(ids, field[k].ID)
	}
	return ids
}

// rerollDemoEntries gives every student of a dummy staff group (staff.<group>@school.test) a fresh
// set of events under the current rules, leaving real accounts and their groups alone.
// Run with: athletic reroll-demo-entries
func (s *Server) rerollDemoEntries() (int, error) {
	events, err := s.listEvents(true)
	if err != nil {
		return 0, err
	}
	rows, err := s.db.Query(`SELECT id FROM students WHERE group_code IN
		(SELECT group_code FROM users WHERE role='staff' AND email LIKE 'staff.%@school.test')`)
	if err != nil {
		return 0, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	rng := rand.New(rand.NewPCG(5, 17))
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM entries WHERE student_id=?`, id); err != nil {
			return 0, err
		}
		for _, eid := range demoPicks(rng, events) {
			if _, err := tx.Exec(`INSERT INTO entries(student_id,event_id) VALUES(?,?)`, id, eid); err != nil {
				return 0, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	touchEntries(s.db)
	return len(ids), nil
}

// seedDemo fills an empty database with fake students and staff so the app can be
// tried out. Enabled with SEED_DEMO=1.
func (s *Server) seedDemo() error {
	first := strings.Fields("James Amir Juan Hiroshi Mohammed Chen Raj Ahmed Lucas Rafael Oliver Leo Ethan Mason Jack Daniel Aiden Noah Liam Jacob William Henry Theo Archie Max Finn Oscar Logan Caleb Isaac Samuel Joshua Nathan Hugo Felix Jude Rowan Kai Matiu Tane Rawiri Manu Wiremu Hemi Ihaka Nikau Zane Jay Tom Ben Sam Dylan Ryan Cooper Blake Jordan Harry George Alex Eli Luca Marco Diego Sione Tevita Malakai Aroha Ezra Arlo Ivan Leon")
	last := strings.Fields("O'Connor Khan Garcia Tanaka Al-Farsi Li Patel Al-Mansoor Pereira Silva Smith Jones Taylor Brown Wilson Evans Thomas Roberts Walker Wright Green Hall Wood Clarke Jackson Lewis Young King Scott Adams Baker Nguyen Kim Singh Murphy Kelly Campbell Walker Ngata Parata Tamihana Wiremu Henare Pene Ropata Tane Hohepa Fa'asavalu Tuilagi Faleolo Taufa Moana Lee Chen Wang Zhang Liu Park Cho Sato Suzuki Ito Fraser Stewart Reid MacDonald Ross Grant Burns Cameron Morrison Hughes Cook Bennett Foster Gray Hunter")
	staffNames := strings.Fields("Adams Brooks Carter Dixon Ellis Foster Grant Hughes Irwin Jenkins Knight Lowe Mason Nash Owens Price Quinn Reed Shaw Todd Upton Vance Webb Xavier Young Zhang Archer Boyd Cole")
	events, err := s.listEvents(true)
	if err != nil {
		return err
	}

	rng := rand.New(rand.NewPCG(11, 831))
	sizes := demoSizes(rng, len(demoGroups), demoStudentTotal)
	today := time.Now()
	staffIdx, students, filled := 0, 0, 0

	for gi, g := range demoGroups {
		if !demoTestGroups[g] {
			name := fmt.Sprintf("%s. %s", []string{"Mr", "Ms", "Mrs"}[rng.IntN(3)], staffNames[staffIdx%len(staffNames)])
			staffIdx++
			if _, err := s.createUser(name, "staff."+strings.ToLower(g)+"@school.test", "staffpass123", roleStaff, g); err != nil {
				return err
			}
		}
		age := 13 + gi%5 // groups are year levels 9 to 13
		for i := 0; i < sizes[gi]; i++ {
			dob := today.AddDate(-age, -rng.IntN(12), -rng.IntN(28))
			res, err := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
				first[rng.IntN(len(first))], last[rng.IntN(len(last))], dob.Format("2006-01-02"), g, now(), now())
			if err != nil {
				return err
			}
			students++
			if demoTestGroups[g] {
				continue // left for the testers to fill in
			}
			id, _ := res.LastInsertId()
			for _, eid := range demoPicks(rng, events) {
				s.db.Exec(`INSERT OR IGNORE INTO entries(student_id,event_id) VALUES(?,?)`, id, eid)
			}
			filled++
		}
	}
	touchEntries(s.db)
	s.logEvent("system", "server", fmt.Sprintf("demo data loaded: %d students in %d groups", students, len(demoGroups)))
	fmt.Printf("Demo data loaded: %d students in %d groups (%d filled in, %d left empty for testers). Dummy staff: staff.<group>@school.test, password staffpass123\n",
		students, len(demoGroups), filled, students-filled)
	return nil
}
