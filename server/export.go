package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// colLetter turns 0 -> A, 25 -> Z, 26 -> AA ...
func colLetter(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// safeCell stops spreadsheet apps treating text as a formula.
func safeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// groupCSV writes one group's students in the same layout as the Flask MVP's formatted.csv:
// five header rows, then Last, First, Group, Age group and a column letter per ticked event.
func groupCSV(students []Student, events []Event, year int) []byte {
	var buf bytes.Buffer
	buf.WriteString("\ufeff") // BOM so Excel reads UTF-8 names correctly
	w := csv.NewWriter(&buf)
	n := 4 + len(events)
	row := func(cells map[int]string) []string {
		r := make([]string, n)
		for i, v := range cells {
			if i < n {
				r[i] = v
			}
		}
		return r
	}

	firstTrack, firstField := -1, -1
	for i, e := range events {
		if e.Kind == "track" && firstTrack < 0 {
			firstTrack = 4 + i
		}
		if e.Kind == "field" && firstField < 0 {
			firstField = 4 + i
		}
	}
	kinds := map[int]string{}
	if firstTrack >= 0 {
		kinds[firstTrack] = "TRACK"
	}
	if firstField >= 0 {
		kinds[firstField] = "FIELD"
	}
	w.Write(row(map[int]string{7: fmt.Sprintf("%d EVENTS", year)}))
	w.Write(row(nil))
	w.Write(row(kinds))
	head, sub := map[int]string{0: "NAME", 3: "Age"}, map[int]string{0: "Last", 1: "First", 2: "Group", 3: "Group"}
	for i, e := range events {
		head[4+i] = colLetter(i)
		sub[4+i] = e.CSVHeader
	}
	w.Write(row(head))
	w.Write(row(sub))

	sorted := append([]Student(nil), students...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if x, y := ageGroupIndex(a.AgeGroup), ageGroupIndex(b.AgeGroup); x != y {
			return x < y
		}
		if a.LastName != b.LastName {
			return strings.ToLower(a.LastName) < strings.ToLower(b.LastName)
		}
		return strings.ToLower(a.FirstName) < strings.ToLower(b.FirstName)
	})
	for _, st := range sorted {
		has := map[int64]bool{}
		for _, id := range st.Events {
			has[id] = true
		}
		cells := map[int]string{0: safeCell(st.LastName), 1: safeCell(st.FirstName), 2: st.GroupCode, 3: st.AgeGroup}
		for i, e := range events {
			if has[e.ID] {
				cells[4+i] = colLetter(i)
			}
		}
		w.Write(row(cells))
	}
	w.Flush()
	return buf.Bytes()
}

func sendFile(w http.ResponseWriter, name, mime string, data []byte) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Write(data)
}

func (s *Server) handleExportGroup(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	group := strings.ToUpper(r.URL.Query().Get("group"))
	if u.Role == roleStaff || group == "" {
		group = u.GroupCode
	}
	if !groupRe.MatchString(group) || !canTouch(u, group) {
		fail(w, 400, "Unknown group")
		return
	}
	students, err := s.loadStudents(group, 0)
	if err != nil {
		serverErr(w, err)
		return
	}
	events, err := s.listEvents(true)
	if err != nil {
		serverErr(w, err)
		return
	}
	sendFile(w, group+".csv", "text/csv; charset=utf-8", groupCSV(students, events, refDate(s.db).Year()))
}

// handleExportZip: one folder per house, one CSV per group inside it.
func (s *Server) handleExportZip(w http.ResponseWriter, r *http.Request) {
	students, err := s.loadStudents("", 0)
	if err != nil {
		serverErr(w, err)
		return
	}
	events, err := s.listEvents(true)
	if err != nil {
		serverErr(w, err)
		return
	}
	year := refDate(s.db).Year()
	byGroup := map[string][]Student{}
	for _, st := range students {
		byGroup[st.GroupCode] = append(byGroup[st.GroupCode], st)
	}
	codes := make([]string, 0, len(byGroup))
	for c := range byGroup {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, h := range houses { // always create all four house folders
		zw.CreateHeader(&zip.FileHeader{Name: h.Name + "/", Modified: time.Now()})
	}
	for _, code := range codes {
		folder := houseOf(code)
		if folder == "" {
			folder = "Other"
		}
		f, err := zw.CreateHeader(&zip.FileHeader{Name: folder + "/" + code + ".csv", Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			serverErr(w, err)
			return
		}
		f.Write(groupCSV(byGroup[code], events, year))
	}
	if err := zw.Close(); err != nil {
		serverErr(w, err)
		return
	}
	sendFile(w, fmt.Sprintf("athletics-entries-%d.zip", year), "application/zip", buf.Bytes())
}
