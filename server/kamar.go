package main

// KAMAR Directory Services listener.
//
// KAMAR pushes data to us; there is no API to call. The school enters our URL
// (POST /api/kamar), a username and a password in Setup > Server > Directory
// Services. KAMAR then sends:
//   - "check"        when the school presses Check and Enable, and daily at 4-5am.
//                    We reply with what data we want.
//   - "full"/"part"  student (and staff) data. We keep only what we asked for.
//   - "deletenotice" the school removed our service: we must delete its data.
//   - anything else  (timetables, attendance...) is acknowledged and ignored.
//
// We only ask for four student fields: preferred first name, last name,
// date of birth and tutor group. Docs: https://directoryservices.kamar.nz/

import (
	"bytes"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const kamarMaxBody = 64 << 20 // timetable messages are ~3 MB; leave plenty of room

// KAMAR field names we ask for. `firstname` is the *preferred* first name
// (`firstnamelegal` is the legal one) and `tutor` is the tutor/form class.
const kamarFields = "firstname;lastname;tutor;datebirth"

// flexString accepts a JSON string or number, because KAMAR is not consistent.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexString(strings.TrimSpace(v))
		return nil
	}
	*f = flexString(s)
	return nil
}

type kamarStudent struct {
	ID        flexString `json:"id"`
	FirstName flexString `json:"firstname"`
	LastName  flexString `json:"lastname"`
	Tutor     flexString `json:"tutor"`
	DateBirth flexString `json:"datebirth"`
}

type kamarEnvelope struct {
	Data struct {
		Sync     string          `json:"sync"`
		Students json.RawMessage `json:"students"`
	} `json:"SMSDirectoryData"`
}

type kamarCounts struct {
	Added    int `json:"added"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	BadGroup int `json:"bad_group"`
}

func kamarReply(w http.ResponseWriter, code int, result string) {
	errNo := 0
	if code != 200 {
		errNo = code
	}
	writeJSON(w, code, map[string]any{"SMSDirectoryData": map[string]any{"error": errNo, "result": result}})
}

// authorised checks the Basic auth header in constant time. With no login
// configured it refuses everything.
func (s *Server) kamarAuthorised(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok || s.kamarUser == "" || s.kamarPass == "" {
		return false
	}
	u := subtle.ConstantTimeCompare([]byte(user), []byte(s.kamarUser))
	p := subtle.ConstantTimeCompare([]byte(pass), []byte(s.kamarPass))
	return u&p == 1
}

func (s *Server) handleKamar(w http.ResponseWriter, r *http.Request) {
	// Authenticate before touching the body: KAMAR sends wrong passwords on purpose.
	if !s.kamarAuthorised(r) {
		kamarReply(w, 403, "Authentication Failed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, kamarMaxBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("kamar: could not read body: %v", err)
		kamarReply(w, 413, "Message too large or interrupted")
		return
	}
	if t := bytes.TrimSpace(body); len(t) > 0 && t[0] == '<' {
		log.Printf("kamar: received XML (%d bytes). This service only reads JSON: ask the school to choose JSON in KAMAR", len(body))
		kamarReply(w, 415, "This service only accepts JSON. Please choose JSON in the KAMAR Directory Services setup.")
		return
	}
	var msg kamarEnvelope
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("kamar: invalid JSON (%d bytes): %v", len(body), err)
		kamarReply(w, 400, "Invalid JSON")
		return
	}

	switch sync := strings.ToLower(msg.Data.Sync); sync {
	case "check":
		log.Printf("kamar: check")
		writeJSON(w, 200, s.kamarCheckReply())
	case "full", "part":
		counts, fields, err := s.kamarApplyStudents(msg.Data.Students)
		if err != nil {
			log.Printf("kamar: %s sync failed: %v", sync, err)
			kamarReply(w, 500, "Could not store the data")
			return
		}
		// Field names only, never values: students are children.
		log.Printf("kamar: %s sync, %d bytes, added=%d updated=%d skipped=%d badgroup=%d fields=%s",
			sync, len(body), counts.Added, counts.Updated, counts.Skipped, counts.BadGroup, strings.Join(fields, ","))
		s.kamarRecord(sync, counts)
		kamarReply(w, 200, "OK")
	case "deletenotice":
		// The school removed our service: delete everything that came from them.
		res, err := s.db.Exec(`DELETE FROM students WHERE kamar_id IS NOT NULL`)
		if err != nil {
			log.Printf("kamar: delete notice failed: %v", err)
			kamarReply(w, 500, "Could not delete the data")
			return
		}
		n, _ := res.RowsAffected()
		touchEntries(s.db)
		setSetting(s.db, "kamar_last", "")
		log.Printf("kamar: delete notice, removed %d students", n)
		kamarReply(w, 200, "OK")
	default:
		// Timetables, attendance, etc. Acknowledge so KAMAR doesn't retry or disable us.
		log.Printf("kamar: ignoring %q message (%d bytes)", sync, len(body))
		kamarReply(w, 200, "OK")
	}
}

func (s *Server) kamarCheckReply() map[string]any {
	minLevel, _ := strconv.Atoi(env("KAMAR_MIN_LEVEL", "9"))
	maxLevel, _ := strconv.Atoi(env("KAMAR_MAX_LEVEL", "13"))
	// Every option is sent explicitly: KAMAR left its data boxes greyed out otherwise.
	return map[string]any{"SMSDirectoryData": map[string]any{
		"error":             0,
		"result":            "OK",
		"service":           "Athletics Day",
		"version":           1.0,
		"status":            "Ready",
		"infourl":           env("KAMAR_INFO_URL", "https://www.kingshigh.school.nz/"),
		"privacystatement":  env("KAMAR_PRIVACY", "Used only to organise the school athletics day. Names, dates of birth and tutor groups are kept for the event and removed afterwards.|Stored in a database on a server in New Zealand with access limited to school staff.|Covered by the NZ Privacy Act 2020."),
		"countryDataStored": env("KAMAR_COUNTRY", "New Zealand"),
		"options": map[string]any{
			"ics": false,
			"students": map[string]any{
				"details":         true,
				"minLevel":        minLevel,
				"maxLevel":        maxLevel,
				"photos":          false,
				"groups":          false,
				"awards":          false,
				"timetables":      false,
				"attendance":      false,
				"assessments":     false,
				"pastoral":        false,
				"recognitions":    false,
				"classefforts":    false,
				"learningsupport": false,
				"fields":          map[string]any{"required": kamarFields},
			},
			"staff":  map[string]any{"details": false, "photos": false, "timetables": false},
			"common": map[string]any{"subjects": false, "notices": false, "calendar": false, "bookings": false},
		},
	}}
}

// studentRecords reads "students" whether it is {"count":n,"data":[...]} (as documented) or a bare array.
func studentRecords(raw json.RawMessage) ([]kamarStudent, []string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil, nil
	}
	var list []json.RawMessage
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, nil, err
		}
	} else {
		var wrapped struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, nil, err
		}
		list = wrapped.Data
	}
	out := make([]kamarStudent, 0, len(list))
	seen := map[string]bool{}
	for _, item := range list {
		var st kamarStudent
		if err := json.Unmarshal(item, &st); err != nil {
			return nil, nil, err
		}
		out = append(out, st)
		var keys map[string]json.RawMessage
		if json.Unmarshal(item, &keys) == nil {
			for k := range keys {
				seen[k] = true
			}
		}
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	sort.Strings(names)
	return out, names, nil
}

// kamarGroup turns KAMAR's tutor value into one of our 3-letter groups.
func kamarGroup(tutor string) string {
	tutor = strings.ToUpper(strings.TrimSpace(strings.SplitN(tutor, "\n", 2)[0]))
	if groupRe.MatchString(tutor) {
		return tutor
	}
	return ""
}

func (s *Server) kamarApplyStudents(raw json.RawMessage) (kamarCounts, []string, error) {
	var c kamarCounts
	recs, fields, err := studentRecords(raw)
	if err != nil || len(recs) == 0 {
		return c, fields, err
	}
	ref := refDate(s.db)
	tx, err := s.db.Begin()
	if err != nil {
		return c, fields, err
	}
	defer tx.Rollback()
	changed := false

	for _, rec := range recs {
		kid := string(rec.ID)
		if kid == "" {
			c.Skipped++
			continue
		}
		// A part sync may leave out unchanged fields: start from what we already have.
		var in studentInput
		var sid int64
		var oldDOB string
		err := tx.QueryRow(`SELECT id,first_name,last_name,dob,group_code FROM students WHERE kamar_id=?`, kid).
			Scan(&sid, &in.FirstName, &in.LastName, &oldDOB, &in.GroupCode)
		existing := err == nil
		if err != nil && err != sql.ErrNoRows {
			return c, fields, err
		}
		prev := in
		in.DOB = oldDOB
		if v := string(rec.FirstName); v != "" {
			in.FirstName = v
		}
		if v := string(rec.LastName); v != "" {
			in.LastName = v
		}
		if v := string(rec.DateBirth); v != "" {
			in.DOB = v
		}
		if string(rec.Tutor) != "" {
			g := kamarGroup(string(rec.Tutor))
			if g == "" {
				c.BadGroup++
				c.Skipped++
				continue
			}
			in.GroupCode = g
		}
		if err := in.clean(ref); err != nil {
			c.Skipped++
			continue
		}

		if existing {
			if prev.FirstName == in.FirstName && prev.LastName == in.LastName && oldDOB == in.DOB && prev.GroupCode == in.GroupCode {
				continue
			}
			if _, err := tx.Exec(`UPDATE students SET first_name=?,last_name=?,dob=?,group_code=?,updated_at=? WHERE id=?`,
				in.FirstName, in.LastName, in.DOB, in.GroupCode, now(), sid); err != nil {
				return c, fields, err
			}
			c.Updated++
			changed = true
			continue
		}

		// Not seen before. If staff or the organiser already added this student by hand, link them.
		err = tx.QueryRow(`SELECT id FROM students WHERE kamar_id IS NULL AND lower(first_name)=lower(?) AND lower(last_name)=lower(?) AND dob=? AND group_code=?`,
			in.FirstName, in.LastName, in.DOB, in.GroupCode).Scan(&sid)
		if err == nil {
			if _, err := tx.Exec(`UPDATE students SET kamar_id=? WHERE id=?`, kid, sid); err != nil {
				return c, fields, err
			}
			c.Updated++
			continue
		}
		if _, err := tx.Exec(`INSERT INTO students(first_name,last_name,dob,group_code,kamar_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
			in.FirstName, in.LastName, in.DOB, in.GroupCode, kid, now(), now()); err != nil {
			return c, fields, fmt.Errorf("insert: %w", err)
		}
		c.Added++
		changed = true
	}
	if err := tx.Commit(); err != nil {
		return c, fields, err
	}
	if changed {
		touchEntries(s.db)
	}
	return c, fields, nil
}

// ---------- status for the organiser ----------

type kamarLast struct {
	At    string      `json:"at"`
	Sync  string      `json:"sync"`
	Count kamarCounts `json:"counts"`
}

func (s *Server) kamarRecord(sync string, c kamarCounts) {
	b, _ := json.Marshal(kamarLast{At: now(), Sync: sync, Count: c})
	setSetting(s.db, "kamar_last", string(b))
}

func (s *Server) handleKamarStatus(w http.ResponseWriter, r *http.Request) {
	var linked int
	s.db.QueryRow(`SELECT COUNT(*) FROM students WHERE kamar_id IS NOT NULL`).Scan(&linked)
	var last *kamarLast
	if raw := getSetting(s.db, "kamar_last", ""); raw != "" {
		last = &kamarLast{}
		if json.Unmarshal([]byte(raw), last) != nil {
			last = nil
		}
	}
	writeJSON(w, 200, map[string]any{
		"configured": s.kamarUser != "" && s.kamarPass != "",
		"linked":     linked,
		"last":       last,
	})
}
