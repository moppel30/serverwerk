package main

// Erste Einrichtung: Solange es keinen Benutzer gibt, legt man im Browser
// selbst das Admin-Konto an. Damit niemand anderes im Netz zuvorkommt,
// braucht man dafür den Einrichtungs-Code, den der Installer als Link ausgibt.

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const setupFile = "einrichtung.txt"

func setupCodePath(data string) string { return filepath.Join(data, setupFile) }

// ensureSetupCode legt den Code an, wenn noch kein Benutzer existiert.
func ensureSetupCode(st *Store, data string) {
	n := 0
	st.Read(func(s *State) { n = len(s.Users) })
	p := setupCodePath(data)
	if n > 0 {
		os.Remove(p)
		return
	}
	if b, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(b))) >= 8 {
		log.Printf("Noch kein Konto. Einrichtungs-Code: %s", strings.TrimSpace(string(b)))
		return
	}
	code := randomPassword()[:10]
	os.WriteFile(p, []byte(code+"\n"), 0600)
	log.Printf("Noch kein Konto. Einrichtungs-Code: %s (Link: http://<adresse>:<port>/#/einrichten/%s)", code, code)
}

func (a *App) setupNeeded() bool {
	n := 0
	a.st.Read(func(s *State) { n = len(s.Users) })
	return n == 0
}

func (a *App) setupStatus(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, map[string]bool{"needed": a.setupNeeded()})
}

func (a *App) setupCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Serverwerk") != "1" {
		jsonErr(w, 403, "Anfrage ohne Kennung abgelehnt")
		return
	}
	ip := clientIP(r)
	if loginLimit.blocked(ip, 8, 10*time.Minute) {
		jsonErr(w, 429, "Zu viele Fehlversuche, bitte 10 Minuten warten")
		return
	}
	var in struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	b, err := os.ReadFile(setupCodePath(a.data))
	want := strings.TrimSpace(string(b))
	if !a.setupNeeded() || err != nil || want == "" {
		jsonErr(w, 409, "Das Panel ist schon eingerichtet. Bitte anmelden.")
		return
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(in.Code)), []byte(want)) != 1 {
		loginLimit.fail(ip)
		jsonErr(w, 403, "Einrichtungs-Code stimmt nicht. Er steht am Ende der Installation und in "+setupCodePath(a.data))
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if !validName(in.Name) {
		jsonErr(w, 400, "Name: 2-32 Zeichen, nur Buchstaben, Zahlen, . _ -")
		return
	}
	if len(in.Password) < 8 {
		jsonErr(w, 400, "Passwort braucht mindestens 8 Zeichen")
		return
	}
	u := &User{ID: newID(6), Name: in.Name, Salt: newID(16), Admin: true, Created: time.Now()}
	u.Hash = hashPassword(in.Password, u.Salt)
	err = a.st.Update(func(s *State) error {
		if len(s.Users) > 0 {
			return os.ErrExist
		}
		s.Users = append(s.Users, u)
		return nil
	})
	if err != nil {
		jsonErr(w, 409, "Das Panel ist schon eingerichtet. Bitte anmelden.")
		return
	}
	os.Remove(setupCodePath(a.data))
	os.Remove(filepath.Join(a.data, "erstes-passwort.txt"))
	log.Printf("Erstes Konto %q angelegt", u.Name)
	a.startSession(w, r, u)
	jsonOut(w, userView(u))
}
