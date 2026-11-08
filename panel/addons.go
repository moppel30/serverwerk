package main

// Plugins, Mods und Datapacks für Minecraft-Server über Modrinth.
// Suchen macht der Browser direkt bei Modrinth, herunterladen der Server.

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Addon struct {
	Project string    `json:"project"`
	Title   string    `json:"title"`
	Icon    string    `json:"icon"`
	File    string    `json:"file"`
	Version string    `json:"version"`
	Added   time.Time `json:"added"`
}

type AddonInfo struct {
	Kind      string   `json:"kind"`  // plugin | mod | datapack | ""
	Label     string   `json:"label"` // Plugins, Mods, Datapacks
	Loaders   []string `json:"loaders"`
	Folder    string   `json:"folder"`
	MCVersion string   `json:"mcVersion"`
}

var mcVersionRe = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)

func (a *App) addonInfo(s *Server) AddonInfo {
	e := a.st.Egg(s.EggID)
	if e == nil {
		return AddonInfo{}
	}
	name := strings.ToLower(e.Name + " " + strings.Join(e.Tags, " "))
	cat := strings.ToLower(e.Category)
	if !strings.Contains(cat, "minecraft") && !strings.Contains(name, "minecraft") &&
		!regexp.MustCompile(`paper|purpur|spigot|bukkit|folia|velocity|bungee|waterfall|fabric|forge|quilt`).MatchString(name) {
		return AddonInfo{}
	}
	if strings.Contains(name, "bedrock") || strings.Contains(name, "nukkit") || strings.Contains(name, "pocketmine") || strings.Contains(name, "gomint") {
		return AddonInfo{}
	}
	in := AddonInfo{}
	switch {
	case strings.Contains(name, "velocity"):
		in = AddonInfo{Kind: "plugin", Label: "Plugins", Loaders: []string{"velocity"}, Folder: "plugins"}
	case strings.Contains(name, "bungee") || strings.Contains(name, "waterfall"):
		in = AddonInfo{Kind: "plugin", Label: "Plugins", Loaders: []string{"bungeecord", "waterfall"}, Folder: "plugins"}
	case strings.Contains(name, "neoforge"):
		in = AddonInfo{Kind: "mod", Label: "Mods", Loaders: []string{"neoforge"}, Folder: "mods"}
	case strings.Contains(name, "forge"):
		in = AddonInfo{Kind: "mod", Label: "Mods", Loaders: []string{"forge"}, Folder: "mods"}
	case strings.Contains(name, "quilt"):
		in = AddonInfo{Kind: "mod", Label: "Mods", Loaders: []string{"quilt", "fabric"}, Folder: "mods"}
	case strings.Contains(name, "fabric"):
		in = AddonInfo{Kind: "mod", Label: "Mods", Loaders: []string{"fabric"}, Folder: "mods"}
	case strings.Contains(name, "folia"):
		in = AddonInfo{Kind: "plugin", Label: "Plugins", Loaders: []string{"folia"}, Folder: "plugins"}
	case strings.Contains(name, "purpur"):
		in = AddonInfo{Kind: "plugin", Label: "Plugins", Loaders: []string{"purpur", "paper", "spigot", "bukkit"}, Folder: "plugins"}
	case regexp.MustCompile(`paper|spigot|bukkit|pufferfish|leaf|mohist|arclight|sponge`).MatchString(name):
		in = AddonInfo{Kind: "plugin", Label: "Plugins", Loaders: []string{"paper", "spigot", "bukkit"}, Folder: "plugins"}
	default:
		in = AddonInfo{Kind: "datapack", Label: "Datapacks", Loaders: []string{"datapack"}, Folder: worldName(a.m.ServerDir(s.ID)) + "/datapacks"}
	}
	in.MCVersion = a.detectMCVersion(s, e)
	return in
}

func worldName(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(l), "level-name="); ok && v != "" && !strings.ContainsAny(v, "/\\") {
				return strings.TrimSpace(v)
			}
		}
	}
	return "world"
}

func (a *App) detectMCVersion(s *Server, e *Egg) string {
	env := mergedEnv(s, e)
	for _, k := range []string{"MINECRAFT_VERSION", "MC_VERSION", "VERSION", "VANILLA_VERSION"} {
		if v := strings.TrimSpace(env[k]); mcVersionRe.MatchString(v) {
			return v
		}
	}
	dir := a.m.ServerDir(s.ID)
	// Paper & Co. legen versions/<version>/ an.
	if ents, err := os.ReadDir(filepath.Join(dir, "versions")); err == nil {
		var vs []string
		for _, x := range ents {
			if x.IsDir() && mcVersionRe.MatchString(x.Name()) {
				vs = append(vs, x.Name())
			}
		}
		sort.Slice(vs, func(i, j int) bool { return natLess(vs[i], vs[j]) })
		if len(vs) > 0 {
			return vs[len(vs)-1]
		}
	}
	// Vanilla: version.json steckt im server.jar.
	jar := env["SERVER_JARFILE"]
	if jar == "" {
		jar = "server.jar"
	}
	if p, err := safeJoin(dir, jar); err == nil {
		if zr, err := zip.OpenReader(p); err == nil {
			defer zr.Close()
			for _, f := range zr.File {
				if f.Name == "version.json" {
					rc, err := f.Open()
					if err == nil {
						var v struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						}
						json.NewDecoder(rc).Decode(&v)
						rc.Close()
						if mcVersionRe.MatchString(v.ID) {
							return v.ID
						}
					}
				}
			}
		}
	}
	return ""
}

func (a *App) addonsGet(w http.ResponseWriter, r *http.Request, s *Server) {
	info := a.addonInfo(s)
	out := map[string]any{"info": info, "installed": []any{}}
	if info.Kind == "" {
		jsonOut(w, out)
		return
	}
	known := map[string]Addon{}
	for _, ad := range s.Addons {
		known[ad.File] = ad
	}
	type row struct {
		File  string `json:"file"`
		Size  int64  `json:"size"`
		Mod   string `json:"mod"`
		Addon *Addon `json:"addon,omitempty"`
		Off   bool   `json:"off"`
	}
	rows := []row{}
	if dir, err := safeJoin(a.m.ServerDir(s.ID), info.Folder); err == nil {
		ents, _ := os.ReadDir(dir)
		for _, e := range ents {
			n := e.Name()
			low := strings.ToLower(n)
			if e.IsDir() && info.Kind != "datapack" {
				continue
			}
			if !(strings.HasSuffix(low, ".jar") || strings.HasSuffix(low, ".zip") || strings.HasSuffix(low, ".disabled") || (info.Kind == "datapack" && e.IsDir())) {
				continue
			}
			fi, _ := e.Info()
			rw := row{File: n, Off: strings.HasSuffix(low, ".disabled")}
			if fi != nil {
				rw.Size, rw.Mod = fi.Size(), fi.ModTime().Format(time.RFC3339)
			}
			if ad, ok := known[strings.TrimSuffix(n, ".disabled")]; ok {
				ad := ad
				rw.Addon = &ad
			}
			rows = append(rows, rw)
		}
	}
	out["installed"] = rows
	jsonOut(w, out)
}

var safeFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._+\-()\[\]]{0,150}\.(jar|zip)$`)

func (a *App) addonInstall(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct {
		URL     string `json:"url"`
		File    string `json:"file"`
		Project string `json:"project"`
		Title   string `json:"title"`
		Icon    string `json:"icon"`
		Version string `json:"version"`
		Replace string `json:"replace"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	info := a.addonInfo(s)
	if info.Kind == "" {
		jsonErr(w, 400, "Dieser Server unterstützt keine Erweiterungen")
		return
	}
	u, err := url.Parse(in.URL)
	if err != nil || u.Scheme != "https" || (u.Host != "cdn.modrinth.com" && u.Host != "cdn-raw.modrinth.com") {
		jsonErr(w, 400, "Nur Downloads von Modrinth erlaubt")
		return
	}
	if !safeFile.MatchString(in.File) {
		jsonErr(w, 400, "ungültiger Dateiname")
		return
	}
	dir, err := safeJoin(a.m.ServerDir(s.ID), info.Folder)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	os.MkdirAll(dir, 0755)
	chownTree(dir)
	req, _ := http.NewRequest("GET", in.URL, nil)
	req.Header.Set("User-Agent", "Serverwerk-Panel/"+a.version)
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		jsonErr(w, 502, fmt.Sprintf("Download fehlgeschlagen (HTTP %d)", resp.StatusCode))
		return
	}
	tmp, err := os.CreateTemp(dir, ".sw-dl-*")
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	_, err = io.Copy(tmp, io.LimitReader(resp.Body, 512<<20))
	tmp.Close()
	if err != nil {
		os.Remove(tmp.Name())
		jsonErr(w, 502, err.Error())
		return
	}
	target := filepath.Join(dir, in.File)
	// Alte Version desselben Projekts ersetzen.
	if in.Replace != "" && in.Replace != in.File && safeFile.MatchString(in.Replace) {
		os.Remove(filepath.Join(dir, in.Replace))
		os.Remove(filepath.Join(dir, in.Replace+".disabled"))
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		os.Remove(tmp.Name())
		jsonErr(w, 500, err.Error())
		return
	}
	os.Chmod(target, 0644)
	os.Lchown(target, containerUID, containerUID)
	// Datapacks als Ordner entpacken? Nein: Minecraft liest .zip direkt.
	ad := Addon{Project: in.Project, Title: in.Title, Icon: in.Icon, File: in.File, Version: in.Version, Added: time.Now()}
	a.st.Update(func(st *State) error {
		x := findServer(st, s.ID)
		if x == nil {
			return nil
		}
		keep := x.Addons[:0]
		for _, o := range x.Addons {
			if o.Project != ad.Project && o.File != ad.File {
				keep = append(keep, o)
			}
		}
		x.Addons = append(keep, ad)
		return nil
	})
	a.m.Get(s.ID).Say("%s installiert (%s). Wird beim nächsten Neustart geladen.", firstNonEmpty(in.Title, in.File), in.File)
	jsonOut(w, ad)
}

func (a *App) addonRemove(w http.ResponseWriter, r *http.Request, s *Server) {
	var in struct {
		File   string `json:"file"`
		Toggle bool   `json:"toggle"`
	}
	readJSON(r, &in)
	info := a.addonInfo(s)
	base := strings.TrimSuffix(in.File, ".disabled")
	if info.Kind == "" || strings.ContainsAny(in.File, "/\\") || in.File == "" || strings.HasPrefix(in.File, ".") {
		jsonErr(w, 400, "ungültige Datei")
		return
	}
	dir, err := safeJoin(a.m.ServerDir(s.ID), info.Folder)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	p := filepath.Join(dir, in.File)
	if in.Toggle {
		// An/aus: Endung .disabled anhängen oder entfernen.
		to := p + ".disabled"
		if strings.HasSuffix(in.File, ".disabled") {
			to = filepath.Join(dir, base)
		}
		if err := os.Rename(p, to); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		ok(w)
		return
	}
	if err := os.RemoveAll(p); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	a.st.Update(func(st *State) error {
		if x := findServer(st, s.ID); x != nil {
			keep := x.Addons[:0]
			for _, o := range x.Addons {
				if o.File != base {
					keep = append(keep, o)
				}
			}
			x.Addons = keep
		}
		return nil
	})
	ok(w)
}
