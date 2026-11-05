// Athletics Day API server.
//
// Configuration (environment variables):
//
//	PORT            HTTP port (default 8080)
//	DB_PATH         SQLite file (default athletic.db)
//	ADMIN_EMAIL     first organiser account, created only when no users exist
//	ADMIN_PASSWORD  password for that account
//	SEED_DEMO=1     fill an empty database with demo students and staff
//	COOKIE_SECURE=1 mark the session cookie Secure (set this behind HTTPS)
//	KAMAR_USERNAME / KAMAR_PASSWORD  login KAMAR must send to POST /api/kamar (put in .env)
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	db        *DB
	secure    bool
	limit     *loginLimiter
	kamarUser string // KAMAR Directory Services login. Empty = KAMAR is refused.
	kamarPass string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadDotEnv reads KEY=VALUE lines from .env (if present) without overriding real
// environment variables. Keep secrets such as the KAMAR password there, out of git.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func main() {
	loadDotEnv(".env")
	db, err := openDB(env("DB_PATH", "athletic.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := &Server{db: db, secure: os.Getenv("COOKIE_SECURE") == "1", limit: newLoginLimiter(),
		kamarUser: os.Getenv("KAMAR_USERNAME"), kamarPass: os.Getenv("KAMAR_PASSWORD")}
	if s.kamarUser == "" || s.kamarPass == "" {
		log.Print("KAMAR_USERNAME / KAMAR_PASSWORD not set: KAMAR requests will be refused")
	}
	if err := s.bootstrap(); err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	log.Printf("API listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	authed := s.auth()
	org := s.auth(roleOrganiser)

	// --- public ---
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/public/schedule", s.handlePublicSchedule)
	mux.HandleFunc("GET /api/public/schedule.csv", s.handlePublicScheduleCSV)

	// --- KAMAR pushes student data here (its own Basic auth, not a session) ---
	mux.HandleFunc("POST /api/kamar", s.handleKamar)

	// --- auth ---
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/me", authed(s.handleMe))

	// --- staff + organiser ---
	mux.HandleFunc("GET /api/settings", authed(s.handleGetSettings))
	mux.HandleFunc("GET /api/events", authed(s.handleListEvents))
	mux.HandleFunc("GET /api/students", authed(s.handleListStudents))
	mux.HandleFunc("POST /api/students", authed(s.handleCreateStudent))
	mux.HandleFunc("PUT /api/students/{id}/events/{eid}", authed(s.handleSetEntry))
	mux.HandleFunc("GET /api/export/group.csv", authed(s.handleExportGroup))

	// --- organiser only ---
	mux.HandleFunc("PUT /api/settings", org(s.handlePutSettings))
	mux.HandleFunc("POST /api/events", org(s.handleCreateEvent))
	mux.HandleFunc("PATCH /api/events/{id}", org(s.handleUpdateEvent))
	mux.HandleFunc("DELETE /api/events/{id}", org(s.handleDeleteEvent))

	mux.HandleFunc("PATCH /api/students/{id}", org(s.handleUpdateStudent))
	mux.HandleFunc("DELETE /api/students/{id}", org(s.handleDeleteStudent))
	mux.HandleFunc("POST /api/students/import", org(s.handleImportStudents))
	mux.HandleFunc("POST /api/students/clear", org(s.handleClearStudents))

	mux.HandleFunc("GET /api/kamar/status", org(s.handleKamarStatus))
	mux.HandleFunc("GET /api/groups", org(s.handleGroups))
	mux.HandleFunc("GET /api/staff", org(s.handleListStaff))
	mux.HandleFunc("POST /api/staff", org(s.handleCreateStaff))
	mux.HandleFunc("PATCH /api/staff/{id}", org(s.handleUpdateStaff))
	mux.HandleFunc("DELETE /api/staff/{id}", org(s.handleDeleteStaff))

	mux.HandleFunc("GET /api/export/zip", org(s.handleExportZip))

	mux.HandleFunc("GET /api/schedule", org(s.handleGetSchedule))
	mux.HandleFunc("POST /api/schedule/generate", org(s.handleGenerate))
	mux.HandleFunc("POST /api/schedule/publish", org(s.handlePublish))
	mux.HandleFunc("POST /api/schedule/times", org(s.handleSetTimes))
	mux.HandleFunc("DELETE /api/schedule", org(s.handleClearSchedule))
	mux.HandleFunc("POST /api/heats", org(s.handleCreateHeat))
	mux.HandleFunc("PATCH /api/heats/{id}", org(s.handleUpdateHeat))
	mux.HandleFunc("DELETE /api/heats/{id}", org(s.handleDeleteHeat))
	mux.HandleFunc("POST /api/heats/{id}/students", org(s.handleHeatAddStudent))
	mux.HandleFunc("DELETE /api/heats/{id}/students/{sid}", org(s.handleHeatRemoveStudent))

	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
