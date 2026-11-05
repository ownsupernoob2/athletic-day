package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type DB = sql.DB

const schema = `
CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  name          TEXT NOT NULL,
  email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  role          TEXT NOT NULL CHECK (role IN ('staff','organiser')),
  group_code    TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS students (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  first_name     TEXT NOT NULL,
  last_name      TEXT NOT NULL,
  dob            TEXT NOT NULL,
  group_code     TEXT NOT NULL,
  added_by_staff INTEGER NOT NULL DEFAULT 0,
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_students_group ON students(group_code);
CREATE TABLE IF NOT EXISTS events (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  code       TEXT NOT NULL,
  name       TEXT NOT NULL,
  kind       TEXT NOT NULL CHECK (kind IN ('track','field')),
  csv_header TEXT NOT NULL,
  position   INTEGER NOT NULL,
  active     INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS entries (
  student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
  event_id   INTEGER NOT NULL REFERENCES events(id)   ON DELETE CASCADE,
  PRIMARY KEY (student_id, event_id)
);
CREATE TABLE IF NOT EXISTS heats (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id  INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  age_group TEXT NOT NULL,
  number    INTEGER NOT NULL,
  time      TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS heat_entries (
  heat_id    INTEGER NOT NULL REFERENCES heats(id)    ON DELETE CASCADE,
  student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
  position   INTEGER NOT NULL,
  PRIMARY KEY (heat_id, student_id)
);
CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

func openDB(path string) (*DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	// Added later: the student's KAMAR id. Fails harmlessly when the column already exists.
	db.Exec(`ALTER TABLE students ADD COLUMN kamar_id TEXT`)
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_students_kamar ON students(kamar_id) WHERE kamar_id IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// Events from the original Flask MVP, in the original column order (A..P).
var defaultEvents = []struct{ code, name, kind, header string }{
	{"100m", "100m", "track", "100"},
	{"200m", "200m", "track", "200"},
	{"400m", "400m", "track", "400"},
	{"800m", "800m", "track", "800"},
	{"1500m", "1500m", "track", "1500"},
	{"3000m", "3000m", "track", "3000"},
	{"DIS", "Discus", "field", "DIS"},
	{"HJ", "High Jump", "field", "HJ"},
	{"TJ", "Triple Jump", "field", "TJ"},
	{"JAV", "Javelin", "field", "JAV"},
	{"SHOT", "Shot Put", "field", "SHOT"},
	{"LJ", "Long Jump", "field", "LJ"},
	{"80H", "80m Hurdles", "track", "80H"},
	{"100H", "100m Hurdles", "track", "100H"},
	{"Softball", "Softball Throw", "field", "Softball"},
	{"U19 Pent", "U19 Pentathlon", "field", "U19 Pent"},
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// bootstrap seeds default events/settings and, on first run, an organiser account.
func (s *Server) bootstrap() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		for i, e := range defaultEvents {
			if _, err := s.db.Exec(`INSERT INTO events(code,name,kind,csv_header,position) VALUES(?,?,?,?,?)`,
				e.code, e.name, e.kind, e.header, i+1); err != nil {
				return err
			}
		}
	}
	for k, v := range map[string]string{
		"title":      "School Athletics Day",
		"min_events": "3",
		"heat_size":  "6",
		"published":  "0",
	} {
		s.db.Exec(`INSERT OR IGNORE INTO settings(key,value) VALUES(?,?)`, k, v)
	}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		email := strings.ToLower(env("ADMIN_EMAIL", "organiser@school.test"))
		pw := env("ADMIN_PASSWORD", "changeme123")
		if _, err := s.createUser("Organiser", email, pw, roleOrganiser, ""); err != nil {
			return err
		}
		fmt.Printf("Created first organiser account: %s (password from ADMIN_PASSWORD, default %q) - change it!\n", email, "changeme123")
		if env("SEED_DEMO", "") == "1" {
			return s.seedDemo()
		}
	}
	return nil
}

func hashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

func (s *Server) createUser(name, email, password, role, group string) (int64, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO users(name,email,password_hash,role,group_code,created_at) VALUES(?,?,?,?,?,?)`,
		name, strings.ToLower(email), hash, role, group, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getSetting(db *DB, key, def string) string {
	var v string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v); err != nil {
		return def
	}
	return v
}

func setSetting(db *DB, key, value string) {
	db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
}

// touchEntries records that entries changed, so the organiser can be told
// the schedule may be out of date.
func touchEntries(db *DB) { setSetting(db, "entries_changed_at", now()) }
