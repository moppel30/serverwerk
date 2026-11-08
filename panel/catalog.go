package main

// Egg-Katalog aus dem Index der Community (pelican-eggs.github.io).

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CatalogEgg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	AltURL      string `json:"altUrl,omitempty"`
}

type Catalog struct {
	mu      sync.Mutex
	path    string
	Fetched time.Time    `json:"fetched"`
	Eggs    []CatalogEgg `json:"eggs"`
}

const (
	indexPelican     = "https://raw.githubusercontent.com/pelican-eggs/pelican-eggs.github.io/main/content/pelican.json"
	indexPterodactyl = "https://raw.githubusercontent.com/pelican-eggs/pelican-eggs.github.io/main/content/pterodactyl.json"
)

var httpc = &http.Client{Timeout: 60 * time.Second}

func httpGet(url string, max int64) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Serverwerk-Panel")
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func loadCatalog(dir string) *Catalog {
	c := &Catalog{path: filepath.Join(dir, "katalog.json")}
	if b, err := os.ReadFile(c.path); err == nil {
		json.Unmarshal(b, c)
	}
	return c
}

type rawIndex struct {
	Nests []struct {
		Type string `json:"nest_type"`
		Eggs []struct {
			Egg struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"egg"`
			URL string `json:"download_url"`
		} `json:"Eggs"`
	} `json:"nests"`
}

func fetchIndex(url string) (*rawIndex, error) {
	b, err := httpGet(url, 16<<20)
	if err != nil {
		return nil, err
	}
	var ri rawIndex
	if err := json.Unmarshal(b, &ri); err != nil {
		return nil, err
	}
	return &ri, nil
}

var categoryNames = map[string]string{
	"Chatbots": "Bots", "Database": "Datenbanken", "Games Standalone": "Spiele", "Games SteamCMD": "Spiele (Steam)",
	"Generic": "Allgemein", "Minecraft": "Minecraft", "Monitoring": "Überwachung", "Software": "Software",
	"Storage": "Speicher", "Voice": "Sprachserver", "Tooling": "Werkzeuge",
}

func (c *Catalog) Refresh() error {
	pel, err1 := fetchIndex(indexPelican)
	ptero, err2 := fetchIndex(indexPterodactyl)
	if err1 != nil && err2 != nil {
		return fmt.Errorf("Katalog nicht erreichbar: %v", err1)
	}
	byKey := map[string]*CatalogEgg{}
	var out []CatalogEgg
	add := func(ri *rawIndex, alt bool) {
		if ri == nil {
			return
		}
		for _, n := range ri.Nests {
			cat := n.Type
			for k, v := range categoryNames {
				if strings.EqualFold(k, n.Type) {
					cat = v
				}
			}
			for _, e := range n.Eggs {
				key := strings.ToLower(n.Type + "|" + e.Egg.Name)
				if ex, ok := byKey[key]; ok {
					if alt && ex.AltURL == "" {
						ex.AltURL = e.URL
					}
					continue
				}
				out = append(out, CatalogEgg{Name: e.Egg.Name, Description: strings.TrimSpace(e.Egg.Description), Category: cat, URL: e.URL})
				byKey[key] = &out[len(out)-1]
			}
		}
	}
	// Erst alle sammeln, dann Zeiger setzen (append verschiebt sonst den Speicher).
	out = make([]CatalogEgg, 0, 2000)
	add(pel, false)
	add(ptero, true)
	c.mu.Lock()
	c.Eggs = out
	c.Fetched = time.Now()
	b, _ := json.Marshal(c)
	c.mu.Unlock()
	return os.WriteFile(c.path, b, 0644)
}

func (c *Catalog) List() ([]CatalogEgg, time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Eggs {
		for k, v := range categoryNames {
			if strings.EqualFold(k, c.Eggs[i].Category) {
				c.Eggs[i].Category = v
			}
		}
	}
	return c.Eggs, c.Fetched
}

// FetchEgg lädt ein Egg aus dem Katalog; nimmt die Ersatzadresse, wenn die erste fehlt.
func FetchEgg(urls ...string) (*Egg, string, error) {
	var lastErr error
	for _, u := range urls {
		if u == "" {
			continue
		}
		if !strings.HasPrefix(u, "https://") {
			return nil, "", fmt.Errorf("nur https-Adressen")
		}
		b, err := httpGet(u, 4<<20)
		if err != nil {
			lastErr = err
			continue
		}
		e, err := ParseEgg(b)
		if err != nil {
			lastErr = err
			continue
		}
		return e, u, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("keine Adresse")
	}
	return nil, "", lastErr
}

var catEggCache sync.Map

// catalogEgg lädt ein Egg aus dem Katalog nur zum Ansehen (ohne Import).
func (a *App) catalogEgg(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("u")
	list, _ := a.cat.List()
	var ce *CatalogEgg
	for i := range list {
		if list[i].URL == u {
			ce = &list[i]
		}
	}
	if ce == nil {
		jsonErr(w, 404, "Nicht im Katalog")
		return
	}
	if v, ok := catEggCache.Load(u); ok {
		jsonOut(w, struct {
			*Egg
			Compatible *bool  `json:"compatible"`
			Arch       string `json:"arch"`
		}{v.(*Egg), a.catalogCompat(u), hostArch()})
		return
	}
	urls := urlVariants(ce.URL)
	if ce.AltURL != "" {
		urls = append(urls, urlVariants(ce.AltURL)...)
	}
	e, src, err := FetchEgg(urls...)
	if err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	e.Source = src
	e.Category = ce.Category
	e.Icon = ""
	catEggCache.Store(u, e)
	jsonOut(w, struct {
		*Egg
		Compatible *bool  `json:"compatible"`
		Arch       string `json:"arch"`
	}{e, a.catalogCompat(u), hostArch()})
}
