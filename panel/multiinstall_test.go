package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Zwei Serverwerk-Installationen im selben Cloudflare-Konto dürfen sich nicht in die Quere kommen.
func TestTwoInstallsSameCloudflare(t *testing.T) {
	cf := &fakeCF{}
	ts := httptest.NewServer(cf)
	defer ts.Close()
	cfBase = ts.URL
	publicIP.ip, publicIP.at = "203.0.113.7", time.Now()

	mk := func(serverID string) *App {
		dir := t.TempDir()
		st, _ := openStore(dir)
		st.S.Settings.CFToken = "tok"
		st.S.Settings.CFZones = []CFZone{{ID: "z1", Name: "example.com"}}
		st.S.Settings.AutoDomain = &AutoDomain{ZoneID: "z1", ZoneName: "example.com", Sub: "panel"}
		st.S.Eggs = []*Egg{{ID: "paper", Name: "Paper"}}
		st.S.Servers = []*Server{{ID: serverID, Name: "Survival", EggID: "paper", Ports: []int{25565}}}
		a := &App{st: st, data: dir}
		a.m = newManager(st, nil, dir)
		return a
	}
	a, b := mk("server-a"), mk("server-b")
	a.applyAutoDomainsAll()
	b.applyAutoDomainsAll()
	if got := a.st.S.Servers[0].Domain.FQDN(); got != "survival.panel.example.com" {
		t.Fatalf("A: %s", got)
	}
	if got := b.st.S.Servers[0].Domain.FQDN(); got != "survival-2.panel.example.com" {
		t.Fatalf("B soll ausweichen, hat %s", got)
	}

	// Aufräumen von B lässt die Einträge von A stehen.
	b.sweepDNS()
	b.runDNSQueue()
	if !strings.Contains(cfNames(cf), "A survival.panel.example.com") {
		t.Fatalf("B hat A's Eintrag gelöscht: %s", cfNames(cf))
	}
	// B löscht seinen Server: nur B's Einträge verschwinden.
	b.removeDomain(b.st.S.Servers[0])
	if got := cfNames(cf); got != "A survival.panel.example.com,SRV _minecraft._tcp.survival.panel.example.com" {
		t.Fatalf("nach Löschen von B: %s", got)
	}

	// Panel-Adresse: B darf A's Panel-Eintrag nicht übernehmen.
	rec := func(app *App) cfRecord {
		return cfRecord{Type: "CNAME", Name: "panel.example.com", Content: "x.cfargotunnel.com", TTL: 1, Comment: app.panelComment()}
	}
	if a.panelComment() == b.panelComment() {
		t.Fatal("beide Installationen haben dieselbe Kennung")
	}
	if err := cfUpsert("tok", "z1", rec(a)); err != nil {
		t.Fatal(err)
	}
	if err := cfUpsert("tok", "z1", rec(b)); err == nil || !strings.Contains(err.Error(), "schon") {
		t.Fatalf("B hat A's Panel-Eintrag übernommen (err=%v)", err)
	}
	// Ein alter Panel-Eintrag ohne Kennung (vor 0.12.1) gilt weiter als eigener.
	if !ownComment("serverwerk:panel", a.panelComment()) || ownComment("serverwerk:panel:fremd", a.panelComment()) {
		t.Fatal("ownComment falsch")
	}
}
