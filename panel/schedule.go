package main

// Zeitpläne (Cron, 5 Felder) und Backups.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type cronField map[int]bool

type Cron struct{ min, hour, dom, mon, dow cronField }

func parseField(s string, lo, hi int) (cronField, error) {
	f := cronField{}
	for _, part := range strings.Split(s, ",") {
		step := 1
		if i := strings.Index(part, "/"); i >= 0 {
			n, err := strconv.Atoi(part[i+1:])
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("ungültiger Schritt %q", part)
			}
			step = n
			part = part[:i]
		}
		a, b := lo, hi
		if part != "*" {
			if x, y, ok := strings.Cut(part, "-"); ok {
				var e1, e2 error
				a, e1 = strconv.Atoi(x)
				b, e2 = strconv.Atoi(y)
				if e1 != nil || e2 != nil {
					return nil, fmt.Errorf("ungültiger Bereich %q", part)
				}
			} else {
				n, err := strconv.Atoi(part)
				if err != nil {
					return nil, fmt.Errorf("ungültiger Wert %q", part)
				}
				a, b = n, n
				if step > 1 {
					b = hi
				}
			}
		}
		if a < lo || b > hi || a > b {
			return nil, fmt.Errorf("Wert außerhalb %d-%d", lo, hi)
		}
		for v := a; v <= b; v += step {
			f[v] = true
		}
	}
	return f, nil
}

func ParseCron(s string) (*Cron, error) {
	p := strings.Fields(s)
	if len(p) != 5 {
		return nil, fmt.Errorf("Cron braucht 5 Felder: Minute Stunde Tag Monat Wochentag")
	}
	var c Cron
	var err error
	if c.min, err = parseField(p[0], 0, 59); err != nil {
		return nil, err
	}
	if c.hour, err = parseField(p[1], 0, 23); err != nil {
		return nil, err
	}
	if c.dom, err = parseField(p[2], 1, 31); err != nil {
		return nil, err
	}
	if c.mon, err = parseField(p[3], 1, 12); err != nil {
		return nil, err
	}
	if c.dow, err = parseField(strings.ReplaceAll(p[4], "7", "0"), 0, 6); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Cron) Match(t time.Time) bool {
	return c.min[t.Minute()] && c.hour[t.Hour()] && c.dom[t.Day()] && c.mon[int(t.Month())] && c.dow[int(t.Weekday())]
}

func (c *Cron) Next(from time.Time) time.Time {
	t := from.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 366*24*60; i++ {
		if c.Match(t) {
			return t
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}
}

func (a *App) scheduleLoop() {
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next) + 200*time.Millisecond)
		t := time.Now().Truncate(time.Minute)
		type job struct {
			srv string
			sc  Schedule
		}
		var jobs []job
		a.st.Read(func(s *State) {
			for _, srv := range s.Servers {
				for _, sc := range srv.Schedules {
					if !sc.Enabled {
						continue
					}
					c, err := ParseCron(sc.Cron)
					if err == nil && c.Match(t) {
						jobs = append(jobs, job{srv.ID, sc})
					}
				}
			}
		})
		for _, j := range jobs {
			go a.runSchedule(j.srv, j.sc)
		}
	}
}

func (a *App) runSchedule(srvID string, sc Schedule) {
	in := a.m.Get(srvID)
	if sc.OnlyIfOn && in.State() != StRunning {
		return
	}
	now := time.Now()
	a.st.Update(func(s *State) error {
		if x := findServer(s, srvID); x != nil {
			for i := range x.Schedules {
				if x.Schedules[i].ID == sc.ID {
					x.Schedules[i].LastRun = &now
				}
			}
		}
		return nil
	})
	in.Say("Zeitplan \"%s\" läuft.", sc.Name)
	for _, t := range sc.Tasks {
		if t.Delay > 0 {
			time.Sleep(time.Duration(t.Delay) * time.Second)
		}
		var err error
		switch t.Action {
		case "command":
			err = in.SendCommand(t.Payload)
		case "power":
			if t.Payload == "restart" {
				if err = in.Stop(true); err == nil {
					err = in.Start()
				}
			} else if t.Payload == "stop" {
				in.setWant(false)
				err = in.Stop(true)
			} else {
				err = in.Power(t.Payload)
			}
		case "backup":
			_, err = a.makeBackup(srvID, "Zeitplan: "+sc.Name)
		}
		if err != nil {
			in.Say("Zeitplan \"%s\": %v", sc.Name, err)
		}
	}
}

// ---------- Backups ----------

type Backup struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
	Note    string    `json:"note"`
}

func (a *App) listBackups(srvID string) []Backup {
	dir := a.m.BackupDir(srvID)
	ents, _ := os.ReadDir(dir)
	out := []Backup{}
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		b := Backup{Name: e.Name(), Size: fi.Size(), Created: fi.ModTime()}
		if note, err := os.ReadFile(filepath.Join(dir, strings.TrimSuffix(e.Name(), ".tar.gz")+".txt")); err == nil {
			b.Note = strings.TrimSpace(string(note))
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

func (a *App) makeBackup(srvID, note string) (string, error) {
	srv := a.st.Server(srvID)
	if srv == nil {
		return "", fmt.Errorf("Server fehlt")
	}
	dir := a.m.BackupDir(srvID)
	os.MkdirAll(dir, 0700)
	name := time.Now().Format("2006-01-02_15-04-05")
	tmp := filepath.Join(dir, name+".tar.gz.part")
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	root := a.m.ServerDir(srvID)
	var ignore []string
	for _, fn := range []string{".swignore", ".pteroignore"} {
		if fh, err := os.Open(filepath.Join(root, fn)); err == nil {
			sc := bufio.NewScanner(fh)
			for sc.Scan() {
				ignore = append(ignore, sc.Text())
			}
			fh.Close()
		}
	}
	in := a.m.Get(srvID)
	in.Say("Backup wird erstellt ...")
	// Minecraft-Welten möglichst konsistent: Speichern kurz anhalten, falls möglich.
	err = tarDir(root, f, ignore)
	f.Close()
	if err != nil {
		os.Remove(tmp)
		in.Say("Backup fehlgeschlagen: %v", err)
		return "", err
	}
	final := filepath.Join(dir, name+".tar.gz")
	os.Rename(tmp, final)
	if note != "" {
		os.WriteFile(filepath.Join(dir, name+".txt"), []byte(note), 0600)
	}
	fi, _ := os.Stat(final)
	if fi != nil {
		in.Say("Backup fertig (%s).", humanBytes(uint64(fi.Size())))
	}
	// Älteste löschen, wenn zu viele.
	max := srv.MaxBackups
	if max <= 0 {
		max = 5
	}
	list := a.listBackups(srvID)
	for i := max; i < len(list); i++ {
		a.deleteBackup(srvID, list[i].Name)
	}
	return name + ".tar.gz", nil
}

func (a *App) backupPath(srvID, name string) (string, error) {
	if strings.ContainsAny(name, "/\\") || !strings.HasSuffix(name, ".tar.gz") {
		return "", fmt.Errorf("ungültiger Name")
	}
	p := filepath.Join(a.m.BackupDir(srvID), name)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("Backup nicht gefunden")
	}
	return p, nil
}

func (a *App) deleteBackup(srvID, name string) error {
	p, err := a.backupPath(srvID, name)
	if err != nil {
		return err
	}
	os.Remove(strings.TrimSuffix(p, ".tar.gz") + ".txt")
	return os.Remove(p)
}

func (a *App) restoreBackup(srvID, name string) error {
	p, err := a.backupPath(srvID, name)
	if err != nil {
		return err
	}
	in := a.m.Get(srvID)
	if in.State() != StOffline {
		return fmt.Errorf("Server muss dafür aus sein")
	}
	root := a.m.ServerDir(srvID)
	ents, _ := os.ReadDir(root)
	for _, e := range ents {
		os.RemoveAll(filepath.Join(root, e.Name()))
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	in.Say("Backup %s wird eingespielt ...", name)
	if err := extractTar(p, root, root); err != nil {
		return err
	}
	in.Say("Backup eingespielt.")
	return nil
}
