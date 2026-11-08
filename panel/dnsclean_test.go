package main

import (
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

func cfNames(cf *fakeCF) string {
	var n []string
	for _, r := range cf.recs {
		n = append(n, r.Type+" "+r.Name)
	}
	sort.Strings(n)
	return strings.Join(n, ",")
}

func TestDNSQueueAndSweep(t *testing.T) {
	cf := &fakeCF{}
	ts := httptest.NewServer(cf)
	defer ts.Close()
	cfBase = ts.URL
	publicIP.ip, publicIP.at = "203.0.113.7", time.Now()
	dir := t.TempDir()
	st, _ := openStore(dir)
	st.S.Settings.CFToken = "tok"
	st.S.Settings.CFZones = []CFZone{{ID: "z1", Name: "example.com"}}
	st.S.Settings.Panel = &PanelPublic{ZoneID: "z1", ZoneName: "example.com", Prefix: "panel"}
	st.S.Eggs = []*Egg{{ID: "paper", Name: "Paper"}}
	weg := &Server{ID: "weg", Name: "Weg", EggID: "paper", Ports: []int{25566}, Domain: &Domain{ZoneID: "z1", ZoneName: "example.com", Prefix: "weg", SRV: true}}
	bleibt := &Server{ID: "bleibt", Name: "Bleibt", EggID: "paper", Ports: []int{25565}, Domain: &Domain{ZoneID: "z1", ZoneName: "example.com", Prefix: "bleibt", SRV: true}}
	st.S.Servers = []*Server{weg, bleibt}
	a := &App{st: st, data: dir}
	a.m = newManager(st, nil, dir)
	a.syncDomain(weg)
	a.syncDomain(bleibt)
	recent := time.Now().Format(time.RFC3339)
	cf.recs = append(cf.recs,
		cfRecord{ID: "alt", Type: "A", Name: "alt.example.com", Content: "1.1.1.1", Comment: "serverwerk:bleibt"},                      // alter Name eines eigenen Servers -> weg
		cfRecord{ID: "neu", Type: "A", Name: "frisch.example.com", Content: "1.1.1.1", Comment: "serverwerk:bleibt", Modified: recent}, // gerade erst geändert -> bleibt
		cfRecord{ID: "andere", Type: "A", Name: "fremd.example.com", Content: "1.1.1.1", Comment: "serverwerk:andere-installation"},    // unbekannte ID -> bleibt
		cfRecord{ID: "panel", Type: "CNAME", Name: "panel.example.com", Content: "x.cfargotunnel.com", Comment: "serverwerk:panel"},    // Panel -> bleibt
		cfRecord{ID: "www", Type: "A", Name: "www.example.com", Content: "1.1.1.1"},                                                    // fremd -> bleibt
	)

	// Server löschen, während Cloudflare nicht erreichbar ist.
	cf.down = true
	if err := a.removeDomain(weg); err == nil {
		t.Fatal("Fehler erwartet")
	}
	st.S.Servers = []*Server{bleibt}
	if len(st.S.DNSCleanup) != 1 {
		t.Fatalf("Warteschlange: %+v", st.S.DNSCleanup)
	}
	a.runDNSQueue() // noch down: bleibt in der Warteschlange
	if len(st.S.DNSCleanup) != 1 {
		t.Fatal("Auftrag verloren, obwohl Cloudflare down war")
	}
	cf.down = false
	a.runDNSQueue()
	a.sweepDNS()
	if len(st.S.DNSCleanup) != 0 {
		t.Fatalf("Warteschlange nicht leer: %+v", st.S.DNSCleanup)
	}
	want := "A bleibt.example.com,A fremd.example.com,A frisch.example.com,A www.example.com,CNAME panel.example.com,SRV _minecraft._tcp.bleibt.example.com"
	if got := cfNames(cf); got != want {
		t.Fatalf("Einträge:\n%s\nerwartet:\n%s", got, want)
	}

	// Wartender Auftrag darf einen inzwischen wieder benutzten Namen nicht löschen.
	a.queueDNS(DNSJob{ZoneID: "z1", Comment: "serverwerk:bleibt", ServerID: "bleibt", Names: domainNames(bleibt.Domain)})
	a.runDNSQueue()
	if got := cfNames(cf); got != want {
		t.Fatalf("lebender Name gelöscht:\n%s", got)
	}
}
