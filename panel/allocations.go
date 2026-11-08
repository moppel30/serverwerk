package main

// Allocations: welche Ports im Router freigegeben sind, gruppiert in Bereiche.
// Jeder Bereich kann einen Alias haben (z. B. die DynDNS-Adresse des Routers),
// der dann als Adresse der Server angezeigt wird.

import (
	"fmt"
	"net/http"
	"strings"
)

type AllocGroup struct {
	Ports string `json:"ports"`
	Alias string `json:"alias"`
}

// allocGroups liefert die Bereiche; alte Einstellung "Ports" wird übernommen.
func (s *Settings) allocGroups() []AllocGroup {
	if len(s.Allocations) > 0 {
		return s.Allocations
	}
	if strings.TrimSpace(s.Ports) != "" {
		return []AllocGroup{{Ports: s.Ports}}
	}
	return nil
}

func (a *App) portAlias(port int) string {
	var groups []AllocGroup
	a.st.Read(func(st *State) { groups = st.Settings.allocGroups() })
	for _, g := range groups {
		l, _ := parsePortList(g.Ports)
		for _, p := range l {
			if p == port {
				return g.Alias
			}
		}
	}
	return ""
}

func (a *App) allocationsGet(w http.ResponseWriter, r *http.Request) {
	var groups []AllocGroup
	owner := map[int][2]string{}
	a.st.Read(func(st *State) {
		groups = st.Settings.allocGroups()
		for _, s := range st.Servers {
			for _, p := range s.Ports {
				owner[p] = [2]string{s.ID, s.Name}
			}
		}
	})
	type portRow struct {
		Port   int    `json:"port"`
		Alias  string `json:"alias"`
		Server string `json:"server,omitempty"`
		Name   string `json:"name,omitempty"`
		Busy   bool   `json:"busy"` // von einem anderen Programm belegt
	}
	rows := []portRow{}
	seen := map[int]bool{}
	for _, g := range groups {
		l, _ := parsePortList(g.Ports)
		for _, p := range l {
			if seen[p] {
				continue
			}
			seen[p] = true
			pr := portRow{Port: p, Alias: g.Alias}
			if o, ok := owner[p]; ok {
				pr.Server, pr.Name = o[0], o[1]
			} else if len(rows) < 2000 {
				pr.Busy = portInUse(p)
			}
			rows = append(rows, pr)
		}
	}
	// Ports von Servern, die nicht (mehr) in der Liste stehen.
	var outside []portRow
	for p, o := range owner {
		if !seen[p] {
			outside = append(outside, portRow{Port: p, Server: o[0], Name: o[1]})
		}
	}
	if groups == nil {
		groups = []AllocGroup{}
	}
	jsonOut(w, map[string]any{"groups": groups, "ports": rows, "outside": outside})
}

func (a *App) allocationsPut(w http.ResponseWriter, r *http.Request) {
	var in []AllocGroup
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	var keep []AllocGroup
	for _, g := range in {
		g.Ports = strings.TrimSpace(g.Ports)
		g.Alias = strings.TrimSuffix(strings.TrimSpace(strings.ToLower(g.Alias)), ".")
		if g.Ports == "" {
			continue
		}
		if _, err := parsePortList(g.Ports); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		if g.Alias != "" && !prefixRe.MatchString(g.Alias) {
			jsonErr(w, 400, fmt.Sprintf("Alias %q ist keine gültige Adresse (z. B. meinrouter.myfritz.net oder 203.0.113.5)", g.Alias))
			return
		}
		keep = append(keep, g)
	}
	a.st.Update(func(st *State) error {
		st.Settings.Allocations = keep
		st.Settings.Ports = ""
		if keep == nil {
			st.Settings.Allocations = []AllocGroup{}
		}
		return nil
	})
	a.allocationsGet(w, r)
}
