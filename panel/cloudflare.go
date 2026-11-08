package main

// Cloudflare-DNS für Server: name.example.com zeigt auf diesen Rechner,
// bei Minecraft Java zusätzlich ein SRV-Eintrag, damit kein Port nötig ist.
// Serverwerk ändert nur Einträge, die es selbst angelegt hat (Kommentar "serverwerk:<id>").

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Domain struct {
	ZoneID   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	Prefix   string `json:"prefix"`
	SRV      bool   `json:"srv"`
	Status   string `json:"status"`
}

func (d *Domain) FQDN() string {
	if d == nil || d.ZoneName == "" {
		return ""
	}
	if d.Prefix == "" || d.Prefix == "@" {
		return d.ZoneName
	}
	return d.Prefix + "." + d.ZoneName
}

type cfResp struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int
		Message string
	} `json:"errors"`
	Result json.RawMessage `json:"result"`
}

var cfBase = "https://api.cloudflare.com/client/v4"

func cfCall(token, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, cfBase+path, rd)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return fmt.Errorf("Cloudflare nicht erreichbar: %v", err)
	}
	defer resp.Body.Close()
	var r cfResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&r); err != nil {
		return fmt.Errorf("Cloudflare: HTTP %d", resp.StatusCode)
	}
	if !r.Success {
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if len(r.Errors) > 0 {
			msg = r.Errors[0].Message
		}
		badToken := resp.StatusCode == 401 || resp.StatusCode == 403
		for _, e := range r.Errors {
			switch e.Code {
			case 6003, 6111, 9106, 9109, 1000, 10000:
				badToken = true
			}
		}
		if badToken {
			msg = "Token ungültig oder ohne Rechte. Bitte einen API-Token mit „DNS-Zone bearbeiten“ verwenden."
		}
		return fmt.Errorf("Cloudflare: %s", msg)
	}
	if out != nil {
		return json.Unmarshal(r.Result, out)
	}
	return nil
}

type cfZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func cfZones(token string) ([]cfZone, error) {
	var z []cfZone
	err := cfCall(token, "GET", "/zones?per_page=50&status=active", nil, &z)
	return z, err
}

type cfRecord struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Content string         `json:"content,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Proxied *bool          `json:"proxied,omitempty"`
	TTL     int            `json:"ttl"`
	Comment string         `json:"comment"`
}

func cfFind(token, zone, typ, name string) ([]cfRecord, error) {
	var rs []cfRecord
	err := cfCall(token, "GET", "/zones/"+zone+"/dns_records?type="+typ+"&name="+url.QueryEscape(name), nil, &rs)
	return rs, err
}

// cfUpsert legt einen Eintrag an oder ändert unseren eigenen; fremde bleiben unangetastet.
func cfUpsert(token, zone string, rec cfRecord) error {
	existing, err := cfFind(token, zone, rec.Type, rec.Name)
	if err != nil {
		return err
	}
	// Ein CNAME schließt andere Typen mit gleichem Namen aus (und umgekehrt).
	for _, other := range []string{"A", "AAAA", "CNAME"} {
		if rec.Type == "SRV" || other == rec.Type {
			continue
		}
		rs, _ := cfFind(token, zone, other, rec.Name)
		for _, r := range rs {
			if r.Comment != rec.Comment {
				return fmt.Errorf("%s gibt es in Cloudflare schon (%s-Eintrag, nicht von Serverwerk)", rec.Name, other)
			}
			cfCall(token, "DELETE", "/zones/"+zone+"/dns_records/"+r.ID, nil, nil)
		}
	}
	for _, r := range existing {
		if r.Comment == rec.Comment {
			return cfCall(token, "PUT", "/zones/"+zone+"/dns_records/"+r.ID, rec, nil)
		}
	}
	if len(existing) > 0 && rec.Type != "SRV" {
		return fmt.Errorf("%s gibt es in Cloudflare schon (nicht von Serverwerk angelegt)", rec.Name)
	}
	return cfCall(token, "POST", "/zones/"+zone+"/dns_records", rec, nil)
}

// cfDeleteOwn löscht alle Einträge mit unserem Kommentar für diesen Server.
func cfDeleteOwn(token, zone, comment string) error {
	var rs []cfRecord
	if err := cfCall(token, "GET", "/zones/"+zone+"/dns_records?per_page=100&comment="+url.QueryEscape(comment), nil, &rs); err != nil {
		return err
	}
	for _, r := range rs {
		if r.Comment == comment {
			cfCall(token, "DELETE", "/zones/"+zone+"/dns_records/"+r.ID, nil, nil)
		}
	}
	return nil
}

var publicIP struct {
	sync.Mutex
	ip string
	at time.Time
}

func detectPublicIP(force bool) (string, error) {
	publicIP.Lock()
	defer publicIP.Unlock()
	if !force && publicIP.ip != "" && time.Since(publicIP.at) < 4*time.Minute {
		return publicIP.ip, nil
	}
	b, err := httpGet("https://1.1.1.1/cdn-cgi/trace", 64<<10)
	if err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "ip="); ok {
				if ip := net.ParseIP(strings.TrimSpace(v)); ip != nil && ip.To4() != nil {
					publicIP.ip, publicIP.at = ip.String(), time.Now()
					return publicIP.ip, nil
				}
			}
		}
	}
	b, err = httpGet("https://api.ipify.org", 1024)
	if err != nil {
		return "", fmt.Errorf("öffentliche IP nicht ermittelbar: %v", err)
	}
	ip := net.ParseIP(strings.TrimSpace(string(b)))
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("öffentliche IP nicht ermittelbar")
	}
	publicIP.ip, publicIP.at = ip.String(), time.Now()
	return publicIP.ip, nil
}

var prefixRe = regexp.MustCompile(`^(@|[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*)$`)

func (a *App) isMinecraftJava(s *Server) bool {
	k := a.addonInfo(s).Kind
	return k == "plugin" || k == "mod" || k == "datapack"
}

// syncDomain schreibt die DNS-Einträge eines Servers nach Cloudflare.
func (a *App) syncDomain(s *Server) error {
	if s.Domain == nil || s.Domain.ZoneID == "" {
		return nil
	}
	var set Settings
	a.st.Read(func(st *State) { set = st.Settings })
	if set.CFToken == "" {
		return fmt.Errorf("Cloudflare ist nicht verbunden (Einstellungen)")
	}
	fq := s.Domain.FQDN()
	comment := "serverwerk:" + s.ID
	f := false
	main := cfRecord{Name: fq, TTL: 300, Comment: comment, Proxied: &f}
	if set.CFTarget == "host" && set.CFHost != "" {
		main.Type, main.Content = "CNAME", set.CFHost
	} else {
		ip, err := detectPublicIP(false)
		if err != nil {
			return err
		}
		main.Type, main.Content = "A", ip
	}
	if err := cfUpsert(set.CFToken, s.Domain.ZoneID, main); err != nil {
		return err
	}
	srvName := "_minecraft._tcp." + fq
	if s.Domain.SRV && len(s.Ports) > 0 {
		rec := cfRecord{Type: "SRV", Name: srvName, TTL: 300, Comment: comment, Data: map[string]any{
			"priority": 0, "weight": 5, "port": s.Ports[0], "target": fq,
		}}
		if err := cfUpsert(set.CFToken, s.Domain.ZoneID, rec); err != nil {
			return err
		}
	} else {
		if rs, err := cfFind(set.CFToken, s.Domain.ZoneID, "SRV", srvName); err == nil {
			for _, r := range rs {
				if r.Comment == comment {
					cfCall(set.CFToken, "DELETE", "/zones/"+s.Domain.ZoneID+"/dns_records/"+r.ID, nil, nil)
				}
			}
		}
	}
	return nil
}

func (a *App) removeDomain(s *Server) error {
	if s.Domain == nil || s.Domain.ZoneID == "" {
		return nil
	}
	var token string
	a.st.Read(func(st *State) { token = st.Settings.CFToken })
	if token == "" {
		return nil
	}
	return cfDeleteOwn(token, s.Domain.ZoneID, "serverwerk:"+s.ID)
}

// dnsLoop hält die A-Einträge aktuell, wenn sich die öffentliche IP ändert (DynDNS).
func (a *App) dnsLoop() {
	last := ""
	for {
		time.Sleep(5 * time.Minute)
		var set Settings
		var list []*Server
		a.st.Read(func(st *State) {
			set = st.Settings
			for _, s := range st.Servers {
				if s.Domain != nil && s.Domain.ZoneID != "" {
					list = append(list, cloneServer(s))
				}
			}
		})
		if set.CFToken == "" || set.CFTarget == "host" || len(list) == 0 {
			continue
		}
		ip, err := detectPublicIP(true)
		if err != nil || ip == last {
			continue
		}
		if last != "" {
			log.Printf("Öffentliche IP geändert: %s -> %s, DNS wird angepasst", last, ip)
		}
		last = ip
		for _, s := range list {
			if err := a.syncDomain(s); err != nil {
				log.Printf("DNS %s: %v", s.Name, err)
			}
		}
	}
}

// ---------- API ----------

func maskToken(t string) string {
	if len(t) < 8 {
		return ""
	}
	return t[:4] + strings.Repeat("•", 8) + t[len(t)-4:]
}

func (a *App) settingsGet(w http.ResponseWriter, r *http.Request) {
	var set Settings
	n := 0
	a.st.Read(func(st *State) {
		set = st.Settings
		for _, s := range st.Servers {
			if s.Domain != nil && s.Domain.ZoneID != "" {
				n++
			}
		}
	})
	out := map[string]any{
		"panelName": set.PanelName, "accent": set.Accent, "language": set.Language,
		"cfConnected": set.CFToken != "", "cfToken": maskToken(set.CFToken), "cfZones": set.CFZones,
		"cfTarget": firstNonEmpty(set.CFTarget, "ip"), "cfHost": set.CFHost, "domainServers": n,
		"ports": set.Ports, "autoDomain": set.AutoDomain,
		"defaults": set.Defaults, "security": set.Security,
		"timezone": set.Timezone, "hostTimezone": hostTimezone(), "version": a.version,
	}
	if l := a.allowedPorts(); len(l) > 0 {
		used := a.usedPorts("")
		u := 0
		for _, p := range l {
			if used[p] {
				u++
			}
		}
		out["portsTotal"], out["portsUsed"] = len(l), u
	}
	if set.Panel != nil {
		run, why := a.tunnelRunning()
		out["panel"] = map[string]any{"url": "https://" + set.Panel.FQDN(), "zoneId": set.Panel.ZoneID, "prefix": set.Panel.Prefix, "running": run, "problem": why}
	}
	if ip, err := detectPublicIP(false); err == nil {
		out["publicIp"] = ip
	}
	jsonOut(w, out)
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (a *App) settingsPut(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PanelName  *string           `json:"panelName"`
		Accent     *string           `json:"accent"`
		CFToken    *string           `json:"cfToken"`
		CFTarget   *string           `json:"cfTarget"`
		CFHost     *string           `json:"cfHost"`
		Ports      *string           `json:"ports"`
		Timezone   *string           `json:"timezone"`
		Language   *string           `json:"language"`
		Defaults   *ServerDefaults   `json:"defaults"`
		Security   *SecuritySettings `json:"security"`
		AutoDomain *struct {
			Enabled bool   `json:"enabled"`
			ZoneID  string `json:"zoneId"`
			Sub     string `json:"sub"`
		} `json:"autoDomain"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	var zones []cfZone
	if in.CFToken != nil && *in.CFToken != "" {
		var err error
		zones, err = cfZones(strings.TrimSpace(*in.CFToken))
		if err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		if len(zones) == 0 {
			jsonErr(w, 400, "Der Token sieht keine Domain. Rechte: Zone > DNS > Bearbeiten und Zone > Zone > Lesen.")
			return
		}
	}
	if in.Timezone != nil && strings.TrimSpace(*in.Timezone) != "" {
		if _, err := time.LoadLocation(strings.TrimSpace(*in.Timezone)); err != nil {
			jsonErr(w, 400, "Unbekannte Zeitzone, z. B. Europe/Berlin")
			return
		}
	}
	if in.Ports != nil {
		if _, err := parsePortList(*in.Ports); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
	}
	if in.Accent != nil && *in.Accent != "" && !hexColor.MatchString(*in.Accent) {
		jsonErr(w, 400, "Farbe im Format #rrggbb")
		return
	}
	if in.CFHost != nil {
		h := strings.TrimSuffix(strings.TrimSpace(strings.ToLower(*in.CFHost)), ".")
		if h != "" && !prefixRe.MatchString(h) {
			jsonErr(w, 400, "Ungültiger Hostname")
			return
		}
		in.CFHost = &h
	}
	var auto *AutoDomain
	if x := in.AutoDomain; x != nil && x.Enabled {
		sub := strings.Trim(strings.ToLower(strings.TrimSpace(x.Sub)), ".")
		if sub != "" && (sub == "@" || !prefixRe.MatchString(sub)) {
			jsonErr(w, 400, "Ungültiger Name: nur a-z, 0-9, Punkt und Bindestrich")
			return
		}
		a.st.Read(func(st *State) {
			for _, z := range st.Settings.CFZones {
				if z.ID == x.ZoneID {
					auto = &AutoDomain{ZoneID: z.ID, ZoneName: z.Name, Sub: sub}
				}
			}
		})
		if auto == nil {
			jsonErr(w, 400, "Domain wählen (vorher Cloudflare in den Einstellungen verbinden)")
			return
		}
	}
	a.st.Update(func(st *State) error {
		s := &st.Settings
		if in.AutoDomain != nil {
			s.AutoDomain = auto
		}
		if in.PanelName != nil && strings.TrimSpace(*in.PanelName) != "" {
			s.PanelName = strings.TrimSpace(*in.PanelName)
		}
		if in.Accent != nil {
			s.Accent = *in.Accent
		}
		if in.Ports != nil {
			s.Ports = strings.TrimSpace(*in.Ports)
		}
		if in.Timezone != nil {
			s.Timezone = strings.TrimSpace(*in.Timezone)
		}
		if in.Language != nil && (*in.Language == "de" || *in.Language == "en") {
			s.Language = *in.Language
			setLang(s.Language)
		}
		if d := in.Defaults; d != nil {
			d.Init = true
			d.Memory, d.CPU, d.Disk = max(d.Memory, 0), max(d.CPU, 0), max(d.Disk, 0)
			d.MaxBackups = min(max(d.MaxBackups, 1), 100)
			s.Defaults = *d
		}
		if x := in.Security; x != nil {
			x.SessionDays = min(max(x.SessionDays, 1), 365)
			x.MaxFails = min(max(x.MaxFails, 3), 100)
			x.LockMinutes = min(max(x.LockMinutes, 1), 1440)
			s.Security = *x
		}
		if in.CFToken != nil {
			s.CFToken = strings.TrimSpace(*in.CFToken)
			s.CFZones = nil
			for _, z := range zones {
				s.CFZones = append(s.CFZones, CFZone{ID: z.ID, Name: z.Name})
			}
		}
		if in.CFTarget != nil && (*in.CFTarget == "ip" || *in.CFTarget == "host") {
			s.CFTarget = *in.CFTarget
		}
		if in.CFHost != nil {
			s.CFHost = *in.CFHost
		}
		return nil
	})
	// Ziel geändert: alle Domains neu schreiben.
	if in.CFTarget != nil || in.CFHost != nil {
		go a.syncAllDomains()
	}
	if auto != nil {
		go a.applyAutoDomainsAll()
	}
	a.settingsGet(w, r)
}

func (a *App) syncAllDomains() {
	var list []*Server
	a.st.Read(func(st *State) {
		for _, s := range st.Servers {
			if s.Domain != nil && s.Domain.ZoneID != "" {
				list = append(list, cloneServer(s))
			}
		}
	})
	for _, s := range list {
		err := a.syncDomain(s)
		a.setDomainStatus(s.ID, err)
	}
}

func (a *App) setDomainStatus(id string, err error) {
	a.st.Update(func(st *State) error {
		if x := findServer(st, id); x != nil && x.Domain != nil {
			x.Domain.Status = "ok"
			if err != nil {
				x.Domain.Status = err.Error()
			}
		}
		return nil
	})
}

func (a *App) serverDomain(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct {
		ZoneID string `json:"zoneId"`
		Prefix string `json:"prefix"`
		Remove bool   `json:"remove"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	if in.Remove {
		if err := a.removeDomain(s); err != nil {
			jsonErr(w, 502, err.Error())
			return
		}
		a.st.Update(func(st *State) error {
			if x := findServer(st, s.ID); x != nil {
				x.Domain = nil
			}
			return nil
		})
		ok(w)
		return
	}
	var zone *CFZone
	a.st.Read(func(st *State) {
		for _, z := range st.Settings.CFZones {
			if z.ID == in.ZoneID {
				c := z
				zone = &c
			}
		}
	})
	if zone == nil {
		jsonErr(w, 400, "Domain wählen (vorher Cloudflare in den Einstellungen verbinden)")
		return
	}
	p := strings.Trim(strings.ToLower(strings.TrimSpace(in.Prefix)), ".")
	if p == "" {
		p = "@"
	}
	if !prefixRe.MatchString(p) {
		jsonErr(w, 400, "Ungültiger Name: nur a-z, 0-9 und Bindestrich")
		return
	}
	nd := &Domain{ZoneID: zone.ID, ZoneName: zone.Name, Prefix: p, SRV: a.isMinecraftJava(s)}
	// Doppelte Vergabe im Panel verhindern.
	clash := ""
	a.st.Read(func(st *State) {
		for _, x := range st.Servers {
			if x.ID != s.ID && x.Domain != nil && x.Domain.FQDN() == nd.FQDN() {
				clash = x.Name
			}
		}
	})
	a.st.Read(func(st *State) {
		if st.Settings.Panel != nil && st.Settings.Panel.FQDN() == nd.FQDN() {
			clash = "das Panel"
		}
	})
	if clash != "" {
		jsonErr(w, 409, nd.FQDN()+" benutzt schon "+clash)
		return
	}
	// Alte Einträge weg, wenn sich der Name ändert.
	if s.Domain != nil && s.Domain.FQDN() != nd.FQDN() {
		a.removeDomain(s)
	}
	s.Domain = nd
	if err := a.syncDomain(s); err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	nd.Status = "ok"
	a.st.Update(func(st *State) error {
		if x := findServer(st, s.ID); x != nil {
			x.Domain = nd
		}
		return nil
	})
	jsonOut(w, nd)
}
