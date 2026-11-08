package main

// Gespeicherter Zustand: Benutzer, Sitzungen, Eggs, Server.
// Alles liegt in einer JSON-Datei. Für einen Rechner mit ein paar Dutzend
// Servern reicht das locker und spart eine Datenbank.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type User struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Hash    string    `json:"hash"`
	Salt    string    `json:"salt"`
	Admin   bool      `json:"admin"`
	Created time.Time `json:"created"`
}

type Session struct {
	UserID  string    `json:"user"`
	Expires time.Time `json:"expires"`
}

type Subuser struct {
	UserID string   `json:"user"`
	Perms  []string `json:"perms"`
}

type Task struct {
	Action  string `json:"action"`  // command | power | backup
	Payload string `json:"payload"` // Befehl oder start/stop/restart/kill
	Delay   int    `json:"delay"`   // Sekunden Wartezeit vor dieser Aufgabe
}

type Schedule struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Cron     string     `json:"cron"`
	Enabled  bool       `json:"enabled"`
	OnlyIfOn bool       `json:"onlyIfOn"`
	Tasks    []Task     `json:"tasks"`
	LastRun  *time.Time `json:"lastRun,omitempty"`
}

type Server struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	OwnerID       string            `json:"owner"`
	EggID         string            `json:"egg"`
	Image         string            `json:"image"`
	Startup       string            `json:"startup"`
	Env           map[string]string `json:"env"`
	Memory        int               `json:"memory"` // MB, 0 = unbegrenzt
	CPU           int               `json:"cpu"`    // Prozent eines Kerns, 0 = unbegrenzt
	Disk          int               `json:"disk"`   // MB, 0 = unbegrenzt
	Ports         []int             `json:"ports"`
	AutoRestart   bool              `json:"autoRestart"`
	MaxBackups    int               `json:"maxBackups"`
	Install       string            `json:"install"` // "", installing, ok, failed
	Subusers      []Subuser         `json:"subusers"`
	Schedules     []Schedule        `json:"schedules"`
	Addons        []Addon           `json:"addons"`
	Domain        *Domain           `json:"domain,omitempty"`
	ImportedFrom  string            `json:"importedFrom,omitempty"`
	ImportedPorts []int             `json:"importedPorts,omitempty"` // Ports in Pterodactyl
	WantRunning   bool              `json:"wantRunning"`             // soll laufen: nach Neustart des Rechners wieder starten
	NoBootStart   bool              `json:"noBootStart"`             // Ausnahme: nach Neustart nicht automatisch starten
	Created       time.Time         `json:"created"`
}

type State struct {
	Users    []*User             `json:"users"`
	Sessions map[string]*Session `json:"sessions"`
	Eggs     []*Egg              `json:"eggs"`
	Servers  []*Server           `json:"servers"`
	Settings Settings            `json:"settings"`
}

type Settings struct {
	PanelName   string           `json:"panelName"`
	Timezone    string           `json:"timezone"`
	Accent      string           `json:"accent"`
	CFToken     string           `json:"cfToken"`
	CFZones     []CFZone         `json:"cfZones"`
	CFTarget    string           `json:"cfTarget"` // ip | host
	CFHost      string           `json:"cfHost"`
	Panel       *PanelPublic     `json:"panel,omitempty"`
	Ports       string           `json:"ports"` // alt, wird in Allocations übernommen
	Allocations []AllocGroup     `json:"allocations"`
	Defaults    ServerDefaults   `json:"defaults"`
	Security    SecuritySettings `json:"security"`
	Shims       []ShimRoute      `json:"shims,omitempty"` // im Router freigegebene Ports, z. B. "25565-25575, 19132"
}

// Standardwerte für neue Server.
type ServerDefaults struct {
	Init        bool `json:"init"`
	Memory      int  `json:"memory"`
	CPU         int  `json:"cpu"`
	Disk        int  `json:"disk"`
	MaxBackups  int  `json:"maxBackups"`
	AutoRestart bool `json:"autoRestart"`
	StartAfter  bool `json:"startAfter"`
}

type SecuritySettings struct {
	SessionDays int `json:"sessionDays"`
	MaxFails    int `json:"maxFails"`
	LockMinutes int `json:"lockMinutes"`
}

// Umweg-Route zu einer macvtap-VM auf diesem Rechner.
type ShimRoute struct {
	IP     string `json:"ip"`
	Parent string `json:"parent"`
}

type CFZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	S    State
}

func newID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func openStore(dir string) (*Store, error) {
	st := &Store{path: filepath.Join(dir, "zustand.json")}
	b, err := os.ReadFile(st.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &st.S); err != nil {
			return nil, err
		}
	}
	if st.S.Sessions == nil {
		st.S.Sessions = map[string]*Session{}
	}
	if st.S.Settings.PanelName == "" {
		st.S.Settings.PanelName = "Serverwerk"
	}
	if d := &st.S.Settings.Defaults; !d.Init {
		*d = ServerDefaults{Init: true, Memory: 2048, MaxBackups: 5, AutoRestart: true, StartAfter: true}
	}
	if s := &st.S.Settings.Security; s.SessionDays == 0 {
		*s = SecuritySettings{SessionDays: 30, MaxFails: 8, LockMinutes: 10}
	}
	for _, s := range st.S.Servers {
		if s.Env == nil {
			s.Env = map[string]string{}
		}
	}
	return st, nil
}

// Save muss mit gehaltener Sperre (Lesen reicht) aufgerufen werden.
func (st *Store) saveLocked() error {
	b, err := json.MarshalIndent(&st.S, "", "  ")
	if err != nil {
		return err
	}
	tmp := st.path + ".neu"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	f.Sync()
	f.Close()
	return os.Rename(tmp, st.path)
}

// Update ändert den Zustand unter Schreibsperre und speichert danach.
func (st *Store) Update(fn func(s *State) error) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if err := fn(&st.S); err != nil {
		return err
	}
	return st.saveLocked()
}

func (st *Store) Read(fn func(s *State)) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	fn(&st.S)
}

// Kopien, damit Aufrufer ohne Sperre damit arbeiten können.
func (st *Store) Server(id string) *Server {
	st.mu.RLock()
	defer st.mu.RUnlock()
	for _, s := range st.S.Servers {
		if s.ID == id {
			return cloneServer(s)
		}
	}
	return nil
}

func cloneServer(s *Server) *Server {
	b, _ := json.Marshal(s)
	var c Server
	json.Unmarshal(b, &c)
	if c.Env == nil {
		c.Env = map[string]string{}
	}
	return &c
}

func (st *Store) Egg(id string) *Egg {
	st.mu.RLock()
	defer st.mu.RUnlock()
	for _, e := range st.S.Eggs {
		if e.ID == id {
			c := *e
			return &c
		}
	}
	return nil
}

func (st *Store) User(id string) *User {
	st.mu.RLock()
	defer st.mu.RUnlock()
	for _, u := range st.S.Users {
		if u.ID == id {
			c := *u
			return &c
		}
	}
	return nil
}

func findServer(s *State, id string) *Server {
	for _, x := range s.Servers {
		if x.ID == id {
			return x
		}
	}
	return nil
}
