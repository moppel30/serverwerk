package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const pbkdfIter = 210000

func hashPassword(pw string, salt string) string {
	k, _ := pbkdf2.Key(sha256.New, pw, []byte(salt), pbkdfIter, 32)
	return hex.EncodeToString(k)
}

func checkPassword(u *User, pw string) bool {
	h := hashPassword(pw, u.Salt)
	return subtle.ConstantTimeCompare([]byte(h), []byte(u.Hash)) == 1
}

func tokenKey(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(h[:])
}

// Login-Bremse: 8 Fehlversuche pro IP in 10 Minuten.
type limiter struct {
	mu sync.Mutex
	m  map[string][]time.Time
}

var loginLimit = &limiter{m: map[string][]time.Time{}}

func (l *limiter) blocked(ip string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	var keep []time.Time
	for _, t := range l.m[ip] {
		if time.Since(t) < window {
			keep = append(keep, t)
		}
	}
	l.m[ip] = keep
	return len(keep) >= max
}

func (l *limiter) fail(ip string) {
	l.mu.Lock()
	l.m[ip] = append(l.m[ip], time.Now())
	l.mu.Unlock()
}

func clientIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		h = r.RemoteAddr
	}
	// Über den Cloudflare Tunnel kommt alles von 127.0.0.1: echte Adresse aus dem Kopf.
	if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		if cf := strings.TrimSpace(r.Header.Get("Cf-Connecting-Ip")); cf != "" {
			return cf
		}
	}
	return h
}

type ctxKey int

const userKey ctxKey = 1

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userKey).(*User)
	return u
}

func (a *App) sessionUser(r *http.Request) *User {
	c, err := r.Cookie("sw_sitzung")
	if err != nil || c.Value == "" {
		return nil
	}
	key := tokenKey(c.Value)
	var uid string
	a.st.Read(func(s *State) {
		if ss, ok := s.Sessions[key]; ok && time.Now().Before(ss.Expires) {
			uid = ss.UserID
		}
	})
	if uid == "" {
		return nil
	}
	return a.st.User(uid)
}

// auth schützt /api: Login nötig, schreibende Anfragen brauchen den Kopf X-Serverwerk.
func (a *App) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("X-Serverwerk") != "1" {
			jsonErr(w, 403, "Anfrage ohne Kennung abgelehnt")
			return
		}
		u := a.sessionUser(r)
		if u == nil {
			jsonErr(w, 401, "Bitte anmelden")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}

// Rechte für Unterbenutzer.
var allPerms = []string{"console", "power", "files", "backups", "schedules", "startup", "settings"}

func can(u *User, srv *Server, perm string) bool {
	if u == nil || srv == nil {
		return false
	}
	if u.Admin || srv.OwnerID == u.ID {
		return true
	}
	for _, su := range srv.Subusers {
		if su.UserID == u.ID {
			if perm == "view" {
				return true
			}
			for _, p := range su.Perms {
				if p == perm {
					return true
				}
			}
		}
	}
	return false
}

func permsOf(u *User, srv *Server) []string {
	if u.Admin || srv.OwnerID == u.ID {
		return append([]string{"owner"}, allPerms...)
	}
	for _, su := range srv.Subusers {
		if su.UserID == u.ID {
			return su.Perms
		}
	}
	return nil
}

func validName(n string) bool {
	if len(n) < 2 || len(n) > 32 {
		return false
	}
	for _, c := range n {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._-", c)) {
			return false
		}
	}
	return true
}
