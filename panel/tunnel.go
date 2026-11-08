package main

// Panel öffentlich machen über einen Cloudflare Tunnel:
// keine Portfreigabe nötig, HTTPS kommt von Cloudflare.
// Serverwerk legt Tunnel + DNS-Eintrag an und startet cloudflared als Container.

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const tunnelContainer = "serverwerk-tunnel"
const tunnelImage = "cloudflare/cloudflared:latest"

type PanelPublic struct {
	ZoneID    string `json:"zoneId"`
	ZoneName  string `json:"zoneName"`
	Prefix    string `json:"prefix"`
	AccountID string `json:"accountId"`
	TunnelID  string `json:"tunnelId"`
	Token     string `json:"token"`
}

func (p *PanelPublic) FQDN() string {
	if p == nil {
		return ""
	}
	return (&Domain{ZoneName: p.ZoneName, Prefix: p.Prefix}).FQDN()
}

func (a *App) startTunnelContainer(token string) error {
	ctx := context.Background()
	if !a.dk.ImageExists(ctx, tunnelImage) {
		if err := a.dk.Pull(ctx, tunnelImage, nil); err != nil {
			return fmt.Errorf("cloudflared laden: %v", err)
		}
	}
	a.dk.Remove(ctx, tunnelContainer)
	id, err := a.dk.Create(ctx, tunnelContainer, map[string]any{
		"Image":  tunnelImage,
		"Cmd":    []string{"tunnel", "--no-autoupdate", "run", "--token", token},
		"Labels": map[string]string{"serverwerk.role": "tunnel"},
		"HostConfig": map[string]any{
			"NetworkMode":   "host",
			"RestartPolicy": map[string]any{"Name": "unless-stopped"},
			"LogConfig":     map[string]any{"Type": "local", "Config": map[string]string{"max-size": "2m", "max-file": "2", "compress": "false"}},
		},
	})
	if err != nil {
		return err
	}
	return a.dk.Start(ctx, id)
}

func (a *App) tunnelRunning() (bool, string) {
	ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
	defer c()
	ci, err := a.dk.Inspect(ctx, tunnelContainer)
	if err != nil {
		return false, "Tunnel-Container fehlt"
	}
	started, _ := time.Parse(time.RFC3339Nano, ci.State.StartedAt)
	if ci.State.Restarting || (ci.RestartCount > 0 && time.Since(started) < 30*time.Second) {
		return false, "Der Tunnel startet immer wieder neu. Meist ist der Tunnel bei Cloudflare gelöscht oder der Token ungültig: Panel einmal wieder privat und dann öffentlich machen."
	}
	if !ci.State.Running {
		return false, fmt.Sprintf("Tunnel-Container läuft nicht (Exit %d)", ci.State.ExitCode)
	}
	return true, ""
}

// ensureTunnel startet den Container nach einem Neustart neu, falls er fehlt.
func (a *App) ensureTunnel() {
	var pp *PanelPublic
	a.st.Read(func(st *State) {
		if st.Settings.Panel != nil {
			c := *st.Settings.Panel
			pp = &c
		}
	})
	if pp == nil || pp.Token == "" {
		return
	}
	if ok, _ := a.tunnelRunning(); ok {
		return
	}
	if err := a.startTunnelContainer(pp.Token); err != nil {
		log.Printf("Tunnel: %v", err)
	}
}

func (a *App) panelPublish(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ZoneID string `json:"zoneId"`
		Prefix string `json:"prefix"`
		Remove bool   `json:"remove"`
	}
	if readJSON(r, &in) != nil {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	var set Settings
	a.st.Read(func(st *State) { set = st.Settings })
	if set.CFToken == "" {
		jsonErr(w, 400, "Erst Cloudflare verbinden")
		return
	}
	tok := set.CFToken
	if in.Remove {
		a.unpublish(tok, set.Panel)
		a.st.Update(func(st *State) error { st.Settings.Panel = nil; return nil })
		ok(w)
		return
	}
	var zone *CFZone
	for _, z := range set.CFZones {
		if z.ID == in.ZoneID {
			c := z
			zone = &c
		}
	}
	if zone == nil {
		jsonErr(w, 400, "Domain wählen")
		return
	}
	p := strings.Trim(strings.ToLower(strings.TrimSpace(in.Prefix)), ".")
	if p == "" {
		p = "@"
	}
	if !prefixRe.MatchString(p) {
		jsonErr(w, 400, "Ungültiger Name: nur a-z, 0-9 und Bindestrich")
		return
	}
	pp := &PanelPublic{ZoneID: zone.ID, ZoneName: zone.Name, Prefix: p}
	fq := pp.FQDN()
	// Kollision mit einem Server?
	clash := ""
	a.st.Read(func(st *State) {
		for _, x := range st.Servers {
			if x.Domain != nil && x.Domain.FQDN() == fq {
				clash = x.Name
			}
		}
	})
	if clash != "" {
		jsonErr(w, 409, fq+" benutzt schon der Server "+clash)
		return
	}
	// Konto der Zone herausfinden.
	var zd struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	if err := cfCall(tok, "GET", "/zones/"+zone.ID, nil, &zd); err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	pp.AccountID = zd.Account.ID
	// Alten Tunnel wiederverwenden, wenn er im selben Konto liegt.
	old := set.Panel
	if old != nil && old.AccountID == pp.AccountID && old.TunnelID != "" {
		pp.TunnelID, pp.Token = old.TunnelID, old.Token
	}
	if pp.TunnelID == "" {
		var t struct {
			ID string `json:"id"`
		}
		name := "serverwerk-" + newID(4)
		err := cfCall(tok, "POST", "/accounts/"+pp.AccountID+"/cfd_tunnel", map[string]any{"name": name, "config_src": "cloudflare"}, &t)
		if err != nil {
			if strings.Contains(err.Error(), "Token ungültig") {
				err = fmt.Errorf("Dem Cloudflare-Token fehlt ein Recht. Bitte im Token zusätzlich „Konto > Cloudflare Tunnel > Bearbeiten“ erlauben und ihn unter Cloudflare neu eintragen.")
			}
			jsonErr(w, 502, err.Error())
			return
		}
		pp.TunnelID = t.ID
		var token string
		if err := cfCall(tok, "GET", "/accounts/"+pp.AccountID+"/cfd_tunnel/"+t.ID+"/token", nil, &token); err != nil {
			jsonErr(w, 502, err.Error())
			return
		}
		pp.Token = token
	}
	port := a.port
	cfgBody := map[string]any{"config": map[string]any{"ingress": []map[string]any{
		{"hostname": fq, "service": fmt.Sprintf("http://localhost:%s", port), "originRequest": map[string]any{"disableChunkedEncoding": false}},
		{"service": "http_status:404"},
	}}}
	if err := cfCall(tok, "PUT", "/accounts/"+pp.AccountID+"/cfd_tunnel/"+pp.TunnelID+"/configurations", cfgBody, nil); err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	// Alter Name weg, falls umbenannt.
	if old != nil && old.FQDN() != fq && old.ZoneID != "" {
		a.deleteOrQueue(tok, DNSJob{ZoneID: old.ZoneID, Comment: "serverwerk:panel", Names: []string{old.FQDN()}})
	}
	t := true
	rec := cfRecord{Type: "CNAME", Name: fq, Content: pp.TunnelID + ".cfargotunnel.com", Proxied: &t, TTL: 1, Comment: "serverwerk:panel"}
	if err := cfUpsert(tok, pp.ZoneID, rec); err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	if err := a.startTunnelContainer(pp.Token); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	a.st.Update(func(st *State) error { st.Settings.Panel = pp; return nil })
	jsonOut(w, map[string]string{"url": "https://" + fq})
}

func (a *App) unpublish(tok string, pp *PanelPublic) {
	a.dk.Remove(context.Background(), tunnelContainer)
	if pp == nil {
		return
	}
	if pp.ZoneID != "" {
		a.deleteOrQueue(tok, DNSJob{ZoneID: pp.ZoneID, Comment: "serverwerk:panel", Names: []string{pp.FQDN()}})
	}
	if pp.AccountID != "" && pp.TunnelID != "" {
		base := "/accounts/" + pp.AccountID + "/cfd_tunnel/" + pp.TunnelID
		cfCall(tok, "DELETE", base+"/connections", nil, nil)
		if err := cfCall(tok, "DELETE", base, nil, nil); err != nil {
			log.Printf("Tunnel löschen: %v", err)
		}
	}
}
