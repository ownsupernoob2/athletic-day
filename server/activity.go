package main

// Progress for everyone, plus a demo-only view of who has finished and what the
// server is doing. The demo view is switched on with DEMO_VISIBILITY=1 and must
// stay off in production: it shows staff names and student names.

import (
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type ActivityEvent struct {
	At    string `json:"at"`
	Kind  string `json:"kind"` // auth, entries, students, schedule, staff, kamar, system
	Actor string `json:"actor"`
	Text  string `json:"text"`
}

// eventLog keeps the most recent events in memory. It resets when the server restarts.
type eventLog struct {
	mu   sync.Mutex
	list []ActivityEvent
	max  int
}

func newEventLog(max int) *eventLog { return &eventLog{max: max} }

func (l *eventLog) add(e ActivityEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.list = append(l.list, e)
	if len(l.list) > l.max {
		l.list = l.list[len(l.list)-l.max:]
	}
}

// newest returns up to n events, newest first.
func (l *eventLog) newest(n int) []ActivityEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n > len(l.list) {
		n = len(l.list)
	}
	out := make([]ActivityEvent, 0, n)
	for i := len(l.list) - 1; i >= len(l.list)-n; i-- {
		out = append(out, l.list[i])
	}
	return out
}

// logEvent records something that happened. Safe to call when logging is not set up (tests).
func (s *Server) logEvent(kind, actor, text string) {
	if s.events == nil {
		return
	}
	s.events.add(ActivityEvent{At: now(), Kind: kind, Actor: actor, Text: text})
}

// countRequests counts every request, for the demo "server" panel.
func (s *Server) countRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.requests, 1)
		next.ServeHTTP(w, r)
	})
}

// groupStatuses is shared by the organiser overview and the progress page.
func (s *Server) groupStatuses() ([]GroupStatus, error) {
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
		return nil, err
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
		return nil, err
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
	return out, nil
}

// handleProgress: how the whole day is coming together. Totals are safe for any signed-in user.
// Group-by-group detail with staff names is demo-only.
func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	groups, err := s.groupStatuses()
	if err != nil {
		serverErr(w, err)
		return
	}
	min, _ := strconv.Atoi(getSetting(s.db, "min_events", "3"))
	p := map[string]any{"min_events": min, "demo": s.demo}
	var students, ready, total, done int
	for _, g := range groups {
		students += g.Students
		ready += g.Complete
		if g.Students > 0 {
			total++
			if g.Complete == g.Students {
				done++
			}
		}
	}
	var heats int
	s.db.QueryRow(`SELECT COUNT(*) FROM heats`).Scan(&heats)
	phase := "none"
	if heats > 0 {
		phase = "draft"
		if getSetting(s.db, "published", "0") == "1" {
			phase = "published"
		}
	}
	p["students"], p["ready"], p["groups_total"], p["groups_done"] = students, ready, total, done
	p["schedule"], p["heats"] = phase, heats
	if s.demo {
		p["groups"] = groups
	}
	writeJSON(w, 200, p)
}

// handleActivity is the demo "what is the backend doing" view.
func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	if !s.demo {
		fail(w, 404, "Not available")
		return
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	count := func(q string) (n int) { s.db.QueryRow(q).Scan(&n); return }
	var dbBytes int64
	if fi, err := os.Stat(s.dbPath); err == nil {
		dbBytes = fi.Size()
	}
	events := []ActivityEvent{}
	if s.events != nil {
		events = s.events.newest(150)
	}
	writeJSON(w, 200, map[string]any{
		"events": events,
		"system": map[string]any{
			"uptime_seconds": int(time.Since(s.started).Seconds()),
			"requests":       atomic.LoadInt64(&s.requests),
			"goroutines":     runtime.NumGoroutine(),
			"memory_mb":      float64(mem.Alloc) / (1 << 20),
			"database_kb":    dbBytes / 1024,
			"go_version":     runtime.Version(),
			"students":       count(`SELECT COUNT(*) FROM students`),
			"entries":        count(`SELECT COUNT(*) FROM entries`),
			"heats":          count(`SELECT COUNT(*) FROM heats`),
			"accounts":       count(`SELECT COUNT(*) FROM users`),
			"sessions":       count(`SELECT COUNT(*) FROM sessions WHERE expires_at > '` + now() + `'`),
			"kamar_enabled":  s.kamarUser != "" && s.kamarPass != "",
			"kamar_last":     getSetting(s.db, "kamar_last", ""),
		},
	})
}
