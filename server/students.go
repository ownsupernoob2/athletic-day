package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------- houses & age groups ----------

type House struct {
	Letter string `json:"letter"`
	Name   string `json:"name"`
}

var houses = []House{{"S", "Stuart"}, {"T", "Tudor"}, {"W", "Windsor"}, {"H", "Hanover"}}

// houseOf: a group's first letter is its house's first letter.
func houseOf(group string) string {
	if group == "" {
		return ""
	}
	for _, h := range houses {
		if strings.EqualFold(group[:1], h.Letter) {
			return h.Name
		}
	}
	return ""
}

var ageGroups = []string{"Under 14", "Under 15", "Under 16", "Under 19", "Over 19"}

func ageGroupIndex(g string) int {
	for i, a := range ageGroups {
		if a == g {
			return i
		}
	}
	return len(ageGroups)
}

func ageGroupOf(age int) string {
	switch {
	case age < 14:
		return "Under 14"
	case age < 15:
		return "Under 15"
	case age < 16:
		return "Under 16"
	case age < 19:
		return "Under 19"
	}
	return "Over 19"
}

func ageOn(dob string, ref time.Time) int {
	d, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return 0
	}
	age := ref.Year() - d.Year()
	if ref.Month() < d.Month() || (ref.Month() == d.Month() && ref.Day() < d.Day()) {
		age--
	}
	return age
}

// refDate is the date ages are measured on: the athletics day if set, else today.
func refDate(db *DB) time.Time {
	if t, err := time.Parse("2006-01-02", getSetting(db, "event_date", "")); err == nil {
		return t
	}
	n := time.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// ---------- model ----------

type Student struct {
	ID           int64   `json:"id"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	DOB          string  `json:"dob"`
	Age          int     `json:"age"`
	AgeGroup     string  `json:"age_group"`
	GroupCode    string  `json:"group_code"`
	House        string  `json:"house"`
	AddedByStaff bool    `json:"added_by_staff"`
	FromKamar    bool    `json:"from_kamar"`
	UpdatedAt    string  `json:"updated_at"`
	Events       []int64 `json:"events"`
}

// loadStudents returns students (optionally for one group) with their event ids.
func (s *Server) loadStudents(group string, id int64) ([]Student, error) {
	ref := refDate(s.db)
	q := `SELECT id,first_name,last_name,dob,group_code,added_by_staff,updated_at,kamar_id IS NOT NULL FROM students`
	var args []any
	switch {
	case id > 0:
		q += ` WHERE id=?`
		args = append(args, id)
	case group != "":
		q += ` WHERE group_code=?`
		args = append(args, group)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Student{}
	idx := map[int64]int{}
	for rows.Next() {
		var st Student
		var added, kamar int
		if err := rows.Scan(&st.ID, &st.FirstName, &st.LastName, &st.DOB, &st.GroupCode, &added, &st.UpdatedAt, &kamar); err != nil {
			return nil, err
		}
		st.AddedByStaff = added == 1
		st.FromKamar = kamar == 1
		st.Age = ageOn(st.DOB, ref)
		st.AgeGroup = ageGroupOf(st.Age)
		st.House = houseOf(st.GroupCode)
		st.Events = []int64{}
		idx[st.ID] = len(out)
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	er, err := s.db.Query(`SELECT student_id,event_id FROM entries`)
	if err != nil {
		return nil, err
	}
	defer er.Close()
	for er.Next() {
		var sid, eid int64
		er.Scan(&sid, &eid)
		if i, ok := idx[sid]; ok {
			out[i].Events = append(out[i].Events, eid)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if x, y := ageGroupIndex(a.AgeGroup), ageGroupIndex(b.AgeGroup); x != y {
			return x < y
		}
		if a.LastName != b.LastName {
			return strings.ToLower(a.LastName) < strings.ToLower(b.LastName)
		}
		return strings.ToLower(a.FirstName) < strings.ToLower(b.FirstName)
	})
	return out, nil
}

// ---------- input validation ----------

var groupRe = regexp.MustCompile(`^[A-Z]{3}$`)

type studentInput struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	DOB       string `json:"dob"`
	GroupCode string `json:"group_code"`
}

func parseDOB(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "20060102", "02/01/2006", "2/1/2006", "02-01-2006", "2/1/06"} {
		if t, err := time.Parse(layout, s); err == nil {
			if layout == "2/1/06" && t.Year() > time.Now().Year() {
				t = t.AddDate(-100, 0, 0)
			}
			return t.Format("2006-01-02"), nil
		}
	}
	return "", errors.New("Date of birth must look like 31/12/2010")
}

func (in *studentInput) clean(ref time.Time) error {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.GroupCode = strings.ToUpper(strings.TrimSpace(in.GroupCode))
	if in.FirstName == "" || in.LastName == "" {
		return errors.New("Please enter a first and last name")
	}
	if utf8.RuneCountInString(in.FirstName) > 60 || utf8.RuneCountInString(in.LastName) > 60 {
		return errors.New("Names must be under 60 characters")
	}
	if !groupRe.MatchString(in.GroupCode) {
		return errors.New("Group must be 3 letters, like SSH")
	}
	dob, err := parseDOB(in.DOB)
	if err != nil {
		return err
	}
	in.DOB = dob
	if age := ageOn(dob, ref); age < 5 || age > 30 {
		return errors.New("That date of birth doesn't look right")
	}
	return nil
}

func pathID(r *http.Request, name string) int64 {
	n, _ := strconv.ParseInt(r.PathValue(name), 10, 64)
	return n
}

// canTouch: organisers can edit anyone; staff only their own group.
func canTouch(u *User, group string) bool { return u.Role == roleOrganiser || u.GroupCode == group }

// ---------- handlers ----------

func (s *Server) handleListStudents(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	group := ""
	if u.Role == roleStaff {
		group = u.GroupCode
		if group == "" {
			writeJSON(w, 200, []Student{})
			return
		}
	} else if g := strings.ToUpper(r.URL.Query().Get("group")); g != "" {
		group = g
	}
	list, err := s.loadStudents(group, 0)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreateStudent(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var in studentInput
	if !readJSON(w, r, &in) {
		return
	}
	if u.Role == roleStaff {
		in.GroupCode = u.GroupCode
		if in.GroupCode == "" {
			fail(w, 400, "Your account has no group yet. Ask the organiser.")
			return
		}
	}
	if err := in.clean(refDate(s.db)); err != nil {
		fail(w, 400, err.Error())
		return
	}
	var existing int64
	err := s.db.QueryRow(`SELECT id FROM students WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?) AND dob=? AND group_code=?`,
		in.FirstName, in.LastName, in.DOB, in.GroupCode).Scan(&existing)
	if err == nil {
		fail(w, 409, fmt.Sprintf("%s %s is already in %s", in.FirstName, in.LastName, in.GroupCode))
		return
	}
	added := 0
	if u.Role == roleStaff {
		added = 1
	}
	res, err := s.db.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,added_by_staff,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		in.FirstName, in.LastName, in.DOB, in.GroupCode, added, now(), now())
	if err != nil {
		serverErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	list, _ := s.loadStudents("", id)
	if len(list) == 0 {
		serverErr(w, errors.New("student vanished"))
		return
	}
	writeJSON(w, 201, list[0])
}

func (s *Server) handleUpdateStudent(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var in studentInput
	if !readJSON(w, r, &in) {
		return
	}
	if err := in.clean(refDate(s.db)); err != nil {
		fail(w, 400, err.Error())
		return
	}
	res, err := s.db.Exec(`UPDATE students SET first_name=?,last_name=?,dob=?,group_code=?,updated_at=? WHERE id=?`,
		in.FirstName, in.LastName, in.DOB, in.GroupCode, now(), id)
	if err != nil {
		serverErr(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "Student not found")
		return
	}
	touchEntries(s.db)
	list, _ := s.loadStudents("", id)
	writeJSON(w, 200, list[0])
}

func (s *Server) handleDeleteStudent(w http.ResponseWriter, r *http.Request) {
	if _, err := s.db.Exec(`DELETE FROM students WHERE id=?`, pathID(r, "id")); err != nil {
		serverErr(w, err)
		return
	}
	touchEntries(s.db)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleClearStudents(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Confirm != "DELETE" {
		fail(w, 400, "Type DELETE to confirm")
		return
	}
	if _, err := s.db.Exec(`DELETE FROM students`); err != nil {
		serverErr(w, err)
		return
	}
	touchEntries(s.db)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// handleSetEntry is the auto-save endpoint: tick or untick one event for one student.
func (s *Server) handleSetEntry(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	sid, eid := pathID(r, "id"), pathID(r, "eid")
	var in struct {
		On bool `json:"on"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	var group string
	if err := s.db.QueryRow(`SELECT group_code FROM students WHERE id=?`, sid).Scan(&group); errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "Student not found")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	if !canTouch(u, group) {
		fail(w, 403, "That student isn't in your group")
		return
	}
	var active int
	if err := s.db.QueryRow(`SELECT active FROM events WHERE id=?`, eid).Scan(&active); err != nil || (active == 0 && u.Role == roleStaff) {
		fail(w, 404, "Event not found")
		return
	}
	var err error
	if in.On {
		_, err = s.db.Exec(`INSERT OR IGNORE INTO entries(student_id,event_id) VALUES(?,?)`, sid, eid)
	} else {
		_, err = s.db.Exec(`DELETE FROM entries WHERE student_id=? AND event_id=?`, sid, eid)
		s.db.Exec(`DELETE FROM heat_entries WHERE student_id=? AND heat_id IN (SELECT id FROM heats WHERE event_id=?)`, sid, eid)
	}
	if err != nil {
		serverErr(w, err)
		return
	}
	s.db.Exec(`UPDATE students SET updated_at=? WHERE id=?`, now(), sid)
	touchEntries(s.db)
	writeJSON(w, 200, map[string]any{"ok": true, "saved_at": now()})
}

// ---------- CSV import ----------

type importResult struct {
	Added   int      `json:"added"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

func norm(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// handleImportStudents reads a CSV with the columns First Name, Last Name, DOB, Group ID
// (the format of old/data.csv). Column order is taken from the header row.
func (s *Server) handleImportStudents(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, _, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "Please choose a CSV file")
		return
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	text := strings.TrimPrefix(string(data), "\ufeff")
	cr := csv.NewReader(strings.NewReader(text))
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil || len(records) < 2 {
		fail(w, 400, "That file doesn't look like a CSV with a header row and students")
		return
	}

	col := map[string]int{"first": -1, "last": -1, "dob": -1, "group": -1}
	for i, h := range records[0] {
		switch n := norm(h); {
		case n == "first name" || n == "first" || n == "firstname" || n == "given name":
			col["first"] = i
		case n == "last name" || n == "last" || n == "lastname" || n == "surname" || n == "family name":
			col["last"] = i
		case n == "dob" || n == "date of birth" || n == "birthdate" || n == "birth date":
			col["dob"] = i
		case n == "group id" || n == "group" || n == "group code" || n == "room" || n == "form":
			col["group"] = i
		}
	}
	for k, v := range col {
		if v < 0 {
			fail(w, 400, fmt.Sprintf("Missing a %q column. Expected: First Name, Last Name, DOB, Group ID", map[string]string{"first": "First Name", "last": "Last Name", "dob": "DOB", "group": "Group ID"}[k]))
			return
		}
	}

	ref := refDate(s.db)
	res := importResult{Errors: []string{}}
	tx, err := s.db.Begin()
	if err != nil {
		serverErr(w, err)
		return
	}
	defer tx.Rollback()
	get := func(rec []string, i int) string {
		if i < len(rec) {
			return rec[i]
		}
		return ""
	}
	for n, rec := range records[1:] {
		line := n + 2
		allBlank := true
		for _, c := range rec {
			if strings.TrimSpace(c) != "" {
				allBlank = false
			}
		}
		if allBlank {
			continue
		}
		in := studentInput{get(rec, col["first"]), get(rec, col["last"]), get(rec, col["dob"]), get(rec, col["group"])}
		if err := in.clean(ref); err != nil {
			res.Skipped++
			if len(res.Errors) < 20 {
				res.Errors = append(res.Errors, fmt.Sprintf("Row %d: %s", line, err.Error()))
			}
			continue
		}
		var dup int64
		if tx.QueryRow(`SELECT id FROM students WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?) AND dob=? AND group_code=?`,
			in.FirstName, in.LastName, in.DOB, in.GroupCode).Scan(&dup) == nil {
			res.Skipped++
			continue
		}
		if _, err := tx.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
			in.FirstName, in.LastName, in.DOB, in.GroupCode, now(), now()); err != nil {
			serverErr(w, err)
			return
		}
		res.Added++
	}
	if err := tx.Commit(); err != nil {
		serverErr(w, err)
		return
	}
	touchEntries(s.db)
	writeJSON(w, 200, res)
}
