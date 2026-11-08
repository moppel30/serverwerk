package main

// Umweg zu VMs, die per macvtap am Netz hängen (libvirt "direct").
// Der Host kann solche VMs über seine normale Netzwerkkarte nicht erreichen,
// eine zusätzliche macvlan-Karte am selben Anschluss aber schon. Serverwerk legt
// sie bei Bedarf an und leitet nur die Adressen dieser VMs darüber.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const shimDev = "sw-shim"

var shimMu sync.Mutex

func ipCmd(args ...string) (string, error) {
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("ip %s: %v %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// routeInfo liefert Ausgangskarte und eigene Adresse für ein Ziel.
func routeInfo(ip net.IP) (dev, src string, direct bool, err error) {
	out, err := ipCmd("-o", "route", "get", ip.String())
	if err != nil {
		return "", "", false, err
	}
	f := strings.Fields(out)
	direct = true
	for i := 0; i+1 < len(f); i++ {
		switch f[i] {
		case "dev":
			dev = f[i+1]
		case "src":
			src = f[i+1]
		case "via":
			direct = false
		}
	}
	return dev, src, direct, nil
}

type macvtapInfo struct {
	Name, Parent, Mode string
}

func macvtapsOn(parent string) []macvtapInfo {
	out, err := ipCmd("-d", "-o", "link", "show", "type", "macvtap")
	if err != nil {
		return nil
	}
	var list []macvtapInfo
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		name := strings.TrimSuffix(f[1], ":")
		par := ""
		if i := strings.Index(name, "@"); i > 0 {
			name, par = name[:i], name[i+1:]
		}
		mode := ""
		for i := 0; i+2 < len(f); i++ {
			if f[i] == "macvtap" && f[i+1] == "mode" {
				mode = f[i+2]
			}
		}
		if par == parent {
			list = append(list, macvtapInfo{name, par, mode})
		}
	}
	return list
}

// ensureShim richtet den Umweg zu ip ein, wenn eine macvtap-VM die Ursache sein kann.
func (a *App) ensureShim(ip net.IP) (bool, error) {
	shimMu.Lock()
	defer shimMu.Unlock()
	dev, src, direct, err := routeInfo(ip)
	if err != nil || !direct || src == "" {
		return false, nil
	}
	if dev == shimDev {
		return false, nil // läuft schon darüber
	}
	taps := macvtapsOn(dev)
	if len(taps) == 0 {
		return false, nil
	}
	bridge := false
	for _, t := range taps {
		if t.Mode == "bridge" {
			bridge = true
		}
	}
	if !bridge {
		return false, fmt.Errorf("die VM hängt per macvtap im Modus %q an %s; ein Umweg geht nur im Modus „bridge“ (in virt-manager: Netzwerkkarte › Quellmodus Bridge)", taps[0].Mode, dev)
	}
	if err := setupShim(dev, ip.String(), src); err != nil {
		return false, err
	}
	a.st.Update(func(st *State) error {
		for _, r := range st.Settings.Shims {
			if r.IP == ip.String() {
				return nil
			}
		}
		st.Settings.Shims = append(st.Settings.Shims, ShimRoute{IP: ip.String(), Parent: dev})
		return nil
	})
	log.Printf("Umweg %s über %s zu %s eingerichtet (macvtap-VM)", shimDev, dev, ip)
	return true, nil
}

func setupShim(parent, ip, src string) error {
	if _, err := ipCmd("link", "show", shimDev); err != nil {
		if _, err := ipCmd("link", "add", shimDev, "link", parent, "type", "macvlan", "mode", "bridge"); err != nil {
			return err
		}
	}
	if _, err := ipCmd("link", "set", shimDev, "up"); err != nil {
		return err
	}
	_, err := ipCmd("route", "replace", ip+"/32", "dev", shimDev, "src", src)
	return err
}

// restoreShims stellt die Umwege nach einem Neustart wieder her.
func (a *App) restoreShims() {
	var list []ShimRoute
	a.st.Read(func(st *State) { list = append(list, st.Settings.Shims...) })
	for _, r := range list {
		_, src, _, err := routeInfo(net.ParseIP(r.IP))
		if err != nil || src == "" {
			continue
		}
		if err := setupShim(r.Parent, r.IP, src); err != nil {
			log.Printf("Umweg zu %s: %v", r.IP, err)
		}
	}
}

func unreachable(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	return errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH) ||
		strings.Contains(err.Error(), "no route to host") || (errors.As(err, &ne) && ne.Timeout())
}

// shimDial verbindet wie gewohnt und baut bei "no route to host" zu einer
// lokalen macvtap-VM automatisch den Umweg auf, dann ein zweiter Versuch.
func (a *App) shimDial(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 6 * time.Second}
	c, err := d.DialContext(ctx, network, addr)
	if err == nil || !unreachable(err) {
		return c, err
	}
	host, _, _ := net.SplitHostPort(addr)
	ips, _ := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	for _, ip := range ips {
		if !ip.IsPrivate() {
			continue
		}
		ok, serr := a.ensureShim(ip)
		if serr != nil {
			return nil, fmt.Errorf("%v (%v)", err, serr)
		}
		if ok {
			return d.DialContext(ctx, network, addr)
		}
	}
	return nil, err
}
