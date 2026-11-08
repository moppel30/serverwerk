package main

// Automatische Server-Adressen: Jeder Server ohne eigene Domain bekommt
// name.<basis>, z. B. survival.panel.example.com. Die Einträge sind „nur DNS“
// und zeigen auf die öffentliche IP (bzw. den eigenen Hostnamen), also auch dann
// erreichbar, wenn das Panel selbst über den Cloudflare Tunnel läuft.

import (
	"fmt"
	"log"
	"strings"
)

type AutoDomain struct {
	ZoneID   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	Sub      string `json:"sub"` // z. B. "panel", leer = direkt unter der Domain
}

// slugName macht aus einem Servernamen einen DNS-Namen: "Survival 2 (Test)" -> "survival-2-test".
func slugName(name string) string {
	r := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "Ä", "ae", "Ö", "oe", "Ü", "ue", "ß", "ss")
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(r.Replace(name)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	if s == "" {
		s = "server"
	}
	return s
}

// applyAutoDomain gibt einem Server ohne Domain automatisch eine Adresse.
func (a *App) applyAutoDomain(id string) {
	var ad *AutoDomain
	var s *Server
	var token string
	taken := map[string]bool{}
	a.st.Read(func(st *State) {
		if st.Settings.AutoDomain != nil {
			c := *st.Settings.AutoDomain
			ad = &c
		}
		token = st.Settings.CFToken
		for _, x := range st.Servers {
			if x.ID == id && x.Domain == nil {
				s = cloneServer(x)
			}
			if x.Domain != nil {
				taken[x.Domain.FQDN()] = true
			}
		}
		if st.Settings.Panel != nil {
			taken[st.Settings.Panel.FQDN()] = true
		}
	})
	if ad == nil || s == nil || token == "" {
		return
	}
	base := slugName(s.Name)
	var lastErr error
	for i, tries := 1, 0; i < 100 && tries < 5; i++ {
		p := base
		if i > 1 {
			p = fmt.Sprintf("%s-%d", base, i)
		}
		if ad.Sub != "" {
			p += "." + ad.Sub
		}
		nd := &Domain{ZoneID: ad.ZoneID, ZoneName: ad.ZoneName, Prefix: p, SRV: a.isMinecraftJava(s)}
		if taken[nd.FQDN()] {
			continue
		}
		tries++
		s.Domain = nd
		err := a.syncDomain(s)
		if err != nil && strings.Contains(err.Error(), "gibt es in Cloudflare schon") {
			lastErr = err
			continue // Name ist bei Cloudflare schon vergeben, nächsten probieren
		}
		nd.Status = "ok"
		if err != nil {
			nd.Status = err.Error()
			log.Printf("Automatische Adresse %s: %v", nd.FQDN(), err)
		}
		stale := false
		a.st.Update(func(st *State) error {
			if x := findServer(st, id); x != nil && x.Domain == nil {
				x.Domain = nd
			} else {
				stale = true
			}
			return nil
		})
		if stale && err == nil { // Server weg oder inzwischen anders vergeben
			a.removeDomain(s)
		}
		return
	}
	if lastErr != nil {
		log.Printf("Automatische Adresse für %s: %v", s.Name, lastErr)
	}
}

func (a *App) applyAutoDomainsAll() {
	var ids []string
	a.st.Read(func(st *State) {
		for _, x := range st.Servers {
			if x.Domain == nil {
				ids = append(ids, x.ID)
			}
		}
	})
	for _, id := range ids {
		a.applyAutoDomain(id)
	}
}
