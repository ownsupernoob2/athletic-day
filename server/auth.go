package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	roleStaff     = "staff"
	roleOrganiser = "organiser"
	cookieName    = "ath_session"
	sessionTTL    = 14 * 24 * time.Hour
)

type User struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	GroupCode string `json:"group_code"`
	House     string `json:"house"`
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

type ctxKey struct{}

func userFrom(r *http.Request) *User { u, _ := r.Context().Value(ctxKey{}).(*User); return u }

// ---------- JSON helpers ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, 400, "Invalid request")
		return false
	}
	return true
}

func serverErr(w http.ResponseWriter, err error) {
	println("error:", err.Error())
	fail(w, 500, "Something went wrong on the server")
}

// ---------- middleware ----------

// auth returns a wrapper that requires a valid session (and, optionally, one of the roles).
func (s *Server) auth(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u := s.currentUser(r)
			if u == nil {
				fail(w, 401, "Please sign in")
				return
			}
			if len(roles) > 0 {
				ok := false
				for _, role := range roles {
					ok = ok || role == u.Role
				}
				if !ok {
					fail(w, 403, "You don't have access to this")
					return
				}
			}
			next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
		}
	}
}

func hashToken(t string) string { h := sha256.Sum256([]byte(t)); return hex.EncodeToString(h[:]) }

func (s *Server) currentUser(r *http.Request) *User {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	var u User
	var exp string
	err = s.db.QueryRow(`SELECT u.id,u.name,u.email,u.role,u.group_code,s.expires_at
		FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=?`, hashToken(c.Value)).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.GroupCode, &exp)
	if err != nil {
		return nil
	}
	if t, _ := time.Parse(time.RFC3339, exp); time.Now().After(t) {
		s.db.Exec(`DELETE FROM sessions WHERE token_hash=?`, hashToken(c.Value))
		return nil
	}
	u.House = houseOf(u.GroupCode)
	return &u
}

// ---------- login rate limit ----------

type loginLimiter struct {
	mu   sync.Mutex
	fail map[string][]time.Time
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{fail: map[string][]time.Time{}} }

func (l *loginLimiter) blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.fail[key][:0]
	for _, t := range l.fail[key] {
		if time.Since(t) < 15*time.Minute {
			recent = append(recent, t)
		}
	}
	l.fail[key] = recent
	return len(recent) >= 10
}

func (l *loginLimiter) record(key string) {
	l.mu.Lock()
	l.fail[key] = append(l.fail[key], time.Now())
	l.mu.Unlock()
}

// ---------- handlers ----------

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	key := ip + "|" + email
	if s.limit.blocked(key) {
		fail(w, 429, "Too many attempts. Please wait a few minutes and try again.")
		return
	}

	var id int64
	var hash string
	err := s.db.QueryRow(`SELECT id,password_hash FROM users WHERE email=?`, email).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password)) // keep timing similar
		s.limit.record(key)
		fail(w, 401, "Wrong email or password")
		return
	} else if err != nil {
		serverErr(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		s.limit.record(key)
		fail(w, 401, "Wrong email or password")
		return
	}

	raw := make([]byte, 32)
	rand.Read(raw)
	token := hex.EncodeToString(raw)
	exp := time.Now().Add(sessionTTL)
	if _, err := s.db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`,
		hashToken(token), id, exp.UTC().Format(time.RFC3339)); err != nil {
		serverErr(w, err)
		return
	}
	s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now())
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", Expires: exp,
		HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})

	u := User{ID: id}
	if err := s.db.QueryRow(`SELECT name,email,role,group_code FROM users WHERE id=?`, id).Scan(&u.Name, &u.Email, &u.Role, &u.GroupCode); err != nil {
		serverErr(w, err)
		return
	}
	u.House = houseOf(u.GroupCode)
	writeJSON(w, 200, u)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.db.Exec(`DELETE FROM sessions WHERE token_hash=?`, hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, userFrom(r)) }
