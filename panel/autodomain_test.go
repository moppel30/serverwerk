package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSlugName(t *testing.T) {
	for in, want := range map[string]string{"Survival": "survival", "Survival 2 (Test)": "survival-2-test", "Größe & Spaß": "groesse-spass", "***": "server", "  a--b  ": "a-b"} {
		if got := slugName(in); got != want {
			t.Errorf("slugName(%q) = %q, want %q", in, got, want)
		}
	}
}

// Nachgebaute Cloudflare-API, die DNS-Einträge im Speicher hält.
type fakeCF struct {
	sync.Mutex
	recs []cfRecord
	n    int
	down bool
}

func (f *fakeCF) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.Lock()
	defer f.Unlock()
	if f.down {
		w.WriteHeader(502)
		w.Write([]byte(`{"success":false,"errors":[{"code":1,"message":"down"}]}`))
		return
	}
	out := func(v any) {
		b, _ := json.Marshal(v)
		json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": json.RawMessage(b)})
	}
	q := r.URL.Query()
	switch r.Method {
	case "GET":
		res := []cfRecord{}
		for _, x := range f.recs {
			if (q.Get("type") == "" || x.Type == q.Get("type")) && (q.Get("name") == "" || x.Name == q.Get("name")) && (q.Get("comment") == "" || x.Comment == q.Get("comment")) {
				res = append(res, x)
			}
		}
		out(res)
	case "POST":
		var rec cfRecord
		json.NewDecoder(r.Body).Decode(&rec)
		f.n++
		rec.ID = strings.Repeat("x", f.n)
		f.recs = append(f.recs, rec)
		out(rec)
	case "PUT", "DELETE":
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		for i, x := range f.recs {
			if x.ID == id {
				if r.Method == "DELETE" {
					f.recs = append(f.recs[:i], f.recs[i+1:]...)
				} else {
					var rec cfRecord
					json.NewDecoder(r.Body).Decode(&rec)
					rec.ID = id
					f.recs[i] = rec
				}
				break
			}
		}
		out(map[string]string{"id": id})
	}
}

func TestAutoDomain(t *testing.T) {
	cf := &fakeCF{recs: []cfRecord{{ID: "fremd", Type: "A", Name: "survival.panel.example.com", Content: "1.2.3.4"}}}
	ts := httptest.NewServer(cf)
	defer ts.Close()
	cfBase = ts.URL
	publicIP.ip, publicIP.at = "203.0.113.7", time.Now()

	dir := t.TempDir()
	st, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	st.S.Settings.CFToken = "tok"
	st.S.Settings.AutoDomain = &AutoDomain{ZoneID: "z1", ZoneName: "example.com", Sub: "panel"}
	st.S.Eggs = []*Egg{{ID: "paper", Name: "Paper"}, {ID: "valheim", Name: "Valheim"}}
	st.S.Servers = []*Server{
		{ID: "s1", Name: "Survival", EggID: "paper", Ports: []int{25570}},
		{ID: "s2", Name: "Valheim Welt", EggID: "valheim", Ports: []int{2456}},
		{ID: "s3", Name: "Hat schon", EggID: "paper", Ports: []int{25571}, Domain: &Domain{ZoneID: "z1", ZoneName: "example.com", Prefix: "eigen"}},
	}
	a := &App{st: st, data: dir}
	a.m = newManager(st, nil, dir)
	a.applyAutoDomainsAll()

	got := map[string]string{}
	for _, s := range st.S.Servers {
		if s.Domain != nil {
			got[s.ID] = s.Domain.FQDN() + "|" + s.Domain.Status
		}
	}
	// survival.panel.example.com gehört schon jemand anderem -> survival-2
	if got["s1"] != "survival-2.panel.example.com|ok" || got["s2"] != "valheim-welt.panel.example.com|ok" || got["s3"] != "eigen.example.com|" {
		t.Fatalf("Domains: %v", got)
	}
	var names []string
	for _, r := range cf.recs {
		names = append(names, r.Type+" "+r.Name)
		if r.ID == "fremd" && r.Content != "1.2.3.4" {
			t.Error("fremder Eintrag wurde geändert")
		}
		if r.Type == "A" && r.ID != "fremd" && r.Content != "203.0.113.7" {
			t.Errorf("A-Eintrag zeigt auf %s", r.Content)
		}
		if r.Proxied != nil && *r.Proxied {
			t.Error("Eintrag ist proxied")
		}
	}
	want := "A survival.panel.example.com,A survival-2.panel.example.com,SRV _minecraft._tcp.survival-2.panel.example.com,A valheim-welt.panel.example.com"
	if strings.Join(names, ",") != want {
		t.Fatalf("Einträge:\n%s\nerwartet:\n%s", strings.Join(names, ","), want)
	}
}

func TestAutoDomainMove(t *testing.T) {
	cf := &fakeCF{}
	ts := httptest.NewServer(cf)
	defer ts.Close()
	cfBase = ts.URL
	publicIP.ip, publicIP.at = "203.0.113.7", time.Now()
	dir := t.TempDir()
	st, _ := openStore(dir)
	old := &AutoDomain{ZoneID: "z1", ZoneName: "example.com", Sub: "panel"}
	st.S.Settings.CFToken = "tok"
	st.S.Settings.AutoDomain = old
	st.S.Eggs = []*Egg{{ID: "paper", Name: "Paper"}}
	st.S.Servers = []*Server{
		// aus 0.11.0, noch ohne Auto-Markierung
		{ID: "s1", Name: "Proxy", EggID: "paper", Ports: []int{25577}, Domain: &Domain{ZoneID: "z1", ZoneName: "example.com", Prefix: "proxy.panel", SRV: true}},
		// von Hand vergeben, muss bleiben
		{ID: "s2", Name: "Lobby", EggID: "paper", Ports: []int{25565}, Domain: &Domain{ZoneID: "z1", ZoneName: "example.com", Prefix: "mc", SRV: true}},
	}
	a := &App{st: st, data: dir}
	a.m = newManager(st, nil, dir)
	syncAll := func() {
		for _, s := range st.S.Servers {
			a.syncDomain(s)
		}
	}
	syncAll()
	now := &AutoDomain{ZoneID: "z1", ZoneName: "example.com"}
	st.S.Settings.AutoDomain = now
	a.moveAutoDomains(old, now)
	a.applyAutoDomainsAll()
	if d := st.S.Servers[0].Domain; d == nil || d.FQDN() != "proxy.example.com" || !d.Auto {
		t.Fatalf("Proxy: %+v", d)
	}
	if d := st.S.Servers[1].Domain; d == nil || d.FQDN() != "mc.example.com" {
		t.Fatalf("Lobby: %+v", d)
	}
	var names []string
	for _, r := range cf.recs {
		names = append(names, r.Type+" "+r.Name)
	}
	want := "A mc.example.com,SRV _minecraft._tcp.mc.example.com,A proxy.example.com,SRV _minecraft._tcp.proxy.example.com"
	if strings.Join(names, ",") != want {
		t.Fatalf("Einträge:\n%s\nerwartet:\n%s", strings.Join(names, ","), want)
	}
}
