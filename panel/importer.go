package main

// Import aus Pterodactyl (und Pelican, gleiche API).
// Mit einem Client-API-Key (ptlc_) werden Server, Ports, Grenzen, Variablen und
// Dateien übernommen. Ein Application-Key (ptla_) ist optional und liefert das exakte Egg.
// Liegt Wings auf diesem Rechner, werden die Dateien direkt kopiert.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type pteroConn struct {
	Base      string
	ClientKey string
	AppKey    string
	dial      func(ctx context.Context, network, addr string) (net.Conn, error)
}

func (c *pteroConn) client(timeout time.Duration) *http.Client {
	tr := &http.Transport{ResponseHeaderTimeout: 10 * time.Minute, TLSHandshakeTimeout: 15 * time.Second}
	if c.dial != nil {
		tr.DialContext = c.dial
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

type pteroScan struct {
	conn    pteroConn
	created time.Time
	servers map[string]*pteroServer
}

type pteroServer struct {
	Identifier string            `json:"identifier"`
	UUID       string            `json:"uuid"`
	Name       string            `json:"name"`
	Desc       string            `json:"description"`
	EggName    string            `json:"eggName"`
	EggUUID    string            `json:"-"`
	Memory     int               `json:"memory"`
	CPU        int               `json:"cpu"`
	Disk       int               `json:"disk"`
	Ports      []int             `json:"ports"`
	Image      string            `json:"image"`
	Startup    string            `json:"startup"`
	Env        map[string]string `json:"-"`
	State      string            `json:"state"`
	DiskBytes  int64             `json:"diskBytes"`
	Local      bool              `json:"local"`
	Match      *eggMatch         `json:"match"`
	Imported   string            `json:"imported,omitempty"` // schon übernommen als Server-ID
}

type eggMatch struct {
	Source string `json:"source"` // installed | catalog | app | none
	EggID  string `json:"eggId,omitempty"`
	Name   string `json:"name"`
	URL    string `json:"-"`
	AltURL string `json:"-"`
	Cat    string `json:"-"`
	App    *Egg   `json:"-"`
}

var pteroScans = struct {
	sync.Mutex
	m map[string]*pteroScan
}{m: map[string]*pteroScan{}}

const pteroVolumes = "/var/lib/pterodactyl/volumes"

func (c *pteroConn) call(key, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.Base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "Application/vnd.pterodactyl.v1+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client(10 * time.Minute).Do(req)
	if err != nil {
		hint := ""
		if strings.Contains(err.Error(), "no route to host") || strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "timeout") {
			hint = " Dieser Rechner kommt nicht an die Adresse. Prüfe Adresse, Port und ob Pterodactyl läuft."
		}
		return fmt.Errorf("Pterodactyl nicht erreichbar: %v.%s", err, hint)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Errors []struct {
				Detail string `json:"detail"`
				Code   string `json:"code"`
			} `json:"errors"`
		}
		json.Unmarshal(b, &e)
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if len(e.Errors) > 0 {
			msg = e.Errors[0].Detail
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			msg = "API-Key ungültig oder ohne Rechte (" + msg + ")"
		}
		return fmt.Errorf("Pterodactyl: %s", msg)
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("Pterodactyl: unerwartete Antwort (ist die Adresse richtig?)")
		}
	}
	return nil
}

type pteroList struct {
	Data []struct {
		Attributes json.RawMessage `json:"attributes"`
	} `json:"data"`
	Meta struct {
		Pagination struct {
			CurrentPage int `json:"current_page"`
			TotalPages  int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"meta"`
}

type pteroClientServer struct {
	Identifier  string `json:"identifier"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DockerImage string `json:"docker_image"`
	Limits      struct {
		Memory int `json:"memory"`
		CPU    int `json:"cpu"`
		Disk   int `json:"disk"`
	} `json:"limits"`
	Relationships struct {
		Allocations struct {
			Data []struct {
				Attributes struct {
					Port      int  `json:"port"`
					IsDefault bool `json:"is_default"`
				} `json:"attributes"`
			} `json:"data"`
		} `json:"allocations"`
		Egg struct {
			Attributes struct {
				UUID string `json:"uuid"`
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"egg"`
	} `json:"relationships"`
}

func (a *App) importScan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL       string `json:"url"`
		ClientKey string `json:"clientKey"`
		AppKey    string `json:"appKey"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	base := strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	if u, err := url.Parse(base); err != nil || u.Host == "" {
		jsonErr(w, 400, "Adresse des Panels prüfen, z. B. https://panel.example.com")
		return
	}
	conn := pteroConn{Base: base, ClientKey: strings.TrimSpace(in.ClientKey), AppKey: strings.TrimSpace(in.AppKey), dial: a.shimDial}
	if conn.ClientKey == "" {
		jsonErr(w, 400, "Client-API-Key fehlt (beginnt mit ptlc_)")
		return
	}
	// Erst alle Server (als Admin), sonst nur die eigenen.
	var all []pteroClientServer
	for _, q := range []string{"type=admin-all&", ""} {
		all = nil
		var err error
		for page := 1; page < 50; page++ {
			var l pteroList
			err = conn.call(conn.ClientKey, "GET", fmt.Sprintf("/api/client?%sinclude=egg,allocations&per_page=100&page=%d", q, page), nil, &l)
			if err != nil {
				break
			}
			for _, d := range l.Data {
				var s pteroClientServer
				json.Unmarshal(d.Attributes, &s)
				all = append(all, s)
			}
			if l.Meta.Pagination.CurrentPage >= l.Meta.Pagination.TotalPages {
				break
			}
		}
		if err == nil {
			break
		}
		if q == "" {
			jsonErr(w, 502, err.Error())
			return
		}
	}
	installed := map[string]string{}
	imported := map[string]string{}
	a.st.Read(func(st *State) {
		for _, e := range st.Eggs {
			installed[strings.ToLower(e.Name)] = e.ID
		}
		for _, s := range st.Servers {
			if s.ImportedFrom != "" {
				imported[s.ImportedFrom] = s.ID
			}
		}
	})
	catalog, _ := a.cat.List()
	if len(catalog) == 0 {
		a.cat.Refresh()
		catalog, _ = a.cat.List()
	}
	scan := &pteroScan{conn: conn, created: time.Now(), servers: map[string]*pteroServer{}}
	out := []*pteroServer{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for _, cs := range all {
		ps := &pteroServer{Identifier: cs.Identifier, UUID: cs.UUID, Name: cs.Name, Desc: cs.Description,
			EggName: cs.Relationships.Egg.Attributes.Name, EggUUID: cs.Relationships.Egg.Attributes.UUID,
			Memory: cs.Limits.Memory, CPU: cs.Limits.CPU, Disk: cs.Limits.Disk, Image: cs.DockerImage, Env: map[string]string{}}
		for _, al := range cs.Relationships.Allocations.Data {
			if al.Attributes.IsDefault {
				ps.Ports = append([]int{al.Attributes.Port}, ps.Ports...)
			} else {
				ps.Ports = append(ps.Ports, al.Attributes.Port)
			}
		}
		if fi, err := os.Stat(filepath.Join(pteroVolumes, cs.UUID)); err == nil && fi.IsDir() {
			ps.Local = true
		}
		ps.Imported = imported[conn.Base+"#"+cs.UUID]
		wg.Add(1)
		go func(ps *pteroServer) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// Startparameter und Variablen.
			var st struct {
				Data []struct {
					Attributes struct {
						Env   string `json:"env_variable"`
						Value string `json:"server_value"`
					} `json:"attributes"`
				} `json:"data"`
				Meta struct {
					Raw string `json:"raw_startup_command"`
				} `json:"meta"`
			}
			if conn.call(conn.ClientKey, "GET", "/api/client/servers/"+ps.Identifier+"/startup", nil, &st) == nil {
				for _, v := range st.Data {
					ps.Env[v.Attributes.Env] = v.Attributes.Value
				}
				ps.Startup = st.Meta.Raw
			}
			var res struct {
				Attributes struct {
					State     string `json:"current_state"`
					Resources struct {
						Disk int64 `json:"disk_bytes"`
					} `json:"resources"`
				} `json:"attributes"`
			}
			if conn.call(conn.ClientKey, "GET", "/api/client/servers/"+ps.Identifier+"/resources", nil, &res) == nil {
				ps.State, ps.DiskBytes = res.Attributes.State, res.Attributes.Resources.Disk
			}
			ps.Match = a.matchEgg(conn, ps, installed, catalog)
			mu.Lock()
			out = append(out, ps)
			scan.servers[ps.Identifier] = ps
			mu.Unlock()
		}(ps)
	}
	wg.Wait()
	id := newID(8)
	pteroScans.Lock()
	for k, v := range pteroScans.m {
		if time.Since(v.created) > time.Hour {
			delete(pteroScans.m, k)
		}
	}
	pteroScans.m[id] = scan
	pteroScans.Unlock()
	sortServers(out)
	jsonOut(w, map[string]any{"scan": id, "servers": out, "appKey": conn.AppKey != ""})
}

func sortServers(l []*pteroServer) {
	for i := 1; i < len(l); i++ {
		for j := i; j > 0 && strings.ToLower(l[j].Name) < strings.ToLower(l[j-1].Name); j-- {
			l[j], l[j-1] = l[j-1], l[j]
		}
	}
}

// matchEgg sucht das passende Egg: exakt über den Application-Key, sonst nach Namen.
func (a *App) matchEgg(conn pteroConn, ps *pteroServer, installed map[string]string, catalog []CatalogEgg) *eggMatch {
	if conn.AppKey != "" {
		if e := pteroAppEgg(conn, ps.UUID); e != nil {
			return &eggMatch{Source: "app", Name: e.Name, App: e}
		}
	}
	name := strings.ToLower(strings.TrimSpace(ps.EggName))
	if id, ok := installed[name]; ok {
		return &eggMatch{Source: "installed", EggID: id, Name: ps.EggName}
	}
	for _, c := range catalog {
		if strings.ToLower(c.Name) == name {
			return &eggMatch{Source: "catalog", Name: c.Name, URL: c.URL, AltURL: c.AltURL, Cat: c.Category}
		}
	}
	return &eggMatch{Source: "none", Name: ps.EggName}
}

// pteroAppEgg holt das Egg eines Servers über die Application-API und baut ein PTDL-Egg daraus.
func pteroAppEgg(conn pteroConn, serverUUID string) *Egg {
	var sl struct {
		Data []struct {
			Attributes struct {
				Egg  int `json:"egg"`
				Nest int `json:"nest"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if conn.call(conn.AppKey, "GET", "/api/application/servers?filter[uuid]="+url.QueryEscape(serverUUID), nil, &sl) != nil || len(sl.Data) == 0 {
		return nil
	}
	ids := sl.Data[0].Attributes
	var eg struct {
		Attributes struct {
			Name         string            `json:"name"`
			Author       string            `json:"author"`
			Description  string            `json:"description"`
			DockerImage  string            `json:"docker_image"`
			DockerImages map[string]string `json:"docker_images"`
			Startup      string            `json:"startup"`
			Config       map[string]any    `json:"config"`
			Script       struct {
				Install   string `json:"install"`
				Entry     string `json:"entry"`
				Container string `json:"container"`
			} `json:"script"`
			Relationships struct {
				Variables struct {
					Data []struct {
						Attributes map[string]any `json:"attributes"`
					} `json:"data"`
				} `json:"variables"`
			} `json:"relationships"`
		} `json:"attributes"`
	}
	if conn.call(conn.AppKey, "GET", fmt.Sprintf("/api/application/nests/%d/eggs/%d?include=variables", ids.Nest, ids.Egg), nil, &eg) != nil {
		return nil
	}
	x := eg.Attributes
	imgs := map[string]any{}
	for k, v := range x.DockerImages {
		imgs[k] = v
	}
	if len(imgs) == 0 && x.DockerImage != "" {
		imgs[x.DockerImage] = x.DockerImage
	}
	var vars []any
	for _, v := range x.Relationships.Variables.Data {
		vars = append(vars, v.Attributes)
	}
	raw := map[string]any{
		"meta": map[string]any{"version": "PTDL_v2"}, "name": x.Name, "author": x.Author, "description": x.Description,
		"docker_images": imgs, "startup": x.Startup, "config": x.Config,
		"scripts":   map[string]any{"installation": map[string]any{"script": x.Script.Install, "container": x.Script.Container, "entrypoint": x.Script.Entry}},
		"variables": vars,
	}
	b, _ := json.Marshal(raw)
	e, err := ParseEgg(b)
	if err != nil {
		return nil
	}
	return e
}

type importItem struct {
	Identifier string `json:"identifier"`
	EggID      string `json:"eggId"` // Wahl des Nutzers, falls kein Treffer
	KeepPorts  bool   `json:"keepPorts"`
}

func (a *App) importRun(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Scan    string       `json:"scan"`
		Owner   string       `json:"owner"`
		Servers []importItem `json:"servers"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	pteroScans.Lock()
	scan := pteroScans.m[in.Scan]
	pteroScans.Unlock()
	if scan == nil {
		jsonErr(w, 410, "Suche ist abgelaufen, bitte neu verbinden")
		return
	}
	owner := in.Owner
	if owner == "" || a.st.User(owner) == nil {
		owner = currentUser(r).ID
	}
	var defs ServerDefaults
	a.st.Read(func(st *State) { defs = st.Settings.Defaults })
	type result struct {
		Name  string `json:"name"`
		ID    string `json:"id,omitempty"`
		Error string `json:"error,omitempty"`
	}
	var results []result
	type startItem struct {
		ps *pteroServer
		id string
	}
	var started []startItem
	for _, it := range in.Servers {
		ps := scan.servers[it.Identifier]
		if ps == nil {
			continue
		}
		res := result{Name: ps.Name}
		egg, err := a.resolveImportEgg(ps, it.EggID)
		if err != nil {
			res.Error = err.Error()
			results = append(results, res)
			continue
		}
		ports := ps.Ports
		// Alte Ports nur, wenn sie in den Allocations stehen und frei sind; sonst neue.
		if !it.KeepPorts || a.checkPorts(ports, "") != nil {
			need := max(len(ps.Ports), 1)
			ports = a.suggestPorts(need)
			if len(ports) < need {
				res.Error = "nicht genug freie Ports in den Allocations"
				results = append(results, res)
				continue
			}
		}
		env := egg.DefaultEnv()
		for _, v := range egg.Variables {
			if val, ok := ps.Env[v.Env]; ok {
				env[v.Env] = val
			}
		}
		img := ps.Image
		if img == "" && len(egg.Images) > 0 {
			img = egg.Images[0].Image
		}
		startup := ""
		if ps.Startup != "" && ps.Startup != egg.Startup {
			startup = ps.Startup
		}
		srv := &Server{
			ID: newUUID(), Name: ps.Name, Description: ps.Desc, OwnerID: owner, EggID: egg.ID, Image: img, Startup: startup,
			Env: env, Memory: ps.Memory, CPU: ps.CPU, Disk: ps.Disk, Ports: ports, AutoRestart: defs.AutoRestart,
			MaxBackups: defs.MaxBackups, Install: "importing", Created: time.Now(), ImportedFrom: scan.conn.Base + "#" + ps.UUID, ImportedPorts: ps.Ports,
		}
		dir := a.m.ServerDir(srv.ID)
		os.MkdirAll(dir, 0755)
		a.st.Update(func(st *State) error { st.Servers = append(st.Servers, srv); return nil })
		ps.Imported = srv.ID
		res.ID = srv.ID
		results = append(results, res)
		started = append(started, startItem{ps, srv.ID})
	}
	// Alte Ports aus Pterodactyl -> neue Ports hier (auch frühere Importe vom selben Panel).
	portMap := map[int]int{}
	a.st.Read(func(st *State) {
		for _, x := range st.Servers {
			if strings.HasPrefix(x.ImportedFrom, scan.conn.Base+"#") {
				for i, p := range x.ImportedPorts {
					if i < len(x.Ports) {
						portMap[p] = x.Ports[i]
					}
				}
			}
		}
	})
	for _, s := range started {
		go a.importFiles(scan.conn, s.ps, s.id, portMap)
	}
	jsonOut(w, map[string]any{"results": results})
}

// rewriteBackends passt Adressen in den Konfigdateien eines Proxys an: Server, die
// vorher in Pterodactyl liefen und mit importiert wurden, sind jetzt hier zu finden.
func rewriteBackends(dir string, egg *Egg, hosts []string, portMap map[int]int, gateway string) []string {
	if len(portMap) == 0 || egg == nil {
		return nil
	}
	var alts []string
	for _, h := range hosts {
		if h != "" {
			alts = append(alts, regexp.QuoteMeta(h))
		}
	}
	re := regexp.MustCompile(`\b(` + strings.Join(alts, "|") + `):(\d{1,5})\b`)
	var changed []string
	for name := range egg.Files {
		p, err := safeJoin(dir, name)
		if err != nil {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		out := re.ReplaceAllStringFunc(string(b), func(m string) string {
			sm := re.FindStringSubmatch(m)
			old, _ := strconv.Atoi(sm[2])
			if np, ok := portMap[old]; ok {
				changed = append(changed, fmt.Sprintf("%s -> %s:%d", m, gateway, np))
				return fmt.Sprintf("%s:%d", gateway, np)
			}
			return m
		})
		if out != string(b) {
			os.WriteFile(p, []byte(out), 0644)
		}
	}
	return changed
}

func (a *App) resolveImportEgg(ps *pteroServer, chosen string) (*Egg, error) {
	if chosen != "" {
		if e := a.st.Egg(chosen); e != nil {
			return e, nil
		}
		return nil, fmt.Errorf("gewähltes Egg gibt es nicht")
	}
	m := ps.Match
	if m == nil {
		return nil, fmt.Errorf("kein passendes Egg, bitte eins wählen")
	}
	switch m.Source {
	case "installed":
		if e := a.st.Egg(m.EggID); e != nil {
			return e, nil
		}
	case "app":
		e := *m.App
		e.Source = "Pterodactyl"
		e.Category = firstNonEmpty(categoryFromTags(&e), "Importiert")
		saved := a.saveEgg(&e, "")
		m.Source, m.EggID = "installed", saved.ID
		return a.st.Egg(saved.ID), nil
	case "catalog":
		urls := urlVariants(m.URL)
		if m.AltURL != "" {
			urls = append(urls, urlVariants(m.AltURL)...)
		}
		e, src, err := FetchEgg(urls...)
		if err != nil {
			return nil, fmt.Errorf("Egg %s laden: %v", m.Name, err)
		}
		e.Source, e.Category = src, m.Cat
		saved := a.saveEgg(e, "")
		m.Source, m.EggID = "installed", saved.ID
		return a.st.Egg(saved.ID), nil
	}
	return nil, fmt.Errorf("kein passendes Egg für %q, bitte eins wählen", ps.EggName)
}

// importFiles holt die Dateien: lokal per Kopie, sonst als Archiv über die Pterodactyl-API.
func (a *App) importFiles(conn pteroConn, ps *pteroServer, id string, portMap map[int]int) {
	in := a.m.Get(id)
	in.setState(StInstalling)
	dir := a.m.ServerDir(id)
	err := func() error {
		local := filepath.Join(pteroVolumes, ps.UUID)
		if fi, err := os.Stat(local); err == nil && fi.IsDir() {
			in.Say("Dateien werden direkt von %s kopiert ...", local)
			out, err := exec.Command("cp", "-a", local+"/.", dir+"/").CombinedOutput()
			if err != nil {
				return fmt.Errorf("kopieren: %v %s", err, strings.TrimSpace(string(out)))
			}
			return nil
		}
		in.Say("Dateien werden in Pterodactyl gepackt (kann bei großen Servern dauern) ...")
		var list struct {
			Data []struct {
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
		}
		if err := conn.call(conn.ClientKey, "GET", "/api/client/servers/"+ps.Identifier+"/files/list?directory=%2F", nil, &list); err != nil {
			return err
		}
		var names []string
		for _, f := range list.Data {
			names = append(names, f.Attributes.Name)
		}
		if len(names) == 0 {
			in.Say("Server in Pterodactyl hat keine Dateien.")
			return nil
		}
		var arc struct {
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		}
		if err := conn.call(conn.ClientKey, "POST", "/api/client/servers/"+ps.Identifier+"/files/compress", map[string]any{"root": "/", "files": names}, &arc); err != nil {
			return fmt.Errorf("packen: %v", err)
		}
		name := arc.Attributes.Name
		defer conn.call(conn.ClientKey, "POST", "/api/client/servers/"+ps.Identifier+"/files/delete", map[string]any{"root": "/", "files": []string{name}}, nil)
		var dl struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		}
		if err := conn.call(conn.ClientKey, "GET", "/api/client/servers/"+ps.Identifier+"/files/download?file="+url.QueryEscape("/"+name), nil, &dl); err != nil {
			return fmt.Errorf("Download-Link: %v", err)
		}
		in.Say("Archiv %s wird geladen ...", name)
		hc := conn.client(6 * time.Hour)
		resp, err := hc.Get(dl.Attributes.URL)
		if err != nil {
			// Wings-Adresse von hier nicht erreichbar (z. B. VM per macvtap):
			// gleichen Port über die Adresse probieren, unter der das Panel erreichbar war.
			if wu, e1 := url.Parse(dl.Attributes.URL); e1 == nil {
				if pu, e2 := url.Parse(conn.Base); e2 == nil && wu.Hostname() != pu.Hostname() {
					alt := *wu
					if p := wu.Port(); p != "" {
						alt.Host = net.JoinHostPort(pu.Hostname(), p)
					} else {
						alt.Host = pu.Hostname()
					}
					in.Say("Wings unter %s nicht erreichbar, versuche %s ...", wu.Host, alt.Host)
					resp, err = hc.Get(alt.String())
				}
			}
		}
		if err != nil {
			return fmt.Errorf("Download von Wings: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("Download: HTTP %d", resp.StatusCode)
		}
		tmp := filepath.Join(a.data, "tmp", "import-"+id+".tar.gz")
		f, err := os.Create(tmp)
		if err != nil {
			return err
		}
		n, err := io.Copy(f, resp.Body)
		f.Close()
		defer os.Remove(tmp)
		if err != nil {
			return fmt.Errorf("Download abgebrochen: %v", err)
		}
		in.Say("%s geladen, wird entpackt ...", humanBytes(uint64(n)))
		return extractTar(tmp, dir, dir)
	}()
	if err == nil {
		if srv := a.st.Server(id); srv != nil {
			host := ""
			if u, e := url.Parse(conn.Base); e == nil {
				host = u.Hostname()
			}
			for _, ch := range rewriteBackends(dir, a.st.Egg(srv.EggID), []string{"172.18.0.1", host, "127.0.0.1", "localhost"}, portMap, a.m.gateway) {
				in.Say("Adresse angepasst: %s", ch)
			}
		}
	}
	chownTree(dir)
	res := "ok"
	if err != nil {
		res = "failed"
		in.Say("Import fehlgeschlagen: %v", err)
	} else {
		in.Say("Import fertig. Achte darauf, den alten Server in Pterodactyl aus zu lassen, sonst laufen beide.")
	}
	a.st.Update(func(st *State) error {
		if x := findServer(st, id); x != nil {
			x.Install = res
		}
		return nil
	})
	in.setState(StOffline)
	in.mu.Lock()
	in.emitLocked(Event{"install", res})
	in.mu.Unlock()
}
