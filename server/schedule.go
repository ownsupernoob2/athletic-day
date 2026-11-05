package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type HeatStudent struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	GroupCode string `json:"group_code"`
	House     string `json:"house"`
	AgeGroup  string `json:"age_group"`
}

type Heat struct {
	ID        int64         `json:"id"`
	EventID   int64         `json:"event_id"`
	EventCode string        `json:"event_code"`
	EventName string        `json:"event_name"`
	Kind      string        `json:"kind"`
	AgeGroup  string        `json:"age_group"`
	Number    int           `json:"number"`
	Time      string        `json:"time"`
	Students  []HeatStudent `json:"students"`
}

func (s *Server) loadHeats() ([]Heat, error) {
	rows, err := s.db.Query(`SELECT h.id,h.event_id,e.code,e.name,e.kind,h.age_group,h.number,h.time,e.position
		FROM heats h JOIN events e ON e.id=h.event_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	heats := []Heat{}
	pos := map[int64]int{}
	byID := map[int64]int{}
	for rows.Next() {
		var h Heat
		var p int
		if err := rows.Scan(&h.ID, &h.EventID, &h.EventCode, &h.EventName, &h.Kind, &h.AgeGroup, &h.Number, &h.Time, &p); err != nil {
			return nil, err
		}
		h.Students = []HeatStudent{}
		pos[h.ID] = p
		heats = append(heats, h)
	}
	rows.Close()
	sort.Slice(heats, func(i, j int) bool {
		a, b := heats[i], heats[j]
		if pos[a.ID] != pos[b.ID] {
			return pos[a.ID] < pos[b.ID]
		}
		if x, y := ageGroupIndex(a.AgeGroup), ageGroupIndex(b.AgeGroup); x != y {
			return x < y
		}
		return a.Number < b.Number
	})
	for i, h := range heats {
		byID[h.ID] = i
	}

	ref := refDate(s.db)
	sr, err := s.db.Query(`SELECT he.heat_id,s.id,s.first_name,s.last_name,s.group_code,s.dob
		FROM heat_entries he JOIN students s ON s.id=he.student_id ORDER BY he.heat_id,he.position`)
	if err != nil {
		return nil, err
	}
	defer sr.Close()
	for sr.Next() {
		var hid int64
		var st HeatStudent
		var dob string
		if err := sr.Scan(&hid, &st.ID, &st.FirstName, &st.LastName, &st.GroupCode, &dob); err != nil {
			return nil, err
		}
		st.House = houseOf(st.GroupCode)
		st.AgeGroup = ageGroupOf(ageOn(dob, ref))
		if i, ok := byID[hid]; ok {
			heats[i].Students = append(heats[i].Students, st)
		}
	}
	return heats, sr.Err()
}

// ---------- organiser: view & generate ----------

type Unplaced struct {
	EventID  int64       `json:"event_id"`
	AgeGroup string      `json:"age_group"`
	Student  HeatStudent `json:"student"`
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	heats, err := s.loadHeats()
	if err != nil {
		serverErr(w, err)
		return
	}
	// Entered but not placed in any heat of that event.
	placed := map[[2]int64]bool{}
	for _, h := range heats {
		for _, st := range h.Students {
			placed[[2]int64{st.ID, h.EventID}] = true
		}
	}
	students, err := s.loadStudents("", 0)
	if err != nil {
		serverErr(w, err)
		return
	}
	unplaced := []Unplaced{}
	if len(heats) > 0 {
		for _, st := range students {
			for _, eid := range st.Events {
				if !placed[[2]int64{st.ID, eid}] {
					unplaced = append(unplaced, Unplaced{eid, st.AgeGroup, HeatStudent{st.ID, st.FirstName, st.LastName, st.GroupCode, st.House, st.AgeGroup}})
				}
			}
		}
	}
	writeJSON(w, 200, map[string]any{"settings": s.settings(), "heats": heats, "unplaced": unplaced})
}

// balanced splits n items into the fewest heats of at most `size`, as evenly as possible
// (so 7 people become 4+3, never 6+1).
func balanced(n, size int) []int {
	if n == 0 {
		return nil
	}
	k := (n + size - 1) / size
	out := make([]int, k)
	for i := range out {
		out[i] = n / k
		if i < n%k {
			out[i]++
		}
	}
	return out
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Randomize bool  `json:"randomize"`
		EventID   int64 `json:"event_id"`
	}
	if r.ContentLength != 0 && !readJSON(w, r, &in) {
		return
	}
	size, _ := strconv.Atoi(getSetting(s.db, "heat_size", "6"))
	if size < 1 {
		size = 6
	}
	events, err := s.listEvents(true)
	if err != nil {
		serverErr(w, err)
		return
	}
	students, err := s.loadStudents("", 0)
	if err != nil {
		serverErr(w, err)
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		serverErr(w, err)
		return
	}
	defer tx.Rollback()
	if in.EventID != 0 {
		_, err = tx.Exec(`DELETE FROM heats WHERE event_id=?`, in.EventID)
	} else {
		_, err = tx.Exec(`DELETE FROM heats`)
	}
	if err != nil {
		serverErr(w, err)
		return
	}

	for _, ev := range events {
		if in.EventID != 0 && ev.ID != in.EventID {
			continue
		}
		byAge := map[string][]Student{}
		for _, st := range students {
			for _, eid := range st.Events {
				if eid == ev.ID {
					byAge[st.AgeGroup] = append(byAge[st.AgeGroup], st)
				}
			}
		}
		for _, age := range ageGroups {
			list := byAge[age]
			if in.Randomize {
				rand.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
			} else {
				sort.SliceStable(list, func(i, j int) bool {
					return strings.ToLower(list[i].LastName+list[i].FirstName) < strings.ToLower(list[j].LastName+list[j].FirstName)
				})
			}
			at := 0
			for n, count := range balanced(len(list), size) {
				res, err := tx.Exec(`INSERT INTO heats(event_id,age_group,number) VALUES(?,?,?)`, ev.ID, age, n+1)
				if err != nil {
					serverErr(w, err)
					return
				}
				hid, _ := res.LastInsertId()
				for p, st := range list[at : at+count] {
					if _, err := tx.Exec(`INSERT INTO heat_entries(heat_id,student_id,position) VALUES(?,?,?)`, hid, st.ID, p+1); err != nil {
						serverErr(w, err)
						return
					}
				}
				at += count
			}
		}
	}
	if err := tx.Commit(); err != nil {
		serverErr(w, err)
		return
	}
	setSetting(s.db, "generated_at", now())
	s.handleGetSchedule(w, r)
}

func (s *Server) handleClearSchedule(w http.ResponseWriter, r *http.Request) {
	s.db.Exec(`DELETE FROM heats`)
	setSetting(s.db, "published", "0")
	setSetting(s.db, "generated_at", "")
	s.handleGetSchedule(w, r)
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Published bool `json:"published"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Published {
		var n int
		s.db.QueryRow(`SELECT COUNT(*) FROM heats`).Scan(&n)
		if n == 0 {
			fail(w, 400, "Create the schedule before publishing it")
			return
		}
	}
	v := "0"
	if in.Published {
		v = "1"
	}
	setSetting(s.db, "published", v)
	writeJSON(w, 200, s.settings())
}

// ---------- organiser: manual editing ----------

func (s *Server) handleCreateHeat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EventID  int64  `json:"event_id"`
		AgeGroup string `json:"age_group"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if ageGroupIndex(in.AgeGroup) >= len(ageGroups) {
		fail(w, 400, "Unknown age group")
		return
	}
	var n int
	s.db.QueryRow(`SELECT COALESCE(MAX(number),0)+1 FROM heats WHERE event_id=? AND age_group=?`, in.EventID, in.AgeGroup).Scan(&n)
	if _, err := s.db.Exec(`INSERT INTO heats(event_id,age_group,number) VALUES(?,?,?)`, in.EventID, in.AgeGroup, n); err != nil {
		fail(w, 400, "Unknown event")
		return
	}
	s.handleGetSchedule(w, r)
}

func (s *Server) handleUpdateHeat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Time string `json:"time"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	t := strings.TrimSpace(in.Time)
	if t != "" {
		m, ok := parseClock(t)
		if !ok {
			fail(w, 400, "Time must look like 09:30")
			return
		}
		t = fmtClock(m)
	}
	s.db.Exec(`UPDATE heats SET time=? WHERE id=?`, t, pathID(r, "id"))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) renumber(eventID int64, age string) {
	rows, err := s.db.Query(`SELECT id FROM heats WHERE event_id=? AND age_group=? ORDER BY number,id`, eventID, age)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for i, id := range ids {
		s.db.Exec(`UPDATE heats SET number=? WHERE id=?`, i+1, id)
	}
}

func (s *Server) handleDeleteHeat(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var eid int64
	var age string
	if err := s.db.QueryRow(`SELECT event_id,age_group FROM heats WHERE id=?`, id).Scan(&eid, &age); err == nil {
		s.db.Exec(`DELETE FROM heats WHERE id=?`, id)
		s.renumber(eid, age)
	}
	s.handleGetSchedule(w, r)
}

// handleHeatAddStudent places a student in a heat, moving them out of any other heat of that event.
func (s *Server) handleHeatAddStudent(w http.ResponseWriter, r *http.Request) {
	hid := pathID(r, "id")
	var in struct {
		StudentID int64 `json:"student_id"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	var eid int64
	if err := s.db.QueryRow(`SELECT event_id FROM heats WHERE id=?`, hid).Scan(&eid); err != nil {
		fail(w, 404, "Heat not found")
		return
	}
	if err := s.db.QueryRow(`SELECT id FROM students WHERE id=?`, in.StudentID).Scan(&in.StudentID); err != nil {
		fail(w, 404, "Student not found")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		serverErr(w, err)
		return
	}
	defer tx.Rollback()
	steps := []struct {
		q    string
		args []any
	}{
		{`DELETE FROM heat_entries WHERE student_id=? AND heat_id IN (SELECT id FROM heats WHERE event_id=?)`, []any{in.StudentID, eid}},
		{`INSERT OR IGNORE INTO entries(student_id,event_id) VALUES(?,?)`, []any{in.StudentID, eid}},
		{`INSERT INTO heat_entries(heat_id,student_id,position) VALUES(?,?,(SELECT COALESCE(MAX(position),0)+1 FROM heat_entries WHERE heat_id=?))`, []any{hid, in.StudentID, hid}},
	}
	for _, st := range steps {
		if _, err := tx.Exec(st.q, st.args...); err != nil {
			serverErr(w, errors.New("place student: "+err.Error()))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		serverErr(w, err)
		return
	}
	s.handleGetSchedule(w, r)
}

func (s *Server) handleHeatRemoveStudent(w http.ResponseWriter, r *http.Request) {
	hid, sid := pathID(r, "id"), pathID(r, "sid")
	s.db.Exec(`DELETE FROM heat_entries WHERE heat_id=? AND student_id=?`, hid, sid)
	rows, err := s.db.Query(`SELECT student_id FROM heat_entries WHERE heat_id=? ORDER BY position`, hid)
	if err == nil {
		var ids []int64
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
		for i, id := range ids {
			s.db.Exec(`UPDATE heat_entries SET position=? WHERE heat_id=? AND student_id=?`, i+1, hid, id)
		}
	}
	s.handleGetSchedule(w, r)
}

// ---------- public ----------

func (s *Server) handlePublicSchedule(w http.ResponseWriter, r *http.Request) {
	st := s.settings()
	out := map[string]any{
		"published":  st["published"],
		"title":      st["title"],
		"event_date": st["event_date"],
		"heats":      []Heat{},
	}
	if st["published"] == true {
		heats, err := s.loadHeats()
		if err != nil {
			serverErr(w, err)
			return
		}
		out["heats"] = heats
	}
	writeJSON(w, 200, out)
}

func (s *Server) handlePublicScheduleCSV(w http.ResponseWriter, r *http.Request) {
	if getSetting(s.db, "published", "0") != "1" {
		fail(w, 404, "The schedule hasn't been published yet")
		return
	}
	heats, err := s.loadHeats()
	if err != nil {
		serverErr(w, err)
		return
	}
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	cw := csv.NewWriter(&buf)
	cw.Write([]string{"Event", "Age Group", "Heat", "Time", "Lane", "Last Name", "First Name", "Group", "House"})
	for _, h := range heats {
		for i, st := range h.Students {
			cw.Write([]string{h.EventName, h.AgeGroup, strconv.Itoa(h.Number), h.Time, strconv.Itoa(i + 1),
				safeCell(st.LastName), safeCell(st.FirstName), st.GroupCode, st.House})
		}
	}
	cw.Flush()
	sendFile(w, "athletics-schedule.csv", "text/csv; charset=utf-8", buf.Bytes())
}

// ---------- times ----------

func parseClock(v string) (int, bool) {
	t, err := time.Parse("15:04", strings.TrimSpace(v))
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func fmtClock(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

// handleSetTimes gives every heat a start time. With start and end it spreads the
// heats evenly across the day, in schedule order, so the last heat finishes at `end`.
// Individual times can still be edited by hand afterwards. {"clear": true} removes all times.
func (s *Server) handleSetTimes(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Clear bool   `json:"clear"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Clear {
		s.db.Exec(`UPDATE heats SET time=''`)
		s.handleGetSchedule(w, r)
		return
	}
	start, ok1 := parseClock(in.Start)
	end, ok2 := parseClock(in.End)
	if !ok1 || !ok2 {
		fail(w, 400, "Please choose a start and a finish time")
		return
	}
	if end <= start {
		fail(w, 400, "The finish time must be after the start time")
		return
	}
	heats, err := s.loadHeats()
	if err != nil {
		serverErr(w, err)
		return
	}
	if len(heats) == 0 {
		fail(w, 400, "Create the schedule first")
		return
	}
	slot := float64(end-start) / float64(len(heats))
	step := 5.0 // tidy 5-minute marks, unless heats are shorter than that
	if slot < 5 {
		step = 1
	}
	tx, err := s.db.Begin()
	if err != nil {
		serverErr(w, err)
		return
	}
	defer tx.Rollback()
	for i, h := range heats {
		m := start + int(math.Round(float64(i)*slot/step)*step)
		if _, err := tx.Exec(`UPDATE heats SET time=? WHERE id=?`, fmtClock(m), h.ID); err != nil {
			serverErr(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		serverErr(w, err)
		return
	}
	setSetting(s.db, "day_start", fmtClock(start))
	setSetting(s.db, "day_end", fmtClock(end))
	s.handleGetSchedule(w, r)
}
