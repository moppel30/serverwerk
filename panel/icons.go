package main

// Logos für Eggs: eigenes Icon aus dem Egg (Pelican), sonst das Bild
// aus dem Steam-Shop, sonst nichts (die Oberfläche zeigt dann Initialen).
// Alles wird in <daten>/icons zwischengespeichert.

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type iconCache struct {
	dir  string
	sem  chan struct{}
	mu   sync.Mutex
	busy map[string]*sync.WaitGroup
}

func newIconCache(data string) *iconCache {
	d := filepath.Join(data, "icons")
	os.MkdirAll(d, 0755)
	return &iconCache{dir: d, sem: make(chan struct{}, 6), busy: map[string]*sync.WaitGroup{}}
}

type iconData struct {
	Type string
	Body []byte
}

func keyOf(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:10])
}

func decodeDataURL(s string) *iconData {
	if !strings.HasPrefix(s, "data:") {
		return nil
	}
	meta, b64, ok := strings.Cut(strings.TrimPrefix(s, "data:"), ",")
	if !ok || !strings.HasSuffix(meta, ";base64") {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil
	}
	return &iconData{Type: strings.TrimSuffix(meta, ";base64"), Body: b}
}

func fetchImage(u string) *iconData {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Serverwerk-Panel")
	resp, err := httpc.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if resp.StatusCode != 200 || !strings.HasPrefix(ct, "image/") {
		return nil
	}
	b, err := readAllLimit(resp, 3<<20)
	if err != nil {
		return nil
	}
	return &iconData{Type: ct, Body: b}
}

func readAllLimit(resp *http.Response, n int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(resp.Body, n))
}

var nameNoise = regexp.MustCompile(`(?i)\b(dedicated|server|vanilla|steamcmd|linux|windows|wine|proton|modded|egg|beta|experimental)\b|\(.*?\)|\[.*?\]`)

// steamImage sucht das Spiel im Steam-Shop und holt das Kopfbild.
func steamImage(name string) *iconData {
	term := strings.Join(strings.Fields(nameNoise.ReplaceAllString(name, " ")), " ")
	if len(term) < 2 {
		return nil
	}
	b, err := httpGet("https://store.steampowered.com/api/storesearch/?cc=us&l=en&term="+url.QueryEscape(term), 1<<20)
	if err != nil {
		return nil
	}
	var r struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"items"`
	}
	if json.Unmarshal(b, &r) != nil {
		return nil
	}
	first := strings.ToLower(strings.Fields(term)[0])
	for _, it := range r.Items {
		low := strings.ToLower(it.Name)
		if it.Type != "app" || !strings.Contains(low, first) || strings.Contains(low, "soundtrack") {
			continue
		}
		for _, u := range []string{
			fmt.Sprintf("https://shared.cloudflare.steamstatic.com/store_item_assets/steam/apps/%d/header.jpg", it.ID),
			fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%d/header.jpg", it.ID),
		} {
			if img := fetchImage(u); img != nil {
				return img
			}
		}
		return nil
	}
	return nil
}

func gameCategory(cat string) bool {
	c := strings.ToLower(cat)
	return strings.Contains(c, "spiel") || strings.Contains(c, "game") || strings.Contains(c, "steam")
}

// get liefert das Bild zu einem Schlüssel; find wird nur einmal pro Schlüssel ausgeführt.
func (ic *iconCache) get(key string, find func() *iconData) *iconData {
	p := filepath.Join(ic.dir, key)
	if b, err := os.ReadFile(p); err == nil {
		if len(b) == 0 {
			// Negativ-Ergebnisse nach einem Tag neu versuchen.
			if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) < 24*time.Hour {
				return nil
			}
			return ic.find(key, p, find)
		}
		t, _ := os.ReadFile(p + ".typ")
		return &iconData{Type: string(t), Body: b}
	}
	return ic.find(key, p, find)
}

func (ic *iconCache) find(key, p string, find func() *iconData) *iconData {
	ic.mu.Lock()
	if wg, ok := ic.busy[key]; ok {
		ic.mu.Unlock()
		wg.Wait()
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			t, _ := os.ReadFile(p + ".typ")
			return &iconData{Type: string(t), Body: b}
		}
		return nil
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	ic.busy[key] = wg
	ic.mu.Unlock()
	defer func() {
		ic.mu.Lock()
		delete(ic.busy, key)
		ic.mu.Unlock()
		wg.Done()
	}()
	ic.sem <- struct{}{}
	img := find()
	<-ic.sem
	if img == nil {
		os.WriteFile(p, nil, 0644)
		return nil
	}
	os.WriteFile(p+".typ", []byte(img.Type), 0644)
	os.WriteFile(p, img.Body, 0644)
	return img
}

func serveIcon(w http.ResponseWriter, img *iconData) {
	if img == nil {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "kein Bild", 404)
		return
	}
	w.Header().Set("Content-Type", img.Type)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.Write(img.Body)
}

func (a *App) eggIcon(w http.ResponseWriter, r *http.Request) {
	e := a.st.Egg(r.PathValue("id"))
	if e == nil {
		serveIcon(w, nil)
		return
	}
	img := a.icons.get("egg3-"+e.ID+"-"+keyOf(e.Icon+e.Name), func() *iconData {
		full := e
		// Vor Version 0.2 importierte Eggs: Logo aus der Quelle nachholen.
		if full.Icon == "" {
			for _, src := range []string{e.Source, e.UpdateURL} {
				if strings.HasPrefix(src, "https://") {
					if x := fetchEggFrom(iconURLs(src)); x != nil && x.Icon != "" {
						full = x
						break
					}
				}
			}
		}
		return findIcon(full, e.Name, e.Category)
	})
	serveIcon(w, img)
}

// findIcon: eigenes Logo, sonst Steam (bei Spielen), sonst Logo der Projektseite.
func findIcon(e *Egg, name, category string) *iconData {
	if e != nil && e.Icon != "" {
		if d := decodeDataURL(e.Icon); d != nil {
			return d
		}
		if d := fetchImage(e.Icon); d != nil {
			return d
		}
	}
	if gameCategory(category) {
		if d := steamImage(name); d != nil {
			return d
		}
	}
	if e != nil {
		if d := siteIcon(e); d != nil {
			return d
		}
		return githubIcon(e, name)
	}
	return nil
}

var ghRe = regexp.MustCompile(`(?i)(?:github\.com/|api\.github\.com/repos/|raw\.githubusercontent\.com/)([A-Za-z0-9-]+)/([A-Za-z0-9_.-]+)`)

var ghSkip = map[string]bool{"pelican-eggs": true, "parkervcp": true, "pterodactyl": true, "pelican-dev": true, "ptero-eggs": true, "orgs": true, "repos": true, "user-attachments": true}

func alnum(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// githubIcon nimmt das Bild der GitHub-Organisation, wenn sie zum Egg-Namen passt
// (z. B. PowerNukkitX/PowerNukkitX), damit keine fremden Profilbilder auftauchen.
func githubIcon(e *Egg, name string) *iconData {
	words := strings.Fields(nameNoise.ReplaceAllString(name, " "))
	if len(words) == 0 {
		return nil
	}
	first := alnum(words[0])
	if len(first) < 3 {
		return nil
	}
	for _, text := range []string{e.Description, e.InstScript} {
		for _, m := range ghRe.FindAllStringSubmatch(text, -1) {
			owner, repo := m[1], m[2]
			if ghSkip[strings.ToLower(owner)] {
				continue
			}
			if strings.Contains(alnum(owner), first) || strings.HasPrefix(alnum(repo), first) {
				if d := fetchImage("https://github.com/" + owner + ".png?size=200"); d != nil {
					return d
				}
			}
		}
	}
	return nil
}

func fetchEggFrom(urls []string) *Egg {
	for _, x := range urls {
		if !strings.HasPrefix(x, "https://") {
			continue
		}
		b, err := httpGet(x, 4<<20)
		if err != nil {
			continue
		}
		if e, err := ParseEgg(b); err == nil {
			return e
		}
	}
	return nil
}

var urlRe = regexp.MustCompile(`https?://([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)

var skipHosts = []string{"github", "githubusercontent", "ghcr.io", "gitlab", "docker", "pterodactyl", "pelican", "jsdelivr", "cloudflare", "amazonaws", "googleapis", "steamstatic", "steamcdn", "discord", "localhost", "example.", "imgur", "npmjs", "pypi", "sourceforge", "archive.org", "bit.ly", "adoptium", "java.com", "oracle.com", "apt.", "deb.", "nodesource", "python.org", "ubuntu", "debian", "alpinelinux", "mozilla", "wine", "curl.se", "tinyurl", "youtube"}

// siteIcon nimmt die erste Projektadresse aus Beschreibung oder Skript und holt deren Logo.
func siteIcon(e *Egg) *iconData {
	seen := map[string]bool{}
	for _, text := range []string{e.Description, e.InstScript, e.Startup} {
		for _, m := range urlRe.FindAllStringSubmatch(text, -1) {
			host := strings.ToLower(m[1])
			bad := false
			for _, sk := range skipHosts {
				if strings.Contains(host, sk) {
					bad = true
				}
			}
			parts := strings.Split(host, ".")
			if len(parts) > 2 {
				host = strings.Join(parts[len(parts)-2:], ".")
			}
			if bad || seen[host] {
				continue
			}
			seen[host] = true
			if d := fetchImage("https://www.google.com/s2/favicons?sz=128&domain=" + url.QueryEscape(host)); d != nil && bigEnough(d) {
				return d
			}
			if len(seen) >= 3 {
				return nil
			}
		}
	}
	return nil
}

// Der Favicon-Dienst liefert für unbekannte Seiten eine kleine Weltkugel.
func bigEnough(d *iconData) bool {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(d.Body))
	return err == nil && cfg.Width >= 48 && cfg.Height >= 48
}

// iconURLs: Pelican-YAML hat oft ein Icon, die Pterodactyl-JSON nie.
func iconURLs(u string) []string {
	dir, file := path.Split(u)
	out := []string{}
	base := strings.TrimSuffix(strings.TrimSuffix(file, ".json"), ".yaml")
	for _, p := range []string{"egg-pterodactyl-", "pterodactyl-egg-"} {
		if strings.HasPrefix(base, p) {
			base = "egg-" + strings.TrimPrefix(base, p)
		}
	}
	out = append(out, dir+base+".yaml", dir+base+".json")
	return out
}

func (a *App) catalogIcon(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("u")
	list, _ := a.cat.List()
	var ce *CatalogEgg
	for i := range list {
		if list[i].URL == u {
			ce = &list[i]
			break
		}
	}
	if ce == nil {
		serveIcon(w, nil)
		return
	}
	img := a.icons.get("cat3-"+keyOf(u), func() *iconData {
		urls := iconURLs(ce.URL)
		if ce.AltURL != "" {
			urls = append(urls, iconURLs(ce.AltURL)...)
		}
		urls = append(urls, ce.URL, ce.AltURL)
		return findIcon(fetchEggFrom(urls), ce.Name, ce.Category)
	})
	serveIcon(w, img)
}
