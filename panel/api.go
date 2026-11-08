package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type App struct {
	st      *Store
	dk      *Docker
	m       *Manager
	cat     *Catalog
	icons   *iconCache
	arch    *archCache
	port    string
	data    string
	version string
}

func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	msg = T(msg)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(v)
}

func ok(w http.ResponseWriter) { jsonOut(w, map[string]bool{"ok": true}) }

func (a *App) routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("GET /api/setup", a.setupStatus)
	mux.HandleFunc("POST /api/setup", a.setupCreate)

	api := http.NewServeMux()
	h := func(p string, f http.HandlerFunc) { api.HandleFunc(p, f) }
	h("GET /api/me", a.me)
	h("POST /api/me/password", a.myPassword)
	h("GET /api/servers", a.serverList)
	h("POST /api/servers", a.admin(a.serverCreate))
	h("GET /api/servers/{id}", a.srv("view", a.serverGet))
	h("PATCH /api/servers/{id}", a.srv("settings", a.serverPatch))
	h("DELETE /api/servers/{id}", a.admin(a.srv("view", a.serverDelete)))
	h("PUT /api/servers/{id}/startup", a.srv("startup", a.serverStartup))
	h("POST /api/servers/{id}/reinstall", a.srv("settings", a.serverReinstall))
	h("POST /api/servers/{id}/power", a.srv("power", a.serverPower))
	h("POST /api/servers/{id}/command", a.srv("console", a.serverCommand))
	h("GET /api/servers/{id}/events", a.srv("view", a.serverEvents))
	h("POST /api/servers/{id}/eula", a.srv("power", a.serverEula))
	h("GET /api/servers/{id}/files", a.srv("files", a.filesList))
	h("GET /api/servers/{id}/files/content", a.srv("files", a.fileRead))
	h("PUT /api/servers/{id}/files/content", a.srv("files", a.fileWrite))
	h("POST /api/servers/{id}/files/upload", a.srv("files", a.fileUpload))
	h("POST /api/servers/{id}/files/chunk", a.srv("files", a.fileChunk))
	h("GET /api/servers/{id}/files/download", a.srv("files", a.fileDownload))
	h("POST /api/servers/{id}/files/rename", a.srv("files", a.fileRename))
	h("POST /api/servers/{id}/files/delete", a.srv("files", a.fileDelete))
	h("POST /api/servers/{id}/files/mkdir", a.srv("files", a.fileMkdir))
	h("POST /api/servers/{id}/files/compress", a.srv("files", a.fileCompress))
	h("POST /api/servers/{id}/files/decompress", a.srv("files", a.fileDecompress))
	h("GET /api/servers/{id}/backups", a.srv("backups", a.backupList))
	h("POST /api/servers/{id}/backups", a.srv("backups", a.backupCreate))
	h("GET /api/servers/{id}/backups/{name}", a.srv("backups", a.backupDownload))
	h("POST /api/servers/{id}/backups/{name}/restore", a.srv("backups", a.backupRestore))
	h("DELETE /api/servers/{id}/backups/{name}", a.srv("backups", a.backupDelete))
	h("PUT /api/servers/{id}/schedules", a.srv("schedules", a.schedulesPut))
	h("POST /api/servers/{id}/schedules/{sid}/run", a.srv("schedules", a.scheduleRun))
	h("PUT /api/servers/{id}/subusers", a.srv("owner", a.subusersPut))
	h("GET /api/usernames", a.userNames)
	h("GET /api/eggs", a.eggList)
	h("GET /api/eggs/{id}", a.eggGet)
	h("GET /api/eggs/{id}/icon", a.eggIcon)
	h("GET /api/catalog/icon", a.admin(a.catalogIcon))
	h("GET /api/servers/{id}/addons", a.srv("files", a.addonsGet))
	h("POST /api/servers/{id}/addons", a.srv("files", a.addonInstall))
	h("POST /api/servers/{id}/addons/remove", a.srv("files", a.addonRemove))
	h("POST /api/eggs", a.admin(a.eggImport))
	h("DELETE /api/eggs/{id}", a.admin(a.eggDelete))
	h("POST /api/eggs/{id}/update", a.admin(a.eggUpdate))
	h("GET /api/catalog", a.admin(a.catalogList))
	h("POST /api/catalog/import", a.admin(a.catalogImport))
	h("GET /api/users", a.admin(a.userList))
	h("POST /api/users", a.admin(a.userCreate))
	h("PATCH /api/users/{id}", a.admin(a.userPatch))
	h("DELETE /api/users/{id}", a.admin(a.userDelete))
	h("GET /api/system", a.admin(a.system))
	h("GET /api/allocations", a.admin(a.allocationsGet))
	h("POST /api/import/pterodactyl/scan", a.admin(a.importScan))
	h("POST /api/import/pterodactyl/run", a.admin(a.importRun))
	h("PUT /api/allocations", a.admin(a.allocationsPut))
	h("GET /api/settings", a.admin(a.settingsGet))
	h("PUT /api/settings", a.admin(a.settingsPut))
	h("POST /api/settings/panel", a.admin(a.panelPublish))
	h("POST /api/servers/{id}/domain", a.admin(a.srv("view", a.serverDomain)))
	h("GET /api/catalog/egg", a.admin(a.catalogEgg))
	h("GET /api/ports/free", a.admin(a.freePorts))
	mux.Handle("/api/", a.auth(api))
}

func (a *App) admin(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := currentUser(r); u == nil || !u.Admin {
			jsonErr(w, 403, "Nur für Admins")
			return
		}
		f(w, r)
	}
}

type srvKey struct{}

func (a *App) srv(perm string, f func(http.ResponseWriter, *http.Request, *Server)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := a.st.Server(r.PathValue("id"))
		u := currentUser(r)
		if s == nil || !can(u, s, "view") {
			jsonErr(w, 404, "Server nicht gefunden")
			return
		}
		if perm == "owner" {
			if !u.Admin && s.OwnerID != u.ID {
				jsonErr(w, 403, "Nur der Besitzer darf das")
				return
			}
		} else if perm != "view" && !can(u, s, perm) {
			jsonErr(w, 403, "Dafür fehlt dir das Recht")
			return
		}
		f(w, r, s)
	}
}

// ---------- Anmeldung ----------

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	var sec SecuritySettings
	a.st.Read(func(s *State) { sec = s.Settings.Security })
	if loginLimit.blocked(ip, max(sec.MaxFails, 1), time.Duration(max(sec.LockMinutes, 1))*time.Minute) {
		jsonErr(w, 429, fmt.Sprintf("Zu viele Fehlversuche, bitte %d Minuten warten", max(sec.LockMinutes, 1)))
		return
	}
	var in struct{ Name, Password string }
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	var u *User
	a.st.Read(func(s *State) {
		for _, x := range s.Users {
			if strings.EqualFold(x.Name, strings.TrimSpace(in.Name)) {
				c := *x
				u = &c
			}
		}
	})
	if u == nil || !checkPassword(u, in.Password) {
		loginLimit.fail(ip)
		jsonErr(w, 401, "Name oder Passwort falsch")
		return
	}
	a.startSession(w, r, u)
	jsonOut(w, userView(u))
}

// startSession meldet u an und setzt das Sitzungs-Cookie.
func (a *App) startSession(w http.ResponseWriter, r *http.Request, u *User) {
	var sec SecuritySettings
	a.st.Read(func(s *State) { sec = s.Settings.Security })
	tok := newID(32)
	exp := time.Now().Add(time.Duration(max(sec.SessionDays, 1)) * 24 * time.Hour)
	a.st.Update(func(s *State) error {
		for k, v := range s.Sessions {
			if time.Now().After(v.Expires) {
				delete(s.Sessions, k)
			}
		}
		s.Sessions[tokenKey(tok)] = &Session{UserID: u.ID, Expires: exp}
		return nil
	})
	http.SetCookie(w, &http.Cookie{Name: "sw_sitzung", Value: tok, Path: "/", Expires: exp, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("sw_sitzung"); err == nil {
		a.st.Update(func(s *State) error { delete(s.Sessions, tokenKey(c.Value)); return nil })
	}
	http.SetCookie(w, &http.Cookie{Name: "sw_sitzung", Value: "", Path: "/", MaxAge: -1})
	ok(w)
}

func userView(u *User) map[string]any {
	return map[string]any{"id": u.ID, "name": u.Name, "admin": u.Admin, "created": u.Created}
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	v := userView(u)
	a.st.Read(func(s *State) { v["panelName"] = s.Settings.PanelName; v["accent"] = s.Settings.Accent })
	v["lang"] = lang()
	v["version"] = a.version
	jsonOut(w, v)
}

func (a *App) myPassword(w http.ResponseWriter, r *http.Request) {
	var in struct{ Old, New string }
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	u := currentUser(r)
	if !checkPassword(u, in.Old) {
		jsonErr(w, 400, "Altes Passwort stimmt nicht")
		return
	}
	if len(in.New) < 8 {
		jsonErr(w, 400, "Neues Passwort braucht mindestens 8 Zeichen")
		return
	}
	a.st.Update(func(s *State) error {
		for _, x := range s.Users {
			if x.ID == u.ID {
				x.Salt = newID(16)
				x.Hash = hashPassword(in.New, x.Salt)
			}
		}
		return nil
	})
	// Das Startpasswort aus der Installation ist jetzt veraltet.
	os.Remove(filepath.Join(a.data, "erstes-passwort.txt"))
	ok(w)
}

// ---------- Server ----------

type serverView struct {
	*Server
	State   string   `json:"state"`
	Stats   Stats    `json:"stats"`
	EggName string   `json:"eggName"`
	Owner   string   `json:"ownerName"`
	Perms   []string `json:"perms"`
	Alias   string   `json:"alias,omitempty"`
}

func (a *App) view(u *User, s *Server) serverView {
	in := a.m.Get(s.ID)
	v := serverView{Server: s, State: in.State(), Stats: in.Stats(), Perms: permsOf(u, s)}
	if len(s.Ports) > 0 {
		v.Alias = a.portAlias(s.Ports[0])
	}
	if e := a.st.Egg(s.EggID); e != nil {
		v.EggName = e.Name
	}
	if o := a.st.User(s.OwnerID); o != nil {
		v.Owner = o.Name
	}
	if !u.Admin && s.OwnerID != u.ID {
		v.Server.Subusers = nil
	}
	return v
}

func (a *App) serverList(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var list []*Server
	a.st.Read(func(s *State) {
		for _, x := range s.Servers {
			if can(u, x, "view") {
				list = append(list, cloneServer(x))
			}
		}
	})
	out := []serverView{}
	for _, s := range list {
		out = append(out, a.view(u, s))
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	jsonOut(w, out)
}

func (a *App) serverGet(w http.ResponseWriter, r *http.Request, s *Server) {
	u := currentUser(r)
	v := a.view(u, s)
	out := map[string]any{"server": v}
	if e := a.st.Egg(s.EggID); e != nil {
		vars := []map[string]any{}
		env := mergedEnv(s, e)
		for _, ev := range e.Variables {
			if !u.Admin && !ev.UserViewable {
				continue
			}
			vars = append(vars, map[string]any{
				"name": ev.Name, "description": ev.Description, "env": ev.Env, "value": env[ev.Env],
				"default": ev.Default, "editable": u.Admin || ev.UserEditable, "rules": ev.Rules, "options": ev.Options(),
			})
		}
		out["variables"] = vars
		out["images"] = e.Images
		out["eggStartup"] = e.Startup
		out["features"] = e.Features
	}
	if info := a.addonInfo(s); info.Kind != "" {
		out["addons"] = info
	}
	jsonOut(w, out)
}

func portInUse(p int) bool {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p))
	if err != nil {
		return true
	}
	l.Close()
	return false
}

func (a *App) usedPorts(except string) map[int]bool {
	used := map[int]bool{}
	a.st.Read(func(s *State) {
		for _, x := range s.Servers {
			if x.ID == except {
				continue
			}
			for _, p := range x.Ports {
				used[p] = true
			}
		}
	})
	return used
}

// suggestPorts nimmt nur Ports aus der freigegebenen Liste (Einstellungen).
func (a *App) suggestPorts(n int) []int {
	allowed := a.allowedPorts()
	used := a.usedPorts("")
	var out []int
	for _, p := range allowed {
		if len(out) >= n {
			break
		}
		if !used[p] && !portInUse(p) {
			out = append(out, p)
		}
	}
	return out
}

func (a *App) allowedPorts() []int {
	var groups []AllocGroup
	a.st.Read(func(st *State) { groups = st.Settings.allocGroups() })
	seen := map[int]bool{}
	var out []int
	for _, g := range groups {
		l, _ := parsePortList(g.Ports)
		for _, p := range l {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func (a *App) freePorts(w http.ResponseWriter, r *http.Request) {
	allowed := a.allowedPorts()
	jsonOut(w, map[string]any{"configured": len(allowed) > 0, "free": a.suggestPorts(8), "total": len(allowed)})
}

// parsePortList liest "25565-25575, 19132" in eine sortierte Liste.
func parsePortList(spec string) ([]int, error) {
	seen := map[int]bool{}
	var out []int
	for _, part := range strings.FieldsFunc(spec, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' }) {
		lo, hi := part, part
		if x, y, ok := strings.Cut(part, "-"); ok {
			lo, hi = x, y
		}
		a, e1 := strconv.Atoi(strings.TrimSpace(lo))
		b, e2 := strconv.Atoi(strings.TrimSpace(hi))
		if e1 != nil || e2 != nil || a < 1 || b > 65535 || a > b {
			return nil, fmt.Errorf("%q ist kein gültiger Port oder Bereich", part)
		}
		if b-a > 5000 || len(out) > 10000 {
			return nil, fmt.Errorf("Bereich %q ist zu groß", part)
		}
		for p := a; p <= b; p++ {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	// Reihenfolge wie eingetragen: der erste Bereich wird zuerst vergeben.
	return out, nil
}

func (a *App) checkPorts(ports []int, except string) error {
	if len(ports) == 0 {
		return fmt.Errorf("mindestens ein Port nötig")
	}
	used := a.usedPorts(except)
	allowed := map[int]bool{}
	for _, p := range a.allowedPorts() {
		allowed[p] = true
	}
	if len(allowed) == 0 {
		return fmt.Errorf("Lege zuerst unter Allocations fest, welche Ports im Router freigegeben sind")
	}
	seen := map[int]bool{}
	for _, p := range ports {
		if p < 1 || p > 65535 {
			return fmt.Errorf("Port %d ist ungültig", p)
		}
		if !allowed[p] {
			return fmt.Errorf("Port %d steht nicht in deinen Allocations", p)
		}
		if used[p] {
			return fmt.Errorf("Port %d nutzt schon ein anderer Server", p)
		}
		if seen[p] {
			return fmt.Errorf("Port %d doppelt", p)
		}
		seen[p] = true
	}
	return nil
}

type serverInput struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	EggID        string            `json:"egg"`
	OwnerID      string            `json:"owner"`
	Memory       int               `json:"memory"`
	CPU          int               `json:"cpu"`
	Disk         int               `json:"disk"`
	Ports        []int             `json:"ports"`
	Image        string            `json:"image"`
	Env          map[string]string `json:"env"`
	AutoRestart  *bool             `json:"autoRestart"`
	BootStart    *bool             `json:"bootStart"`
	MaxBackups   *int              `json:"maxBackups"`
	StartAfter   bool              `json:"startAfter"`
	NoAutoDomain bool              `json:"noAutoDomain"`
}

func (a *App) serverCreate(w http.ResponseWriter, r *http.Request) {
	var in serverInput
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 60 {
		jsonErr(w, 400, "Name fehlt oder ist zu lang")
		return
	}
	egg := a.st.Egg(in.EggID)
	if egg == nil {
		jsonErr(w, 400, "Egg wählen")
		return
	}
	if c := a.eggCompat(egg); c != nil && !*c {
		jsonErr(w, 400, fmt.Sprintf("%s läuft nicht auf diesem Rechner (%s): die Docker-Images gibt es dafür nicht", egg.Name, hostArch()))
		return
	}
	if in.OwnerID == "" {
		in.OwnerID = currentUser(r).ID
	}
	if a.st.User(in.OwnerID) == nil {
		jsonErr(w, 400, "Besitzer gibt es nicht")
		return
	}
	if len(in.Ports) == 0 {
		in.Ports = a.suggestPorts(1)
	}
	if err := a.checkPorts(in.Ports, ""); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	env := egg.DefaultEnv()
	for _, v := range egg.Variables {
		if val, ok := in.Env[v.Env]; ok {
			env[v.Env] = val
		}
		// Paper und Co. wollen einen User-Agent für ihre Download-API.
		if v.Env == "USER_AGENT" && strings.TrimSpace(env[v.Env]) == "" {
			env[v.Env] = "Serverwerk/" + a.version + " (selbst gehostet)"
		}
		if err := ValidateVar(v, env[v.Env]); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
	}
	img := in.Image
	if img == "" {
		img = egg.Images[0].Image
	}
	srv := &Server{
		ID: newUUID(), Name: in.Name, Description: in.Description, OwnerID: in.OwnerID, EggID: egg.ID,
		Image: img, Env: env, Memory: max(in.Memory, 0), CPU: max(in.CPU, 0), Disk: max(in.Disk, 0),
		Ports: in.Ports, AutoRestart: true, MaxBackups: 5, Created: time.Now(),
	}
	a.st.Read(func(st *State) {
		srv.AutoRestart, srv.MaxBackups = st.Settings.Defaults.AutoRestart, max(st.Settings.Defaults.MaxBackups, 1)
	})
	if in.AutoRestart != nil {
		srv.AutoRestart = *in.AutoRestart
	}
	if in.MaxBackups != nil {
		srv.MaxBackups = *in.MaxBackups
	}
	if err := os.MkdirAll(a.m.ServerDir(srv.ID), 0755); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	os.Chown(a.m.ServerDir(srv.ID), containerUID, containerUID)
	a.st.Update(func(s *State) error { s.Servers = append(s.Servers, srv); return nil })
	if !in.NoAutoDomain {
		go a.applyAutoDomain(srv.ID)
	}
	inst := a.m.Get(srv.ID)
	if err := inst.Install(); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	if in.StartAfter {
		go func() {
			for i := 0; i < 3600; i++ {
				time.Sleep(time.Second)
				if s := a.st.Server(srv.ID); s == nil || s.Install == "failed" {
					return
				} else if s.Install == "ok" {
					// Genau einmal: läuft er schon (z. B. über die EULA), nichts tun.
					if inst.State() == StOffline {
						inst.Start()
					}
					return
				}
			}
		}()
	}
	jsonOut(w, srv)
}

func (a *App) serverPatch(w http.ResponseWriter, r *http.Request, s *Server) {
	var in serverInput
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	u := currentUser(r)
	if in.Ports != nil && u.Admin {
		if err := a.checkPorts(in.Ports, s.ID); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
	}
	if in.OwnerID != "" && u.Admin && a.st.User(in.OwnerID) == nil {
		jsonErr(w, 400, "Besitzer gibt es nicht")
		return
	}
	err := a.st.Update(func(st *State) error {
		x := findServer(st, s.ID)
		if x == nil {
			return fmt.Errorf("Server fehlt")
		}
		if n := strings.TrimSpace(in.Name); n != "" {
			x.Name = n
		}
		x.Description = in.Description
		if in.AutoRestart != nil {
			x.AutoRestart = *in.AutoRestart
		}
		if in.BootStart != nil {
			x.NoBootStart = !*in.BootStart
		}
		if u.Admin {
			x.Memory, x.CPU, x.Disk = max(in.Memory, 0), max(in.CPU, 0), max(in.Disk, 0)
			if in.Ports != nil {
				x.Ports = in.Ports
			}
			if in.OwnerID != "" {
				x.OwnerID = in.OwnerID
			}
			if in.MaxBackups != nil {
				x.MaxBackups = *in.MaxBackups
			}
		}
		return nil
	})
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	// Port geändert: SRV-Eintrag nachziehen.
	if in.Ports != nil && u.Admin && s.Domain != nil {
		go func() {
			if x := a.st.Server(s.ID); x != nil {
				a.setDomainStatus(x.ID, a.syncDomain(x))
			}
		}()
	}
	ok(w)
}

func (a *App) serverDelete(w http.ResponseWriter, r *http.Request, s *Server) {
	in := a.m.Get(s.ID)
	ctx := context.Background()
	in.Kill()
	a.removeDomain(s)
	a.dk.Remove(ctx, containerName(s.ID))
	a.dk.Remove(ctx, containerName(s.ID)+"-install")
	if r.URL.Query().Get("dateien") != "behalten" {
		os.RemoveAll(a.m.ServerDir(s.ID))
		os.RemoveAll(a.m.BackupDir(s.ID))
	}
	a.st.Update(func(st *State) error {
		for i, x := range st.Servers {
			if x.ID == s.ID {
				st.Servers = append(st.Servers[:i], st.Servers[i+1:]...)
				break
			}
		}
		return nil
	})
	a.m.Forget(s.ID)
	ok(w)
}

func (a *App) serverStartup(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct {
		Env     map[string]string `json:"env"`
		Image   string            `json:"image"`
		Startup *string           `json:"startup"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	u := currentUser(r)
	egg := a.st.Egg(s.EggID)
	if egg == nil {
		jsonErr(w, 400, "Egg fehlt")
		return
	}
	env := mergedEnv(s, egg)
	for _, v := range egg.Variables {
		val, ok := in.Env[v.Env]
		if !ok || val == env[v.Env] {
			continue
		}
		if !u.Admin && !(v.UserEditable && v.UserViewable) {
			jsonErr(w, 403, v.Name+" darfst du nicht ändern")
			return
		}
		if err := ValidateVar(v, val); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		env[v.Env] = val
	}
	if in.Image != "" {
		allowed := u.Admin
		for _, im := range egg.Images {
			if im.Image == in.Image {
				allowed = true
			}
		}
		if !allowed {
			jsonErr(w, 403, "Dieses Image ist nicht erlaubt")
			return
		}
	}
	a.st.Update(func(st *State) error {
		x := findServer(st, s.ID)
		if x == nil {
			return nil
		}
		x.Env = env
		if in.Image != "" {
			x.Image = in.Image
		}
		if in.Startup != nil && u.Admin {
			x.Startup = strings.TrimSpace(*in.Startup)
			if x.Startup == egg.Startup {
				x.Startup = ""
			}
		}
		return nil
	})
	ok(w)
}

func (a *App) serverReinstall(w http.ResponseWriter, r *http.Request, s *Server) {
	if err := a.m.Get(s.ID).Install(); err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	ok(w)
}

func (a *App) serverPower(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct{ Action string }
	readJSON(r, &in)
	if err := a.m.Get(s.ID).Power(in.Action); err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	ok(w)
}

func (a *App) serverCommand(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct{ Command string }
	readJSON(r, &in)
	if strings.TrimSpace(in.Command) == "" {
		jsonErr(w, 400, "Befehl fehlt")
		return
	}
	if err := a.m.Get(s.ID).SendCommand(in.Command); err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	ok(w)
}

func (a *App) serverEula(w http.ResponseWriter, r *http.Request, s *Server) {
	p := filepath.Join(a.m.ServerDir(s.ID), "eula.txt")
	if err := os.WriteFile(p, []byte("# Akzeptiert über Serverwerk\neula=true\n"), 0644); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	os.Lchown(p, containerUID, containerUID)
	in := a.m.Get(s.ID)
	in.setHint("")
	in.Say("EULA akzeptiert.")
	if in.State() == StOffline {
		in.Start()
	}
	ok(w)
}

func (a *App) serverEvents(w http.ResponseWriter, r *http.Request, s *Server) {
	fl, okf := w.(http.Flusher)
	if !okf {
		jsonErr(w, 500, "Streaming nicht möglich")
		return
	}
	u := currentUser(r)
	console := can(u, s, "console")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	in := a.m.Get(s.ID)
	ch, lines, state, stats, hint := in.Subscribe()
	defer in.Unsubscribe(ch)
	if !console {
		lines = nil
	}
	send := func(e Event) {
		b, _ := json.Marshal(e)
		fmt.Fprintf(w, "data: %s\n\n", b)
	}
	st := in.Stats()
	if st.Uptime == 0 {
		st = stats
	}
	send(Event{"init", map[string]any{"lines": lines, "state": state, "stats": st, "hint": hint}})
	fl.Flush()
	ka := time.NewTicker(20 * time.Second)
	defer ka.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-ch:
			if e.T == "line" && !console {
				continue
			}
			send(e)
			// Mehrere Zeilen am Stück zusammen abschicken.
			for n := 0; n < 200; n++ {
				select {
				case e2 := <-ch:
					if e2.T == "line" && !console {
						continue
					}
					send(e2)
					continue
				default:
				}
				break
			}
			fl.Flush()
		case <-ka.C:
			fmt.Fprint(w, ": ka\n\n")
			fl.Flush()
		}
	}
}

// ---------- Dateien ----------

func (a *App) root(s *Server) string { return a.m.ServerDir(s.ID) }

func (a *App) denied(s *Server, rel string) bool {
	egg := a.st.Egg(s.EggID)
	if egg == nil {
		return false
	}
	rel = strings.TrimPrefix(path.Clean("/"+rel), "/")
	return ignored(rel, false, egg.FileDenylist)
}

func (a *App) filesList(w http.ResponseWriter, r *http.Request, s *Server) {
	list, err := listDir(a.root(s), r.URL.Query().Get("path"))
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonOut(w, list)
}

func (a *App) fileRead(w http.ResponseWriter, r *http.Request, s *Server) {
	rel := r.URL.Query().Get("path")
	if a.denied(s, rel) {
		jsonErr(w, 403, "Diese Datei ist gesperrt")
		return
	}
	p, err := safeJoin(a.root(s), rel)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		jsonErr(w, 404, "Datei nicht gefunden")
		return
	}
	if fi.Size() > 8<<20 {
		jsonErr(w, 413, "Datei ist zu groß für den Editor (über 8 MB)")
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	if isBinary(b) {
		jsonErr(w, 415, "Das ist keine Textdatei")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}

func (a *App) fileWrite(w http.ResponseWriter, r *http.Request, s *Server) {
	rel := r.URL.Query().Get("path")
	if a.denied(s, rel) {
		jsonErr(w, 403, "Diese Datei ist gesperrt")
		return
	}
	p, err := safeJoin(a.root(s), rel)
	if err != nil || p == a.root(s) {
		jsonErr(w, 400, "ungültiger Pfad")
		return
	}
	os.MkdirAll(filepath.Dir(p), 0755)
	f, err := os.CreateTemp(filepath.Dir(p), ".sw-*")
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	_, err = io.Copy(f, io.LimitReader(r.Body, 64<<20))
	f.Close()
	if err != nil {
		os.Remove(f.Name())
		jsonErr(w, 500, err.Error())
		return
	}
	mode := os.FileMode(0644)
	if fi, err := os.Stat(p); err == nil {
		mode = fi.Mode().Perm()
	}
	os.Chmod(f.Name(), mode)
	if err := os.Rename(f.Name(), p); err != nil {
		os.Remove(f.Name())
		jsonErr(w, 500, err.Error())
		return
	}
	os.Lchown(p, containerUID, containerUID)
	ok(w)
}

func (a *App) fileUpload(w http.ResponseWriter, r *http.Request, s *Server) {
	dir, err := safeJoin(a.root(s), r.URL.Query().Get("path"))
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		jsonErr(w, 400, "Upload erwartet")
		return
	}
	n := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		name := part.FileName()
		if name == "" {
			continue
		}
		// Ordner-Uploads schicken relative Pfade mit.
		rel, _ := filepath.Rel(a.root(s), dir)
		target, err := safeJoin(a.root(s), path.Join(filepath.ToSlash(rel), strings.ReplaceAll(name, "\\", "/")))
		if err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		f, err := os.Create(target)
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		_, err = io.Copy(f, part)
		f.Close()
		os.Lchown(target, containerUID, containerUID)
		os.Lchown(filepath.Dir(target), containerUID, containerUID)
		if err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		n++
	}
	jsonOut(w, map[string]int{"files": n})
}

func (a *App) fileDownload(w http.ResponseWriter, r *http.Request, s *Server) {
	rel := r.URL.Query().Get("path")
	if a.denied(s, rel) {
		jsonErr(w, 403, "Diese Datei ist gesperrt")
		return
	}
	p, err := safeJoin(a.root(s), rel)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	fi, err := os.Stat(p)
	if err != nil {
		jsonErr(w, 404, "nicht gefunden")
		return
	}
	if fi.IsDir() {
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fi.Name()+".tar.gz"))
		tarDir(p, w, nil)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fi.Name()))
	http.ServeFile(w, r, p)
}

func (a *App) fileRename(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct{ From, To string }
	readJSON(r, &in)
	from, err1 := safeJoin(a.root(s), in.From)
	to, err2 := safeJoin(a.root(s), in.To)
	if err1 != nil || err2 != nil || from == a.root(s) {
		jsonErr(w, 400, "ungültiger Pfad")
		return
	}
	if _, err := os.Lstat(to); err == nil {
		jsonErr(w, 409, "Ziel gibt es schon")
		return
	}
	os.MkdirAll(filepath.Dir(to), 0755)
	if err := os.Rename(from, to); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	ok(w)
}

func (a *App) fileDelete(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct {
		Path  string   `json:"path"`
		Names []string `json:"names"`
	}
	readJSON(r, &in)
	for _, n := range in.Names {
		// Der Eintrag selbst darf ein Symlink sein, gelöscht wird dann nur der Link.
		dir, err := safeJoin(a.root(s), in.Path)
		if err != nil || strings.ContainsAny(n, "/\\") || n == ".." || n == "." || n == "" {
			jsonErr(w, 400, "ungültiger Pfad")
			return
		}
		os.RemoveAll(filepath.Join(dir, n))
	}
	ok(w)
}

func (a *App) fileMkdir(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct{ Path string }
	readJSON(r, &in)
	p, err := safeJoin(a.root(s), in.Path)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	os.Lchown(p, containerUID, containerUID)
	ok(w)
}

func (a *App) fileCompress(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct {
		Path  string   `json:"path"`
		Names []string `json:"names"`
	}
	readJSON(r, &in)
	dir, err := safeJoin(a.root(s), in.Path)
	if err != nil || len(in.Names) == 0 {
		jsonErr(w, 400, "nichts ausgewählt")
		return
	}
	name := "archiv-" + time.Now().Format("2006-01-02_15-04-05") + ".zip"
	if len(in.Names) == 1 {
		name = in.Names[0] + ".zip"
	}
	dest := filepath.Join(dir, name)
	if err := compressZip(a.root(s), in.Path, in.Names, dest); err != nil {
		os.Remove(dest)
		jsonErr(w, 500, err.Error())
		return
	}
	os.Lchown(dest, containerUID, containerUID)
	jsonOut(w, map[string]string{"name": name})
}

func (a *App) fileDecompress(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct{ Path, Name string }
	readJSON(r, &in)
	if err := extractArchive(a.root(s), path.Join(in.Path, in.Name), in.Path); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	ok(w)
}

// ---------- Backups ----------

func (a *App) backupList(w http.ResponseWriter, r *http.Request, s *Server) {
	jsonOut(w, map[string]any{"backups": a.listBackups(s.ID), "max": s.MaxBackups})
}

func (a *App) backupCreate(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct{ Note string }
	readJSON(r, &in)
	go a.makeBackup(s.ID, in.Note)
	ok(w)
}

func (a *App) backupDownload(w http.ResponseWriter, r *http.Request, s *Server) {
	p, err := a.backupPath(s.ID, r.PathValue("name"))
	if err != nil {
		jsonErr(w, 404, err.Error())
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", s.Name+"_"+filepath.Base(p)))
	http.ServeFile(w, r, p)
}

func (a *App) backupRestore(w http.ResponseWriter, r *http.Request, s *Server) {
	if err := a.restoreBackup(s.ID, r.PathValue("name")); err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	ok(w)
}

func (a *App) backupDelete(w http.ResponseWriter, r *http.Request, s *Server) {
	if err := a.deleteBackup(s.ID, r.PathValue("name")); err != nil {
		jsonErr(w, 404, err.Error())
		return
	}
	ok(w)
}

// ---------- Zeitpläne & Unterbenutzer ----------

func (a *App) schedulesPut(w http.ResponseWriter, r *http.Request, s *Server) {
	var in []Schedule
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	for i := range in {
		if _, err := ParseCron(in[i].Cron); err != nil {
			jsonErr(w, 400, fmt.Sprintf("%s: %v", in[i].Name, err))
			return
		}
		if in[i].ID == "" {
			in[i].ID = newID(6)
		}
		for _, t := range in[i].Tasks {
			if t.Action != "command" && t.Action != "power" && t.Action != "backup" {
				jsonErr(w, 400, "unbekannte Aufgabe")
				return
			}
		}
	}
	a.st.Update(func(st *State) error {
		if x := findServer(st, s.ID); x != nil {
			old := map[string]*time.Time{}
			for _, sc := range x.Schedules {
				old[sc.ID] = sc.LastRun
			}
			for i := range in {
				in[i].LastRun = old[in[i].ID]
			}
			x.Schedules = in
		}
		return nil
	})
	jsonOut(w, in)
}

func (a *App) scheduleRun(w http.ResponseWriter, r *http.Request, s *Server) {
	for _, sc := range s.Schedules {
		if sc.ID == r.PathValue("sid") {
			go a.runSchedule(s.ID, sc)
			ok(w)
			return
		}
	}
	jsonErr(w, 404, "Zeitplan nicht gefunden")
}

func (a *App) subusersPut(w http.ResponseWriter, r *http.Request, s *Server) {
	var in []Subuser
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	valid := map[string]bool{}
	for _, p := range allPerms {
		valid[p] = true
	}
	for i := range in {
		if a.st.User(in[i].UserID) == nil || in[i].UserID == s.OwnerID {
			jsonErr(w, 400, "unbekannter Benutzer")
			return
		}
		var keep []string
		for _, p := range in[i].Perms {
			if valid[p] {
				keep = append(keep, p)
			}
		}
		in[i].Perms = keep
	}
	a.st.Update(func(st *State) error {
		if x := findServer(st, s.ID); x != nil {
			x.Subusers = in
		}
		return nil
	})
	ok(w)
}

func (a *App) userNames(w http.ResponseWriter, r *http.Request) {
	out := []map[string]string{}
	a.st.Read(func(s *State) {
		for _, u := range s.Users {
			out = append(out, map[string]string{"id": u.ID, "name": u.Name})
		}
	})
	jsonOut(w, out)
}

// ---------- Eggs ----------

func eggSummary(e *Egg, uses int) map[string]any {
	return map[string]any{
		"id": e.ID, "name": e.Name, "author": e.Author, "description": e.Description, "category": e.Category,
		"format": e.Format, "features": e.Features, "images": e.Images, "updateUrl": e.UpdateURL, "imported": e.Imported,
		"variables": e.Variables, "startup": e.Startup, "servers": uses, "tags": e.Tags,
	}
}

func (a *App) eggList(w http.ResponseWriter, r *http.Request) {
	out := []map[string]any{}
	a.st.Read(func(s *State) {
		for _, e := range s.Eggs {
			n := 0
			for _, x := range s.Servers {
				if x.EggID == e.ID {
					n++
				}
			}
			v := eggSummary(e, n)
			v["compatible"] = a.eggCompat(e)
			out = append(out, v)
		}
	})
	sort.Slice(out, func(i, j int) bool {
		ci, cj := out[i]["category"].(string), out[j]["category"].(string)
		if ci != cj {
			return ci < cj
		}
		return strings.ToLower(out[i]["name"].(string)) < strings.ToLower(out[j]["name"].(string))
	})
	jsonOut(w, map[string]any{"eggs": out, "arch": hostArch()})
}

func (a *App) eggGet(w http.ResponseWriter, r *http.Request) {
	e := a.st.Egg(r.PathValue("id"))
	if e == nil {
		jsonErr(w, 404, "Egg nicht gefunden")
		return
	}
	jsonOut(w, struct {
		*Egg
		Compatible *bool  `json:"compatible"`
		Arch       string `json:"arch"`
	}{e, a.eggCompat(e), hostArch()})
}

func (a *App) saveEgg(e *Egg, replaceID string) *Egg {
	e.Imported = time.Now()
	defer func() {
		// Architektur des neuen Eggs gleich prüfen.
		go func(e Egg) {
			var imgs []string
			for _, im := range e.Images {
				imgs = append(imgs, im.Image)
			}
			a.compatOf(imgs, e.InstImage, strings.TrimSpace(e.InstScript) != "")
			a.arch.save()
		}(*e)
	}()
	a.st.Update(func(s *State) error {
		if replaceID != "" {
			for i, x := range s.Eggs {
				if x.ID == replaceID {
					e.ID = x.ID
					if e.Category == "" {
						e.Category = x.Category
					}
					s.Eggs[i] = e
					return nil
				}
			}
		}
		// Gleiches Egg (Name + Kategorie) schon da? Dann ersetzen statt doppeln.
		for i, x := range s.Eggs {
			if strings.EqualFold(x.Name, e.Name) && x.Category == e.Category {
				e.ID = x.ID
				s.Eggs[i] = e
				return nil
			}
		}
		e.ID = newID(6)
		s.Eggs = append(s.Eggs, e)
		return nil
	})
	return e
}

func (a *App) eggImport(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	var e *Egg
	var src string
	var asURL struct {
		URL      string `json:"url"`
		Category string `json:"category"`
	}
	if json.Unmarshal(b, &asURL) == nil && asURL.URL != "" {
		e, src, err = FetchEgg(urlVariants(asURL.URL)...)
	} else {
		e, err = ParseEgg(b)
		src = "Datei"
	}
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	e.Source = src
	e.Category = firstNonEmpty(asURL.Category, categoryFromTags(e), "Eigene")
	jsonOut(w, eggSummary(a.saveEgg(e, ""), 0))
}

func categoryFromTags(e *Egg) string {
	for _, t := range e.Tags {
		if strings.EqualFold(t, "minecraft") {
			return "Minecraft"
		}
	}
	return ""
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}

// urlVariants: Der Katalog-Index hinkt manchmal hinter Umbenennungen her.
func urlVariants(u string) []string {
	out := []string{u}
	dir, file := path.Split(u)
	if strings.HasPrefix(file, "egg-pterodactyl-") {
		out = append(out, dir+"pterodactyl-egg-"+strings.TrimPrefix(file, "egg-pterodactyl-"))
	}
	if strings.HasSuffix(file, ".json") && !strings.Contains(file, "pterodactyl") {
		out = append(out, strings.TrimSuffix(u, ".json")+".yaml")
	}
	if strings.HasSuffix(file, ".yaml") {
		out = append(out, strings.TrimSuffix(u, ".yaml")+".json")
	}
	return out
}

func (a *App) eggDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.st.Update(func(s *State) error {
		for _, x := range s.Servers {
			if x.EggID == id {
				return fmt.Errorf("Egg wird noch von %s benutzt", x.Name)
			}
		}
		for i, e := range s.Eggs {
			if e.ID == id {
				s.Eggs = append(s.Eggs[:i], s.Eggs[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("Egg nicht gefunden")
	})
	if err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	ok(w)
}

func (a *App) eggUpdate(w http.ResponseWriter, r *http.Request) {
	old := a.st.Egg(r.PathValue("id"))
	if old == nil {
		jsonErr(w, 404, "Egg nicht gefunden")
		return
	}
	urls := []string{}
	if strings.HasPrefix(old.Source, "https://") {
		urls = append(urls, urlVariants(old.Source)...)
	}
	if old.UpdateURL != "" {
		urls = append(urls, urlVariants(old.UpdateURL)...)
	}
	if len(urls) == 0 {
		jsonErr(w, 400, "Egg hat keine Update-Adresse")
		return
	}
	e, src, err := FetchEgg(urls...)
	if err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	e.Source = src
	e.Category = old.Category
	jsonOut(w, eggSummary(a.saveEgg(e, old.ID), 0))
}

func (a *App) catalogList(w http.ResponseWriter, r *http.Request) {
	list, fetched := a.cat.List()
	if r.URL.Query().Get("refresh") == "1" || len(list) == 0 || time.Since(fetched) > 24*time.Hour {
		if err := a.cat.Refresh(); err != nil && len(list) == 0 {
			jsonErr(w, 502, err.Error())
			return
		}
		list, fetched = a.cat.List()
		go a.archScan()
	}
	installed := map[string]string{}
	a.st.Read(func(s *State) {
		for _, e := range s.Eggs {
			installed[strings.ToLower(e.Category+"|"+e.Name)] = e.ID
		}
	})
	type row struct {
		CatalogEgg
		Installed  string `json:"installed,omitempty"`
		Compatible *bool  `json:"compatible"`
	}
	out := make([]row, 0, len(list))
	for _, c := range list {
		out = append(out, row{c, installed[strings.ToLower(c.Category+"|"+c.Name)], a.catalogCompat(c.URL)})
	}
	jsonOut(w, map[string]any{"eggs": out, "fetched": fetched, "arch": a.archStatus()})
}

func (a *App) catalogImport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL      string `json:"url"`
		AltURL   string `json:"altUrl"`
		Category string `json:"category"`
	}
	if readJSON(r, &in) != nil || in.URL == "" {
		jsonErr(w, 400, "Adresse fehlt")
		return
	}
	urls := append(urlVariants(in.URL), urlVariants(in.AltURL)...)
	if in.AltURL == "" {
		urls = urlVariants(in.URL)
	}
	e, src, err := FetchEgg(urls...)
	if err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	e.Source = src
	e.Category = firstNonEmpty(in.Category, "Eigene")
	jsonOut(w, eggSummary(a.saveEgg(e, ""), 0))
}

// ---------- Benutzer ----------

func (a *App) userList(w http.ResponseWriter, r *http.Request) {
	out := []map[string]any{}
	a.st.Read(func(s *State) {
		for _, u := range s.Users {
			v := userView(u)
			n := 0
			for _, x := range s.Servers {
				if x.OwnerID == u.ID {
					n++
				}
			}
			v["servers"] = n
			out = append(out, v)
		}
	})
	jsonOut(w, out)
}

func (a *App) userCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Admin    bool   `json:"admin"`
	}
	readJSON(r, &in)
	in.Name = strings.TrimSpace(in.Name)
	if !validName(in.Name) {
		jsonErr(w, 400, "Name: 2-32 Zeichen, nur Buchstaben, Zahlen, . _ -")
		return
	}
	if len(in.Password) < 8 {
		jsonErr(w, 400, "Passwort braucht mindestens 8 Zeichen")
		return
	}
	u := &User{ID: newID(6), Name: in.Name, Salt: newID(16), Admin: in.Admin, Created: time.Now()}
	u.Hash = hashPassword(in.Password, u.Salt)
	err := a.st.Update(func(s *State) error {
		for _, x := range s.Users {
			if strings.EqualFold(x.Name, u.Name) {
				return fmt.Errorf("Name ist schon vergeben")
			}
		}
		s.Users = append(s.Users, u)
		return nil
	})
	if err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	jsonOut(w, userView(u))
}

func (a *App) userPatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
		Admin    *bool  `json:"admin"`
	}
	readJSON(r, &in)
	id := r.PathValue("id")
	me := currentUser(r)
	err := a.st.Update(func(s *State) error {
		for _, u := range s.Users {
			if u.ID != id {
				continue
			}
			if in.Password != "" {
				if len(in.Password) < 8 {
					return fmt.Errorf("Passwort braucht mindestens 8 Zeichen")
				}
				u.Salt = newID(16)
				u.Hash = hashPassword(in.Password, u.Salt)
				for k, ss := range s.Sessions {
					if ss.UserID == u.ID {
						delete(s.Sessions, k)
					}
				}
			}
			if in.Admin != nil {
				if u.ID == me.ID && !*in.Admin {
					return fmt.Errorf("Du kannst dir die Admin-Rechte nicht selbst nehmen")
				}
				u.Admin = *in.Admin
			}
			return nil
		}
		return fmt.Errorf("Benutzer nicht gefunden")
	})
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	ok(w)
}

func (a *App) userDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == currentUser(r).ID {
		jsonErr(w, 400, "Du kannst dich nicht selbst löschen")
		return
	}
	err := a.st.Update(func(s *State) error {
		for _, x := range s.Servers {
			if x.OwnerID == id {
				return fmt.Errorf("Benutzer besitzt noch %s", x.Name)
			}
		}
		for i, u := range s.Users {
			if u.ID == id {
				s.Users = append(s.Users[:i], s.Users[i+1:]...)
				for k, ss := range s.Sessions {
					if ss.UserID == id {
						delete(s.Sessions, k)
					}
				}
				for _, x := range s.Servers {
					var keep []Subuser
					for _, su := range x.Subusers {
						if su.UserID != id {
							keep = append(keep, su)
						}
					}
					x.Subusers = keep
				}
				return nil
			}
		}
		return fmt.Errorf("Benutzer nicht gefunden")
	})
	if err != nil {
		jsonErr(w, 409, err.Error())
		return
	}
	ok(w)
}

// ---------- System ----------

func (a *App) system(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	out := map[string]any{"version": a.version, "data": a.data, "os": runtime.GOOS, "arch": runtime.GOARCH}
	if v, err := a.dk.Version(ctx); err == nil {
		out["docker"] = v["Version"]
	} else {
		out["dockerError"] = err.Error()
	}
	if info, err := a.dk.Info(ctx); err == nil {
		out["os"] = info["OperatingSystem"]
		out["kernel"] = info["KernelVersion"]
		out["cpus"] = info["NCPU"]
		out["memTotal"] = info["MemTotal"]
		out["hostname"] = info["Name"]
	}
	out["host"] = hostStats(a.data)
	jsonOut(w, out)
}
