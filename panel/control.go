package main

// Steuerung über einen lokalen Socket (nur root): Befehle wie
// "serverwerk passwort admin" landen so im laufenden Panel. Früher hat der
// Befehl nur die Datei geändert, und das laufende Panel hat sie beim nächsten
// Speichern mit seinem alten Stand überschrieben.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type controlReq struct {
	Cmd      string `json:"cmd"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type controlResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func controlPath(data string) string { return filepath.Join(data, "steuerung.sock") }

// setUserPassword setzt das Passwort eines Benutzers (Name ohne Groß/klein).
func setUserPassword(st *Store, name, pw string) error {
	if len(pw) < 8 {
		return errors.New("Passwort braucht mindestens 8 Zeichen")
	}
	found := false
	err := st.Update(func(s *State) error {
		for _, u := range s.Users {
			if strings.EqualFold(u.Name, name) {
				u.Salt = newID(16)
				u.Hash = hashPassword(pw, u.Salt)
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Benutzer %s gibt es nicht", name)
		}
		return nil
	})
	return err
}

func (a *App) controlSocket() {
	p := controlPath(a.data)
	os.Remove(p)
	l, err := net.Listen("unix", p)
	if err != nil {
		log.Printf("Steuer-Socket: %v", err)
		return
	}
	os.Chmod(p, 0600)
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		go a.handleControl(c)
	}
}

func (a *App) handleControl(c net.Conn) {
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	var req controlReq
	var resp controlResp
	if err := json.NewDecoder(c).Decode(&req); err != nil {
		resp.Error = "ungültige Anfrage"
	} else {
		switch req.Cmd {
		case "password":
			if err := setUserPassword(a.st, req.Name, req.Password); err != nil {
				resp.Error = err.Error()
			} else {
				resp.OK = true
				log.Printf("Passwort von %s über die Kommandozeile geändert", req.Name)
			}
		default:
			resp.Error = "unbekannter Befehl"
		}
	}
	json.NewEncoder(c).Encode(resp)
}

// sendControl schickt einen Befehl an das laufende Panel.
// running=false heißt: kein Panel erreichbar, der Aufrufer ändert dann die Datei selbst.
func sendControl(data string, req controlReq) (running bool, err error) {
	c, err := net.DialTimeout("unix", controlPath(data), 2*time.Second)
	if err != nil {
		return false, nil
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(15 * time.Second))
	if err := json.NewEncoder(c).Encode(req); err != nil {
		return true, err
	}
	var resp controlResp
	if err := json.NewDecoder(c).Decode(&resp); err != nil {
		return true, fmt.Errorf("keine Antwort vom Panel: %v", err)
	}
	if !resp.OK {
		return true, errors.New(resp.Error)
	}
	return true, nil
}
