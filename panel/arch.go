package main

// Prüft, ob ein Egg auf diesem Rechner laufen kann: Docker verrät über
// /distribution/<image>/json, für welche Prozessoren ein Image gebaut ist.
// Ergebnisse werden in <daten>/architektur.json gespeichert.

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func hostArch() string {
	if v := os.Getenv("SERVERWERK_ARCH"); v != "" { // nur zum Testen
		return v
	}
	return runtime.GOARCH
}

type archEntry struct {
	Arches  []string  `json:"arches"`
	Checked time.Time `json:"checked"`
	Failed  bool      `json:"failed,omitempty"`
}

type catImages struct {
	Images  []string  `json:"images"`
	Install string    `json:"install"`
	Checked time.Time `json:"checked"`
}

type archCache struct {
	mu      sync.Mutex
	path    string
	Images  map[string]*archEntry `json:"images"`
	Catalog map[string]*catImages `json:"catalog"`
	running bool
	Done    int `json:"-"`
	Total   int `json:"-"`
}

func loadArchCache(dir string) *archCache {
	c := &archCache{path: filepath.Join(dir, "architektur.json"), Images: map[string]*archEntry{}, Catalog: map[string]*catImages{}}
	if b, err := os.ReadFile(c.path); err == nil {
		json.Unmarshal(b, c)
	}
	if c.Images == nil {
		c.Images = map[string]*archEntry{}
	}
	if c.Catalog == nil {
		c.Catalog = map[string]*catImages{}
	}
	return c
}

func (c *archCache) save() {
	c.mu.Lock()
	b, _ := json.Marshal(c)
	c.mu.Unlock()
	os.WriteFile(c.path, b, 0644)
}

// imageArches fragt Docker nach den Plattformen eines Images (mit Cache).
func (a *App) imageArches(ref string) ([]string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, false
	}
	c := a.arch
	c.mu.Lock()
	e := c.Images[ref]
	c.mu.Unlock()
	if e != nil && ((!e.Failed && time.Since(e.Checked) < 7*24*time.Hour) || (e.Failed && time.Since(e.Checked) < 6*time.Hour)) {
		return e.Arches, !e.Failed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var out struct {
		Platforms []struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"Platforms"`
	}
	err := a.dk.call(ctx, "GET", "/distribution/"+url.PathEscape(ref)+"/json", nil, &out)
	ne := &archEntry{Checked: time.Now()}
	if err != nil {
		ne.Failed = true
	} else {
		seen := map[string]bool{}
		for _, p := range out.Platforms {
			if p.OS == "linux" && p.Architecture != "" && p.Architecture != "unknown" && !seen[p.Architecture] {
				seen[p.Architecture] = true
				ne.Arches = append(ne.Arches, p.Architecture)
			}
		}
		if len(ne.Arches) == 0 {
			ne.Failed = true
		}
	}
	c.mu.Lock()
	c.Images[ref] = ne
	c.mu.Unlock()
	return ne.Arches, !ne.Failed
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// compat: true = läuft hier, false = läuft nicht, nil = unbekannt.
func (a *App) compatOf(images []string, install string, hasScript bool) *bool {
	arch := hostArch()
	known := false
	anyOK := false
	for _, im := range images {
		if l, ok := a.imageArches(im); ok {
			known = true
			if contains(l, arch) {
				anyOK = true
				break
			}
		}
	}
	if !known {
		return nil
	}
	res := anyOK
	if res && hasScript && install != "" {
		if l, ok := a.imageArches(install); ok && !contains(l, arch) {
			res = false
		}
	}
	return &res
}

// eggCompatCached liefert das Ergebnis nur aus dem Cache (für schnelle Listen).
func (a *App) cachedCompat(images []string, install string, hasScript bool) *bool {
	arch := hostArch()
	c := a.arch
	c.mu.Lock()
	defer c.mu.Unlock()
	known, anyOK := false, false
	for _, im := range images {
		if e := c.Images[im]; e != nil && !e.Failed {
			known = true
			if contains(e.Arches, arch) {
				anyOK = true
			}
		}
	}
	if !known {
		return nil
	}
	res := anyOK
	if res && hasScript && install != "" {
		if e := c.Images[install]; e != nil && !e.Failed && !contains(e.Arches, arch) {
			res = false
		}
	}
	return &res
}

func (a *App) eggCompat(e *Egg) *bool {
	var imgs []string
	for _, im := range e.Images {
		imgs = append(imgs, im.Image)
	}
	return a.cachedCompat(imgs, e.InstImage, strings.TrimSpace(e.InstScript) != "")
}

func (a *App) catalogCompat(u string) *bool {
	a.arch.mu.Lock()
	ci := a.arch.Catalog[u]
	a.arch.mu.Unlock()
	if ci == nil {
		return nil
	}
	return a.cachedCompat(ci.Images, ci.Install, ci.Install != "")
}

// archScan prüft im Hintergrund alle installierten Eggs und den Katalog.
func (a *App) archScan() {
	c := a.arch
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		c.save()
	}()
	// Installierte Eggs.
	var eggs []*Egg
	a.st.Read(func(s *State) {
		for _, e := range s.Eggs {
			x := *e
			eggs = append(eggs, &x)
		}
	})
	for _, e := range eggs {
		var imgs []string
		for _, im := range e.Images {
			imgs = append(imgs, im.Image)
		}
		a.compatOf(imgs, e.InstImage, strings.TrimSpace(e.InstScript) != "")
	}
	// Katalog: Egg-Dateien laden (einmal), dann die Images prüfen.
	list, _ := a.cat.List()
	c.mu.Lock()
	c.Total, c.Done = len(list), 0
	c.mu.Unlock()
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, ce := range list {
		ce := ce
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			c.mu.Lock()
			ci := c.Catalog[ce.URL]
			c.mu.Unlock()
			if ci == nil || time.Since(ci.Checked) > 7*24*time.Hour {
				urls := urlVariants(ce.URL)
				if ce.AltURL != "" {
					urls = append(urls, urlVariants(ce.AltURL)...)
				}
				if e, _, err := FetchEgg(urls...); err == nil {
					ci = &catImages{Checked: time.Now()}
					for _, im := range e.Images {
						ci.Images = append(ci.Images, im.Image)
					}
					if strings.TrimSpace(e.InstScript) != "" {
						ci.Install = e.InstImage
					}
					c.mu.Lock()
					c.Catalog[ce.URL] = ci
					c.mu.Unlock()
				}
			}
			if ci != nil {
				a.compatOf(ci.Images, ci.Install, ci.Install != "")
			}
			c.mu.Lock()
			c.Done++
			done := c.Done
			c.mu.Unlock()
			if done%40 == 0 {
				c.save()
			}
		}()
	}
	wg.Wait()
}

func (a *App) archLoop() {
	for {
		a.archScan()
		time.Sleep(24 * time.Hour)
	}
}

func (a *App) archStatus() map[string]any {
	c := a.arch
	c.mu.Lock()
	defer c.mu.Unlock()
	return map[string]any{"arch": hostArch(), "scanning": c.running, "done": c.Done, "total": c.Total}
}
