package main

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// ---------- settings ----------

func (s *Server) settings() map[string]any {
	min, _ := strconv.Atoi(getSetting(s.db, "min_events", "3"))
	heat, _ := strconv.Atoi(getSetting(s.db, "heat_size", "6"))
	return map[string]any{
		"title":              getSetting(s.db, "title", "School Athletics Day"),
		"event_date":         getSetting(s.db, "event_date", ""),
		"min_events":         min,
		"heat_size":          heat,
		"published":          getSetting(s.db, "published", "0") == "1",
		"generated_at":       getSetting(s.db, "generated_at", ""),
		"day_start":          getSetting(s.db, "day_start", "09:00"),
		"day_end":            getSetting(s.db, "day_end", "15:00"),
		"entries_changed_at": getSetting(s.db, "entries_changed_at", ""),
		"houses":             houses,
	}
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.settings())
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title     *string `json:"title"`
		EventDate *string `json:"event_date"`
		MinEvents *int    `json:"min_events"`
		HeatSize  *int    `json:"heat_size"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" || len(t) > 100 {
			fail(w, 400, "Please enter a title under 100 characters")
			return
		}
		setSetting(s.db, "title", t)
	}
	if in.EventDate != nil {
		if *in.EventDate != "" {
			if _, err := parseDOB(*in.EventDate); err != nil {
				fail(w, 400, "Please enter a valid date")
				return
			}
		}
		setSetting(s.db, "event_date", *in.EventDate)
		touchEntries(s.db) // ages (and so age groups) may change
	}
	if in.MinEvents != nil {
		if *in.MinEvents < 0 || *in.MinEvents > 20 {
			fail(w, 400, "Minimum events must be between 0 and 20")
			return
		}
		setSetting(s.db, "min_events", strconv.Itoa(*in.MinEvents))
	}
	if in.HeatSize != nil {
		if *in.HeatSize < 1 || *in.HeatSize > 20 {
			fail(w, 400, "Heat size must be between 1 and 20")
			return
		}
		setSetting(s.db, "heat_size", strconv.Itoa(*in.HeatSize))
	}
	writeJSON(w, 200, s.settings())
}

// ---------- events ----------

type Event struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	CSVHeader  string `json:"csv_header"`
	Position   int    `json:"position"`
	Active     bool   `json:"active"`
	EntryCount int    `json:"entry_count"`
}

func (s *Server) listEvents(onlyActive bool) ([]Event, error) {
	q := `SELECT e.id,e.code,e.name,e.kind,e.csv_header,e.position,e.active,
		(SELECT COUNT(*) FROM entries n WHERE n.event_id=e.id) FROM events e`
	if onlyActive {
		q += ` WHERE e.active=1`
	}
	q += ` ORDER BY e.position,e.id`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var a int
		if err := rows.Scan(&e.ID, &e.Code, &e.Name, &e.Kind, &e.CSVHeader, &e.Position, &a, &e.EntryCount); err != nil {
			return nil, err
		}
		e.Active = a == 1
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	evs, err := s.listEvents(userFrom(r).Role == roleStaff)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 200, evs)
}

type eventInput struct {
	Code      *string `json:"code"`
	Name      *string `json:"name"`
	Kind      *string `json:"kind"`
	CSVHeader *string `json:"csv_header"`
	Active    *bool   `json:"active"`
	MoveUp    *bool   `json:"move_up"`
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var in eventInput
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name == nil || in.Kind == nil || strings.TrimSpace(*in.Name) == "" || (*in.Kind != "track" && *in.Kind != "field") {
		fail(w, 400, "Please enter a name and choose Track or Field")
		return
	}
	name := strings.TrimSpace(*in.Name)
	code := name
	if in.Code != nil && strings.TrimSpace(*in.Code) != "" {
		code = strings.TrimSpace(*in.Code)
	}
	var pos int
	s.db.QueryRow(`SELECT COALESCE(MAX(position),0)+1 FROM events`).Scan(&pos)
	res, err := s.db.Exec(`INSERT INTO events(code,name,kind,csv_header,position) VALUES(?,?,?,?,?)`, code, name, *in.Kind, code, pos)
	if err != nil {
		serverErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	evs, _ := s.listEvents(false)
	for _, e := range evs {
		if e.ID == id {
			writeJSON(w, 201, e)
			return
		}
	}
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var in eventInput
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			fail(w, 400, "Name can't be empty")
			return
		}
		s.db.Exec(`UPDATE events SET name=? WHERE id=?`, strings.TrimSpace(*in.Name), id)
	}
	if in.Code != nil && strings.TrimSpace(*in.Code) != "" {
		c := strings.TrimSpace(*in.Code)
		s.db.Exec(`UPDATE events SET code=?, csv_header=? WHERE id=?`, c, c, id)
	}
	if in.Kind != nil && (*in.Kind == "track" || *in.Kind == "field") {
		s.db.Exec(`UPDATE events SET kind=? WHERE id=?`, *in.Kind, id)
	}
	if in.Active != nil {
		a := 0
		if *in.Active {
			a = 1
		}
		s.db.Exec(`UPDATE events SET active=? WHERE id=?`, a, id)
	}
	if in.MoveUp != nil {
		s.moveEvent(id, *in.MoveUp)
	}
	evs, err := s.listEvents(false)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 200, evs)
}

// moveEvent swaps an event with its neighbour in the display order.
func (s *Server) moveEvent(id int64, up bool) {
	evs, err := s.listEvents(false)
	if err != nil {
		return
	}
	i := -1
	for k, e := range evs {
		if e.ID == id {
			i = k
		}
	}
	j := i + 1
	if up {
		j = i - 1
	}
	if i < 0 || j < 0 || j >= len(evs) {
		return
	}
	evs[i], evs[j] = evs[j], evs[i]
	for k, e := range evs {
		s.db.Exec(`UPDATE events SET position=? WHERE id=?`, k+1, e.ID)
	}
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	if _, err := s.db.Exec(`DELETE FROM events WHERE id=?`, pathID(r, "id")); err != nil {
		serverErr(w, err)
		return
	}
	touchEntries(s.db)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- group progress ----------

type GroupStatus struct {
	Code       string   `json:"code"`
	House      string   `json:"house"`
	Staff      []string `json:"staff"`
	Students   int      `json:"students"`
	Complete   int      `json:"complete"`
	Started    int      `json:"started"`
	LastUpdate string   `json:"last_update"`
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	min, _ := strconv.Atoi(getSetting(s.db, "min_events", "3"))
	groups := map[string]*GroupStatus{}
	get := func(code string) *GroupStatus {
		g, ok := groups[code]
		if !ok {
			g = &GroupStatus{Code: code, House: houseOf(code), Staff: []string{}}
			groups[code] = g
		}
		return g
	}

	rows, err := s.db.Query(`SELECT s.group_code, COUNT(e.event_id), s.updated_at FROM students s
		LEFT JOIN entries e ON e.student_id=s.id GROUP BY s.id`)
	if err != nil {
		serverErr(w, err)
		return
	}
	for rows.Next() {
		var code, upd string
		var n int
		rows.Scan(&code, &n, &upd)
		g := get(code)
		g.Students++
		if n >= min {
			g.Complete++
		}
		if n > 0 {
			g.Started++
			if upd > g.LastUpdate {
				g.LastUpdate = upd
			}
		}
	}
	rows.Close()

	sr, err := s.db.Query(`SELECT group_code,name FROM users WHERE role='staff' AND group_code<>''`)
	if err != nil {
		serverErr(w, err)
		return
	}
	for sr.Next() {
		var code, name string
		sr.Scan(&code, &name)
		g := get(code)
		g.Staff = append(g.Staff, name)
	}
	sr.Close()

	out := make([]GroupStatus, 0, len(groups))
	for _, g := range groups {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	writeJSON(w, 200, out)
}

// ---------- staff accounts ----------

type StaffUser struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	GroupCode string `json:"group_code"`
	House     string `json:"house"`
}

func (s *Server) handleListStaff(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`SELECT id,name,email,role,group_code FROM users ORDER BY role DESC, group_code, name`)
	if err != nil {
		serverErr(w, err)
		return
	}
	defer rows.Close()
	out := []StaffUser{}
	for rows.Next() {
		var u StaffUser
		rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.GroupCode)
		u.House = houseOf(u.GroupCode)
		out = append(out, u)
	}
	writeJSON(w, 200, out)
}

type staffInput struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	GroupCode string `json:"group_code"`
}

func (in *staffInput) clean(needPassword bool) string {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.GroupCode = strings.ToUpper(strings.TrimSpace(in.GroupCode))
	if in.Role == "" {
		in.Role = roleStaff
	}
	switch {
	case in.Name == "":
		return "Please enter a name"
	case !strings.Contains(in.Email, "@") || len(in.Email) > 200:
		return "Please enter a valid email"
	case in.Role != roleStaff && in.Role != roleOrganiser:
		return "Unknown role"
	case in.Role == roleStaff && !groupRe.MatchString(in.GroupCode):
		return "Staff need a 3-letter group, like SSH"
	case (needPassword || in.Password != "") && len(in.Password) < 8:
		return "Password must be at least 8 characters"
	}
	if in.Role == roleOrganiser {
		in.GroupCode = ""
	}
	return ""
}

func isUnique(err error) bool { return err != nil && strings.Contains(err.Error(), "UNIQUE") }

func (s *Server) handleCreateStaff(w http.ResponseWriter, r *http.Request) {
	var in staffInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.clean(true); msg != "" {
		fail(w, 400, msg)
		return
	}
	id, err := s.createUser(in.Name, in.Email, in.Password, in.Role, in.GroupCode)
	if isUnique(err) {
		fail(w, 409, "That email already has an account")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 201, StaffUser{id, in.Name, in.Email, in.Role, in.GroupCode, houseOf(in.GroupCode)})
}

func (s *Server) handleUpdateStaff(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var in staffInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.clean(false); msg != "" {
		fail(w, 400, msg)
		return
	}
	if me := userFrom(r); me.ID == id && in.Role != roleOrganiser {
		fail(w, 400, "You can't remove your own organiser access")
		return
	}
	_, err := s.db.Exec(`UPDATE users SET name=?,email=?,role=?,group_code=? WHERE id=?`, in.Name, in.Email, in.Role, in.GroupCode, id)
	if isUnique(err) {
		fail(w, 409, "That email already has an account")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	if in.Password != "" {
		hash, err := hashPassword(in.Password)
		if err != nil {
			serverErr(w, err)
			return
		}
		s.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, hash, id)
		s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
	}
	writeJSON(w, 200, StaffUser{id, in.Name, in.Email, in.Role, in.GroupCode, houseOf(in.GroupCode)})
}

func (s *Server) handleDeleteStaff(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if userFrom(r).ID == id {
		fail(w, 400, "You can't delete your own account")
		return
	}
	s.db.Exec(`DELETE FROM users WHERE id=?`, id)
	writeJSON(w, 200, map[string]bool{"ok": true})
}
