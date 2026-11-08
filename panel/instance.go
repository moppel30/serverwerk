package main

// Laufzeit der Server: Installieren, Starten, Stoppen, Konsole, Werte.
// Jeder Server läuft in einem eigenen Docker-Container, wie bei Wings.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const containerUID = 988

const (
	StOffline    = "offline"
	StStarting   = "starting"
	StRunning    = "running"
	StStopping   = "stopping"
	StInstalling = "installing"
)

type Event struct {
	T string `json:"t"`
	D any    `json:"d"`
}

type Manager struct {
	st      *Store
	dk      *Docker
	data    string
	mu      sync.Mutex
	inst    map[string]*Instance
	gateway string
}

type Instance struct {
	m  *Manager
	id string

	mu        sync.Mutex
	state     string
	lines     []string
	partial   []byte
	flushT    *time.Timer
	subs      map[chan Event]struct{}
	att       *Attachment
	container string
	stopping  bool
	started   time.Time
	stats     Stats
	cancel    context.CancelFunc
	hint      string // eula, java
	busy      bool   // Start/Stopp/Installation läuft gerade
	powerLock sync.Mutex
}

func newManager(st *Store, dk *Docker, data string) *Manager {
	return &Manager{st: st, dk: dk, data: data, inst: map[string]*Instance{}}
}

func (m *Manager) ServerDir(id string) string { return filepath.Join(m.data, "server", id) }
func (m *Manager) BackupDir(id string) string { return filepath.Join(m.data, "backups", id) }

func (m *Manager) Get(id string) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	in, ok := m.inst[id]
	if !ok {
		in = &Instance{m: m, id: id, state: StOffline, subs: map[chan Event]struct{}{}}
		m.inst[id] = in
	}
	return in
}

func (m *Manager) Forget(id string) {
	m.mu.Lock()
	delete(m.inst, id)
	m.mu.Unlock()
}

func containerName(id string) string { return "serverwerk-" + id[:8] }

// Restore hängt sich nach einem Panel-Neustart wieder an laufende Container.
func (m *Manager) Restore() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	m.gateway = m.dk.BridgeGateway(ctx)
	list, err := m.dk.ListByLabel(ctx, "serverwerk.server")
	if err != nil {
		log.Printf("Docker: %v", err)
		return
	}
	for _, c := range list {
		id := c.Labels["serverwerk.server"]
		if m.st.Server(id) == nil {
			continue
		}
		if c.Labels["serverwerk.role"] == "install" {
			m.dk.Remove(ctx, c.ID)
			m.st.Update(func(s *State) error {
				if x := findServer(s, id); x != nil && x.Install == "installing" {
					x.Install = "failed"
				}
				return nil
			})
			continue
		}
		if c.State != "running" {
			// Letzte Ausgabe eines gestoppten Servers trotzdem zeigen.
			if rc, err := m.dk.Logs(ctx, c.ID, false, 200); err == nil {
				b, _ := io.ReadAll(io.LimitReader(rc, 1<<20))
				rc.Close()
				in := m.Get(id)
				in.feed(b)
				in.flush()
			}
			continue
		}
		in := m.Get(id)
		in.container = c.ID
		if rc, err := m.dk.Logs(ctx, c.ID, false, 200); err == nil {
			b, _ := io.ReadAll(io.LimitReader(rc, 1<<20))
			rc.Close()
			in.feed(b)
			in.flush()
		}
		att, err := m.dk.Attach(context.Background(), c.ID)
		if err != nil {
			log.Printf("Attach %s: %v", id, err)
			continue
		}
		in.mu.Lock()
		in.att = att
		in.state = StRunning
		if ci, err := m.dk.Inspect(ctx, c.ID); err == nil {
			in.started, _ = time.Parse(time.RFC3339Nano, ci.State.StartedAt)
		}
		in.mu.Unlock()
		in.watch(c.ID, att)
		in.setWant(true) // lief schon vor diesem Update: nach Neustart wieder starten
	}
	// Installationen, die beim Absturz liefen, als fehlgeschlagen markieren.
	m.st.Update(func(s *State) error {
		for _, x := range s.Servers {
			if x.Install == "installing" || x.Install == "importing" {
				x.Install = "failed"
			}
		}
		return nil
	})
}

// ---------- Ereignisse an die Weboberfläche ----------

func (in *Instance) Subscribe() (chan Event, []string, string, Stats, string) {
	ch := make(chan Event, 256)
	in.mu.Lock()
	defer in.mu.Unlock()
	in.subs[ch] = struct{}{}
	lines := append([]string(nil), in.lines...)
	return ch, lines, in.state, in.stats, in.hint
}

func (in *Instance) Unsubscribe(ch chan Event) {
	in.mu.Lock()
	delete(in.subs, ch)
	in.mu.Unlock()
}

func (in *Instance) emitLocked(e Event) {
	for ch := range in.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (in *Instance) State() string {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.state
}

func (in *Instance) Stats() Stats {
	in.mu.Lock()
	defer in.mu.Unlock()
	s := in.stats
	if in.state == StRunning || in.state == StStarting || in.state == StStopping {
		if !in.started.IsZero() {
			s.Uptime = int64(time.Since(in.started).Seconds())
		}
	} else {
		s = Stats{Disk: in.stats.Disk}
	}
	return s
}

func (in *Instance) setState(s string) {
	in.mu.Lock()
	if in.state != s {
		in.state = s
		in.emitLocked(Event{"status", s})
	}
	in.mu.Unlock()
}

func (in *Instance) setHint(h string) {
	in.mu.Lock()
	if in.hint != h {
		in.hint = h
		in.emitLocked(Event{"hint", h})
	}
	in.mu.Unlock()
}

// Panel-Meldung in der Konsole (wie "[Pterodactyl Daemon]").
func (in *Instance) Say(format string, a ...any) {
	line := "\x1b[1m\x1b[33m[serverwerk]\x1b[0m " + fmt.Sprintf(format, a...)
	in.mu.Lock()
	in.addLineLocked(line)
	in.mu.Unlock()
}

const maxLines = 1000

func (in *Instance) addLineLocked(l string) {
	in.lines = append(in.lines, l)
	if len(in.lines) > maxLines {
		in.lines = append([]string(nil), in.lines[len(in.lines)-maxLines:]...)
	}
	in.emitLocked(Event{"line", l})
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07`)

// feed nimmt rohe Ausgabe entgegen und macht daraus Zeilen.
func (in *Instance) feed(b []byte) {
	in.mu.Lock()
	defer in.mu.Unlock()
	in.partial = append(in.partial, b...)
	for {
		i := bytes.IndexByte(in.partial, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(in.partial[:i]), "\r")
		in.partial = in.partial[i+1:]
		// Ein \r mitten in der Zeile überschreibt (Fortschrittsbalken): nur den letzten Teil behalten.
		if j := strings.LastIndex(line, "\r"); j >= 0 {
			line = line[j+1:]
		}
		in.addLineLocked(line)
		in.checkLine(line)
	}
	if len(in.partial) > 0 {
		if in.flushT != nil {
			in.flushT.Stop()
		}
		in.flushT = time.AfterFunc(250*time.Millisecond, in.flush)
	}
	if len(in.partial) > 64<<10 {
		in.addLineLocked(string(in.partial))
		in.partial = nil
	}
}

func (in *Instance) flush() {
	in.mu.Lock()
	defer in.mu.Unlock()
	if len(in.partial) > 0 {
		line := strings.TrimRight(string(in.partial), "\r")
		if j := strings.LastIndex(line, "\r"); j >= 0 {
			line = line[j+1:]
		}
		in.partial = nil
		if strings.TrimSpace(ansiRe.ReplaceAllString(line, "")) != ">" {
			in.addLineLocked(line)
			in.checkLine(line)
		}
	}
}

// checkLine erkennt "Server ist bereit" und typische Probleme.
func (in *Instance) checkLine(line string) {
	plain := ansiRe.ReplaceAllString(line, "")
	if in.state == StStarting {
		if egg := in.eggLocked(); egg != nil {
			for _, d := range egg.Done {
				if d != "" && strings.Contains(plain, d) {
					in.state = StRunning
					in.emitLocked(Event{"status", StRunning})
					break
				}
			}
		}
	}
	low := strings.ToLower(plain)
	if strings.Contains(low, "agree to the eula") {
		in.hint = "eula"
		in.emitLocked(Event{"hint", "eula"})
	} else if strings.Contains(plain, "UnsupportedClassVersionError") || strings.Contains(low, "requires running the server with java") || strings.Contains(low, "has been compiled by a more recent version of the java") {
		in.hint = "java"
		in.emitLocked(Event{"hint", "java"})
	}
}

// eggLocked darf nur kurz unter in.mu laufen (liest den Store mit eigener Sperre).
func (in *Instance) eggLocked() *Egg {
	s := in.m.st.Server(in.id)
	if s == nil {
		return nil
	}
	return in.m.st.Egg(s.EggID)
}

// ---------- Befehle ----------

func (in *Instance) SendCommand(cmd string) error {
	in.mu.Lock()
	att := in.att
	st := in.state
	in.mu.Unlock()
	if att == nil || (st != StRunning && st != StStarting && st != StStopping) {
		return fmt.Errorf("Server läuft nicht")
	}
	_, err := att.Write([]byte(cmd + "\n"))
	return err
}

func (in *Instance) Power(action string) error {
	switch action {
	case "start":
		return in.Start()
	case "stop":
		in.setWant(false)
		return in.Stop(false)
	case "restart":
		go func() {
			if err := in.Stop(true); err != nil {
				in.Say("Neustart abgebrochen: %v", err)
				return
			}
			if err := in.Start(); err != nil {
				in.Say("Start fehlgeschlagen: %v", err)
			}
		}()
		return nil
	case "kill":
		in.setWant(false)
		return in.Kill()
	}
	return fmt.Errorf("unbekannte Aktion")
}

// setWant merkt sich, ob der Server laufen soll (für den Start nach einem Neustart).
func (in *Instance) setWant(v bool) {
	in.m.st.Update(func(s *State) error {
		if x := findServer(s, in.id); x != nil {
			x.WantRunning = v
		}
		return nil
	})
}

func (in *Instance) Kill() error {
	in.mu.Lock()
	c := in.container
	in.stopping = true
	in.mu.Unlock()
	if c == "" {
		in.setState(StOffline)
		return nil
	}
	in.Say("Server wird hart beendet.")
	return in.m.dk.Kill(context.Background(), c, "SIGKILL")
}

// Stop schickt den Stopp-Befehl aus dem Egg. wait=true wartet bis aus.
func (in *Instance) Stop(wait bool) error {
	in.mu.Lock()
	st := in.state
	c := in.container
	att := in.att
	in.mu.Unlock()
	if st == StOffline || st == StInstalling {
		return nil
	}
	if c == "" {
		in.setState(StOffline)
		return nil
	}
	in.mu.Lock()
	in.stopping = true
	in.mu.Unlock()
	in.setState(StStopping)
	stop := "^C"
	if s := in.m.st.Server(in.id); s != nil {
		if e := in.m.st.Egg(s.EggID); e != nil {
			stop = e.Stop
		}
	}
	ctx := context.Background()
	switch stop {
	case "^C", "^c":
		in.m.dk.Kill(ctx, c, "SIGINT")
	case "^^C":
		in.m.dk.Kill(ctx, c, "SIGTERM")
	default:
		if strings.HasPrefix(stop, "^") {
			in.m.dk.Kill(ctx, c, "SIG"+strings.TrimPrefix(stop, "^"))
		} else if att != nil {
			att.Write([]byte(stop + "\n"))
		}
	}
	// Nach 3 Minuten hart beenden.
	done := make(chan struct{})
	go func() {
		t := time.NewTimer(3 * time.Minute)
		defer t.Stop()
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-t.C:
				in.Say("Server reagiert nicht, wird hart beendet.")
				in.m.dk.Kill(context.Background(), c, "SIGKILL")
			case <-tick.C:
				if in.State() == StOffline {
					close(done)
					return
				}
			}
		}
	}()
	if wait {
		<-done
	}
	return nil
}

func (in *Instance) Start() error {
	if !in.powerLock.TryLock() {
		return fmt.Errorf("es läuft schon ein Vorgang")
	}
	st := in.State()
	if st != StOffline {
		in.powerLock.Unlock()
		if st == StInstalling {
			return fmt.Errorf("Installation läuft noch")
		}
		return fmt.Errorf("Server läuft schon")
	}
	in.setState(StStarting)
	in.setHint("")
	in.setWant(true)
	go func() {
		defer in.powerLock.Unlock()
		if err := in.start(); err != nil {
			in.Say("Start fehlgeschlagen: %v", err)
			in.setState(StOffline)
		}
	}()
	return nil
}

func (in *Instance) start() error {
	srv := in.m.st.Server(in.id)
	if srv == nil {
		return fmt.Errorf("Server gibt es nicht mehr")
	}
	egg := in.m.st.Egg(srv.EggID)
	if egg == nil {
		return fmt.Errorf("Egg fehlt")
	}
	if srv.Install == "installing" {
		return fmt.Errorf("Installation läuft noch")
	}
	if srv.Install == "importing" {
		return fmt.Errorf("Import läuft noch")
	}
	if srv.Install != "ok" && srv.Install != "failed" {
		return fmt.Errorf("Server ist noch nicht installiert")
	}
	dir := in.m.ServerDir(srv.ID)
	os.MkdirAll(dir, 0755)
	if srv.Disk > 0 {
		used := dirSize(dir)
		in.mu.Lock()
		in.stats.Disk = used
		in.mu.Unlock()
		if used > uint64(srv.Disk)<<20 {
			return fmt.Errorf("Speicherplatz voll (%s von %d MB belegt)", humanBytes(used), srv.Disk)
		}
	}
	env := mergedEnv(srv, egg)
	vc := &varCtx{srv: srv, env: env, gateway: in.m.gateway}
	in.Say("Konfigurationsdateien werden angepasst ...")
	for _, w := range applyConfigFiles(dir, egg, vc) {
		in.Say("Hinweis: %s", w)
	}
	ctx := context.Background()
	img := srv.Image
	if img == "" && len(egg.Images) > 0 {
		img = egg.Images[0].Image
	}
	if !in.m.dk.ImageExists(ctx, img) {
		in.Say("Image %s wird geladen ...", img)
		if err := in.m.dk.Pull(ctx, img, func(l string) { in.Say("%s", l) }); err != nil {
			return fmt.Errorf("Image laden: %v", err)
		}
	}
	name := containerName(srv.ID)
	in.m.dk.Remove(ctx, name)
	cfg := runtimeConfig(srv, egg, img, env, dir, in.m.timezone())
	id, err := in.m.dk.Create(ctx, name, cfg)
	if err != nil {
		return err
	}
	att, err := in.m.dk.Attach(context.Background(), id)
	if err != nil {
		in.m.dk.Remove(ctx, id)
		return err
	}
	in.mu.Lock()
	in.container = id
	in.att = att
	in.stopping = false
	in.started = time.Now()
	in.mu.Unlock()
	in.watch(id, att)
	in.Say("Server wird gestartet ...")
	if err := in.m.dk.Start(ctx, id); err != nil {
		in.mu.Lock()
		in.stopping = true
		in.mu.Unlock()
		att.Close()
		in.m.dk.Remove(ctx, id)
		return err
	}
	if len(egg.Done) == 0 {
		in.setState(StRunning)
	}
	return nil
}

// watch liest die Ausgabe, wartet auf das Ende und sammelt Werte.
func (in *Instance) watch(id string, att *Attachment) {
	ctx, cancel := context.WithCancel(context.Background())
	in.mu.Lock()
	if in.cancel != nil {
		in.cancel()
	}
	in.cancel = cancel
	in.mu.Unlock()

	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := att.R.Read(buf)
			if n > 0 {
				in.feed(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		for ctx.Err() == nil {
			err := in.m.dk.StatsStream(ctx, id, func(s Stats) {
				in.mu.Lock()
				disk := in.stats.Disk
				in.stats = s
				in.stats.Disk = disk
				out := in.stats
				if !in.started.IsZero() {
					out.Uptime = int64(time.Since(in.started).Seconds())
				}
				in.emitLocked(Event{"stats", out})
				in.mu.Unlock()
			})
			if err != nil && ctx.Err() == nil {
				time.Sleep(2 * time.Second)
			}
		}
	}()
	go func() {
		code, err := in.m.dk.Wait(context.Background(), id)
		cancel()
		time.Sleep(200 * time.Millisecond)
		in.flush()
		att.Close()
		in.mu.Lock()
		wasStopping := in.stopping
		ranFor := time.Since(in.started)
		if in.container == id {
			in.att = nil
		}
		in.mu.Unlock()
		oom := false
		if ci, e := in.m.dk.Inspect(context.Background(), id); e == nil {
			oom = ci.State.OOMKilled
		}
		in.setState(StOffline)
		if err != nil {
			in.Say("Verbindung zu Docker verloren: %v", err)
			return
		}
		if oom {
			in.Say("Server wurde beendet, weil der Arbeitsspeicher voll war (Exit %d).", code)
		} else {
			in.Say("Server ist aus (Exit-Code %d).", code)
		}
		in.mu.Lock()
		eula := in.hint == "eula"
		in.mu.Unlock()
		if !wasStopping && !eula {
			srv := in.m.st.Server(in.id)
			if srv != nil && srv.AutoRestart {
				if ranFor < 60*time.Second {
					in.Say("Absturz kurz nach dem Start, kein automatischer Neustart.")
					return
				}
				in.Say("Absturz erkannt, Server startet neu.")
				time.Sleep(2 * time.Second)
				in.Start()
			}
		}
	}()
}

func mergedEnv(srv *Server, egg *Egg) map[string]string {
	env := egg.DefaultEnv()
	for k, v := range srv.Env {
		env[k] = v
	}
	return env
}

// Speicher-Aufschlag wie Wings, damit die JVM nicht am Limit klebt.
func memLimit(mb int) int64 {
	if mb <= 0 {
		return 0
	}
	f := 1.05
	if mb <= 2048 {
		f = 1.15
	} else if mb <= 4096 {
		f = 1.10
	}
	return int64(float64(mb)*f) << 20
}

func baseEnv(srv *Server, env map[string]string, tz string) []string {
	port := 0
	if len(srv.Ports) > 0 {
		port = srv.Ports[0]
	}
	if tz == "" {
		tz = hostTimezone()
	}
	out := []string{
		"SERVER_MEMORY=" + strconv.Itoa(srv.Memory),
		"SERVER_IP=0.0.0.0",
		"SERVER_PORT=" + strconv.Itoa(port),
		"P_SERVER_LOCATION=serverwerk",
		"P_SERVER_UUID=" + srv.ID,
		"P_SERVER_ALLOCATION_LIMIT=0",
		"TZ=" + tz,
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}

func hostTimezone() string {
	if b, err := os.ReadFile("/etc/timezone"); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			return t
		}
	}
	if l, err := os.Readlink("/etc/localtime"); err == nil {
		if i := strings.Index(l, "zoneinfo/"); i >= 0 {
			return l[i+9:]
		}
	}
	return "Europe/Berlin"
}

func portBindings(ports []int) (map[string]any, map[string]any) {
	exposed := map[string]any{}
	bind := map[string]any{}
	for _, p := range ports {
		for _, proto := range []string{"tcp", "udp"} {
			k := fmt.Sprintf("%d/%s", p, proto)
			exposed[k] = map[string]any{}
			bind[k] = []map[string]string{{"HostIp": "0.0.0.0", "HostPort": strconv.Itoa(p)}}
		}
	}
	return exposed, bind
}

func (m *Manager) timezone() string {
	tz := ""
	m.st.Read(func(s *State) { tz = s.Settings.Timezone })
	return tz
}

func runtimeConfig(srv *Server, egg *Egg, img string, env map[string]string, dir string, tz string) map[string]any {
	startup := srv.Startup
	if startup == "" {
		startup = egg.Startup
	}
	e := append(baseEnv(srv, env, tz), "STARTUP="+startup)
	exposed, bind := portBindings(srv.Ports)
	host := map[string]any{
		"PortBindings": bind,
		"Mounts": []map[string]any{
			{"Type": "bind", "Source": dir, "Target": "/home/container", "ReadOnly": false},
		},
		"Tmpfs":       map[string]string{"/tmp": "rw,exec,nosuid,size=100M"},
		"LogConfig":   map[string]any{"Type": "local", "Config": map[string]string{"max-size": "5m", "max-file": "2", "compress": "false"}},
		"SecurityOpt": []string{"no-new-privileges"},
		"CapDrop":     []string{"setpcap", "mknod", "audit_write", "net_raw", "dac_override", "fowner", "fsetid", "net_bind_service", "sys_chroot", "setfcap"},
		"DNS":         []string{"1.1.1.1", "1.0.0.1"},
	}
	if lim := memLimit(srv.Memory); lim > 0 {
		host["Memory"] = lim
		host["MemoryReservation"] = int64(srv.Memory) << 20
		host["MemorySwap"] = lim
	}
	if srv.CPU > 0 {
		host["CpuPeriod"] = 100000
		host["CpuQuota"] = srv.CPU * 1000
	}
	if egg.HasFeature("pid_limit") || true {
		host["PidsLimit"] = 1024
	}
	return map[string]any{
		// Falls Docker die Container selbst stoppt: genug Zeit zum Speichern lassen.
		"StopTimeout":  90,
		"Hostname":     containerName(srv.ID),
		"User":         fmt.Sprintf("%d:%d", containerUID, containerUID),
		"Image":        img,
		"Env":          e,
		"Tty":          true,
		"OpenStdin":    true,
		"AttachStdin":  true,
		"AttachStdout": true,
		"AttachStderr": true,
		"StdinOnce":    false,
		"WorkingDir":   "/home/container",
		"ExposedPorts": exposed,
		"Labels":       map[string]string{"serverwerk.server": srv.ID, "serverwerk.role": "server"},
		"HostConfig":   host,
	}
}

// ---------- Installation ----------

func (in *Instance) Install() error {
	if !in.powerLock.TryLock() {
		return fmt.Errorf("es läuft schon ein Vorgang")
	}
	if st := in.State(); st != StOffline {
		in.powerLock.Unlock()
		return fmt.Errorf("Server muss aus sein")
	}
	in.setState(StInstalling)
	in.m.st.Update(func(s *State) error {
		if x := findServer(s, in.id); x != nil {
			x.Install = "installing"
		}
		return nil
	})
	go func() {
		defer in.powerLock.Unlock()
		err := in.install()
		res := "ok"
		if err != nil {
			res = "failed"
			in.Say("Installation fehlgeschlagen: %v", err)
		} else {
			in.Say("Installation fertig.")
		}
		in.m.st.Update(func(s *State) error {
			if x := findServer(s, in.id); x != nil {
				x.Install = res
			}
			return nil
		})
		in.setState(StOffline)
		in.mu.Lock()
		in.emitLocked(Event{"install", res})
		in.mu.Unlock()
	}()
	return nil
}

func (in *Instance) install() error {
	srv := in.m.st.Server(in.id)
	if srv == nil {
		return fmt.Errorf("Server gibt es nicht mehr")
	}
	egg := in.m.st.Egg(srv.EggID)
	if egg == nil {
		return fmt.Errorf("Egg fehlt")
	}
	dir := in.m.ServerDir(srv.ID)
	os.MkdirAll(dir, 0755)
	if strings.TrimSpace(egg.InstScript) == "" || egg.InstImage == "" {
		in.Say("Egg hat kein Installationsskript, nichts zu tun.")
		chownTree(dir)
		return nil
	}
	ctx := context.Background()
	in.Say("Installation von %s beginnt.", egg.Name)
	if !in.m.dk.ImageExists(ctx, egg.InstImage) {
		in.Say("Installations-Image %s wird geladen ...", egg.InstImage)
	}
	// Immer frisch ziehen, falls möglich (Installer-Images ändern sich).
	if err := in.m.dk.Pull(ctx, egg.InstImage, func(l string) { in.Say("%s", l) }); err != nil {
		if !in.m.dk.ImageExists(ctx, egg.InstImage) {
			return fmt.Errorf("Image laden: %v", err)
		}
		in.Say("Image konnte nicht aktualisiert werden, nehme vorhandenes.")
	}
	instDir := filepath.Join(in.m.data, "tmp", "install-"+srv.ID)
	os.RemoveAll(instDir)
	os.MkdirAll(instDir, 0755)
	defer os.RemoveAll(instDir)
	script := strings.ReplaceAll(egg.InstScript, "\r\n", "\n")
	if err := os.WriteFile(filepath.Join(instDir, "install.sh"), []byte(script), 0755); err != nil {
		return err
	}
	env := mergedEnv(srv, egg)
	mem := srv.Memory
	if mem < 1024 {
		mem = 1024
	}
	name := containerName(srv.ID) + "-install"
	in.m.dk.Remove(ctx, name)
	cfg := map[string]any{
		"Hostname":     "installer",
		"Image":        egg.InstImage,
		"Env":          baseEnv(srv, env, in.m.timezone()),
		"Cmd":          []string{egg.InstEntry, "/mnt/install/install.sh"},
		"Tty":          true,
		"AttachStdout": true,
		"AttachStderr": true,
		"Labels":       map[string]string{"serverwerk.server": srv.ID, "serverwerk.role": "install"},
		"HostConfig": map[string]any{
			"Mounts": []map[string]any{
				{"Type": "bind", "Source": dir, "Target": "/mnt/server"},
				{"Type": "bind", "Source": instDir, "Target": "/mnt/install"},
			},
			"Tmpfs":     map[string]string{"/tmp": "rw,exec,nosuid,size=200M"},
			"Memory":    int64(mem) << 20,
			"LogConfig": map[string]any{"Type": "local"},
			"DNS":       []string{"1.1.1.1", "1.0.0.1"},
		},
	}
	id, err := in.m.dk.Create(ctx, name, cfg)
	if err != nil {
		return err
	}
	defer in.m.dk.Remove(context.Background(), id)
	if err := in.m.dk.Start(ctx, id); err != nil {
		return err
	}
	in.mu.Lock()
	in.container = id
	in.mu.Unlock()
	if rc, err := in.m.dk.Logs(ctx, id, true, -1); err == nil {
		buf := make([]byte, 32<<10)
		for {
			n, err := rc.Read(buf)
			if n > 0 {
				in.feed(buf[:n])
			}
			if err != nil {
				break
			}
		}
		rc.Close()
		in.flush()
	}
	code, err := in.m.dk.Wait(ctx, id)
	in.mu.Lock()
	in.container = ""
	in.mu.Unlock()
	if err != nil {
		return err
	}
	chownTree(dir)
	if code != 0 {
		return fmt.Errorf("Skript endete mit Exit-Code %d", code)
	}
	return nil
}

// ---------- Hilfen ----------

func chownTree(dir string) {
	filepath.Walk(dir, func(p string, _ os.FileInfo, err error) error {
		if err == nil {
			os.Lchown(p, containerUID, containerUID)
		}
		return nil
	})
}

func dirSize(dir string) uint64 {
	// du ist deutlich schneller als Walk bei vielen kleinen Dateien.
	if out, err := exec.Command("du", "-sb", dir).Output(); err == nil {
		if f := strings.Fields(string(out)); len(f) > 0 {
			if n, err := strconv.ParseUint(f[0], 10, 64); err == nil {
				return n
			}
		}
	}
	var total uint64
	filepath.Walk(dir, func(_ string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			total += uint64(fi.Size())
		}
		return nil
	})
	return total
}

func humanBytes(n uint64) string {
	f := float64(n)
	for _, u := range []string{"B", "KB", "MB", "GB", "TB"} {
		if f < 1024 || u == "TB" {
			if u == "B" {
				return fmt.Sprintf("%d B", n)
			}
			return fmt.Sprintf("%.1f %s", f, u)
		}
		f /= 1024
	}
	return ""
}

// DiskLoop misst regelmäßig den Speicherplatz aller Server.
func (m *Manager) DiskLoop() {
	for {
		var ids []string
		m.st.Read(func(s *State) {
			for _, x := range s.Servers {
				ids = append(ids, x.ID)
			}
		})
		for _, id := range ids {
			used := dirSize(m.ServerDir(id))
			in := m.Get(id)
			in.mu.Lock()
			in.stats.Disk = used
			in.mu.Unlock()
		}
		time.Sleep(60 * time.Second)
	}
}
