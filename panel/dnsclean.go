package main

// Aufräumen von DNS-Einträgen, die Serverwerk angelegt hat und die nicht mehr gebraucht werden.
//
// Gelöscht wird nur, was sicher zu diesem Panel gehört:
//   - Einträge aus der Warteschlange (Löschen ist früher fehlgeschlagen),
//   - Einträge eigener Server, deren Name nicht mehr zur Domain des Servers passt.
// Einträge mit unbekannter Server-ID bleiben stehen: Sie könnten einer zweiten
// Serverwerk-Installation im selben Cloudflare-Konto gehören.

import (
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DNSJob struct {
	ZoneID   string    `json:"zoneId"`
	Comment  string    `json:"comment"`
	ServerID string    `json:"serverId,omitempty"`
	Names    []string  `json:"names"`
	Since    time.Time `json:"since"`
}

func (j DNSJob) same(x DNSJob) bool {
	return x.ZoneID == j.ZoneID && x.Comment == j.Comment && strings.Join(x.Names, ",") == strings.Join(j.Names, ",")
}

func (a *App) queueDNS(j DNSJob) {
	if j.Since.IsZero() {
		j.Since = time.Now()
	}
	a.st.Update(func(st *State) error {
		for _, x := range st.DNSCleanup {
			if x.same(j) {
				return nil
			}
		}
		st.DNSCleanup = append(st.DNSCleanup, j)
		return nil
	})
}

func (a *App) deleteOrQueue(token string, j DNSJob) {
	if token == "" || cfDeleteOwn(token, j.ZoneID, j.Comment, j.Names...) != nil {
		a.queueDNS(j)
	}
}

// liveNames sind Namen, die inzwischen wieder benutzt werden und nicht gelöscht werden dürfen.
func liveNames(st *State, j DNSJob) map[string]bool {
	live := map[string]bool{}
	if j.Comment == "serverwerk:panel" {
		if p := st.Settings.Panel; p != nil && p.ZoneID == j.ZoneID {
			live[p.FQDN()] = true
		}
		return live
	}
	if s := findServer(st, j.ServerID); s != nil && s.Domain != nil && s.Domain.ZoneID == j.ZoneID {
		for _, n := range domainNames(s.Domain) {
			live[n] = true
		}
	}
	return live
}

// runDNSQueue versucht alle wartenden Löschungen erneut.
func (a *App) runDNSQueue() {
	var token string
	var jobs []DNSJob
	a.st.Read(func(st *State) { token, jobs = st.Settings.CFToken, append([]DNSJob(nil), st.DNSCleanup...) })
	if token == "" || len(jobs) == 0 {
		return
	}
	for _, j := range jobs {
		var names []string
		a.st.Read(func(st *State) {
			live := liveNames(st, j)
			for _, n := range j.Names {
				if !live[n] {
					names = append(names, n)
				}
			}
		})
		err := cfDeleteOwn(token, j.ZoneID, j.Comment, names...)
		// Nach 30 Tagen aufgeben (z. B. Domain gibt es im Konto nicht mehr).
		if err != nil && time.Since(j.Since) < 30*24*time.Hour {
			continue
		}
		if err != nil {
			log.Printf("DNS löschen aufgegeben: %v (%v)", j.Names, err)
		}
		a.st.Update(func(st *State) error {
			for i, x := range st.DNSCleanup {
				if x.same(j) {
					st.DNSCleanup = append(st.DNSCleanup[:i], st.DNSCleanup[i+1:]...)
					break
				}
			}
			return nil
		})
	}
}

// cfAllRecords liest alle Einträge einer Zone seitenweise.
func cfAllRecords(token, zone string) ([]cfRecord, error) {
	var all []cfRecord
	for page := 1; page <= 200; page++ {
		var rs []cfRecord
		if err := cfCall(token, "GET", "/zones/"+zone+"/dns_records?per_page=100&page="+strconv.Itoa(page), nil, &rs); err != nil {
			return nil, err
		}
		all = append(all, rs...)
		if len(rs) < 100 {
			break
		}
	}
	return all, nil
}

// inUse sagt, ob ein Name gerade zur Domain dieses Servers gehört (frisch aus dem Zustand).
func (a *App) inUse(serverID, zoneID, name string) bool {
	use := false
	a.st.Read(func(st *State) {
		if s := findServer(st, serverID); s != nil && s.Domain != nil && s.Domain.ZoneID == zoneID {
			for _, n := range domainNames(s.Domain) {
				if n == name {
					use = true
				}
			}
		}
	})
	return use
}

// sweepDNS löscht Einträge eigener Server, deren Name nicht mehr stimmt.
func (a *App) sweepDNS() {
	var set Settings
	mine := map[string]bool{}
	a.st.Read(func(st *State) {
		set = st.Settings
		for _, s := range st.Servers {
			mine[s.ID] = true
		}
	})
	if set.CFToken == "" {
		return
	}
	for _, z := range set.CFZones {
		recs, err := cfAllRecords(set.CFToken, z.ID)
		if err != nil {
			log.Printf("DNS aufräumen %s: %v", z.Name, err)
			continue
		}
		for _, r := range recs {
			id, ok := strings.CutPrefix(r.Comment, "serverwerk:")
			if !ok || id == "panel" || !mine[id] {
				continue // fremd, Panel oder unbekannte ID (vielleicht eine andere Installation)
			}
			// Gerade erst angelegt oder geändert: in Ruhe lassen (Vergabe läuft evtl. noch).
			if t, err := time.Parse(time.RFC3339, r.Modified); err == nil && time.Since(t) < 15*time.Minute {
				continue
			}
			if a.inUse(id, z.ID, r.Name) {
				continue
			}
			if err := cfCall(set.CFToken, "DELETE", "/zones/"+z.ID+"/dns_records/"+url.PathEscape(r.ID), nil, nil); err != nil {
				log.Printf("DNS aufräumen %s: %v", r.Name, err)
			} else {
				log.Printf("Alten DNS-Eintrag gelöscht: %s %s", r.Type, r.Name)
			}
		}
	}
}

func (a *App) dnsCleanupLoop() {
	time.Sleep(2 * time.Minute)
	for {
		a.runDNSQueue()
		a.sweepDNS()
		time.Sleep(time.Hour)
	}
}
