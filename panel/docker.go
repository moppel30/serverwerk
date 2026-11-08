package main

// Kleiner Docker-Client direkt über die Engine-API (Unix-Socket).
// Kein SDK nötig, nur die paar Aufrufe, die ein Panel braucht.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Docker struct {
	hc   *http.Client
	dial func(ctx context.Context) (net.Conn, error)
}

func newDocker() *Docker {
	host := os.Getenv("DOCKER_HOST")
	network, addr := "unix", "/var/run/docker.sock"
	if strings.HasPrefix(host, "unix://") {
		addr = strings.TrimPrefix(host, "unix://")
	} else if strings.HasPrefix(host, "tcp://") {
		network, addr = "tcp", strings.TrimPrefix(host, "tcp://")
	}
	d := &Docker{}
	d.dial = func(ctx context.Context) (net.Conn, error) {
		var nd net.Dialer
		return nd.DialContext(ctx, network, addr)
	}
	d.hc = &http.Client{Transport: &http.Transport{
		DialContext:  func(ctx context.Context, _, _ string) (net.Conn, error) { return d.dial(ctx) },
		MaxIdleConns: 20, IdleConnTimeout: 60 * time.Second,
	}}
	return d
}

type dockerErr struct {
	Status int
	Msg    string
}

func (e *dockerErr) Error() string { return fmt.Sprintf("Docker %d: %s", e.Status, e.Msg) }

func isNotFound(err error) bool {
	de, ok := err.(*dockerErr)
	return ok && de.Status == 404
}

func (d *Docker) req(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	r, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, rd)
	if err != nil {
		return nil, err
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.hc.Do(r)
	if err != nil {
		return nil, fmt.Errorf("Docker nicht erreichbar: %v", err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var m struct {
			Message string `json:"message"`
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		json.Unmarshal(b, &m)
		if m.Message == "" {
			m.Message = strings.TrimSpace(string(b))
		}
		return nil, &dockerErr{resp.StatusCode, m.Message}
	}
	return resp, nil
}

func (d *Docker) call(ctx context.Context, method, path string, body, out any) error {
	resp, err := d.req(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (d *Docker) Info(ctx context.Context) (map[string]any, error) {
	var m map[string]any
	err := d.call(ctx, "GET", "/info", nil, &m)
	return m, err
}

func (d *Docker) Version(ctx context.Context) (map[string]any, error) {
	var m map[string]any
	err := d.call(ctx, "GET", "/version", nil, &m)
	return m, err
}

func (d *Docker) ImageExists(ctx context.Context, ref string) bool {
	return d.call(ctx, "GET", "/images/"+url.PathEscape(ref)+"/json", nil, nil) == nil
}

// Pull lädt ein Image und meldet den Fortschritt zeilenweise.
func (d *Docker) Pull(ctx context.Context, ref string, progress func(string)) error {
	img, tag := ref, "latest"
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		img, tag = ref[:i], ref[i+1:]
	}
	if i := strings.Index(ref, "@"); i > 0 {
		img, tag = ref[:i], ref[i+1:]
	}
	resp, err := d.req(ctx, "POST", "/images/create?fromImage="+url.QueryEscape(img)+"&tag="+url.QueryEscape(tag), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	layers := map[string]bool{} // Schicht -> fertig
	lastSent := time.Time{}
	report := func(force bool) {
		if progress == nil || len(layers) == 0 || (!force && time.Since(lastSent) < 2*time.Second) {
			return
		}
		done := 0
		for _, d := range layers {
			if d {
				done++
			}
		}
		lastSent = time.Now()
		progress(fmt.Sprintf("Image %s: %d von %d Schichten fertig", ref, done, len(layers)))
	}
	for {
		var m struct {
			Status string `json:"status"`
			ID     string `json:"id"`
			Error  string `json:"error"`
		}
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				report(true)
				return nil
			}
			return err
		}
		if m.Error != "" {
			return fmt.Errorf("%s", m.Error)
		}
		if m.ID == "" || strings.HasPrefix(m.Status, "Pulling from") {
			continue
		}
		switch {
		case m.Status == "Pull complete" || m.Status == "Already exists":
			layers[m.ID] = true
		case !layers[m.ID]:
			layers[m.ID] = false
		}
		report(false)
	}
}

type ContainerSummary struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Labels map[string]string `json:"Labels"`
}

func (d *Docker) ListByLabel(ctx context.Context, label string) ([]ContainerSummary, error) {
	f, _ := json.Marshal(map[string][]string{"label": {label}})
	var out []ContainerSummary
	err := d.call(ctx, "GET", "/containers/json?all=1&filters="+url.QueryEscape(string(f)), nil, &out)
	return out, err
}

func (d *Docker) Create(ctx context.Context, name string, cfg map[string]any) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := d.call(ctx, "POST", "/containers/create?name="+url.QueryEscape(name), cfg, &out)
	return out.ID, err
}

func (d *Docker) Start(ctx context.Context, id string) error {
	return d.call(ctx, "POST", "/containers/"+id+"/start", nil, nil)
}

func (d *Docker) Kill(ctx context.Context, id, sig string) error {
	return d.call(ctx, "POST", "/containers/"+id+"/kill?signal="+sig, nil, nil)
}

func (d *Docker) Remove(ctx context.Context, id string) error {
	err := d.call(ctx, "DELETE", "/containers/"+url.PathEscape(id)+"?force=1&v=1", nil, nil)
	if isNotFound(err) {
		return nil
	}
	return err
}

type ContainerInspect struct {
	ID           string `json:"Id"`
	RestartCount int    `json:"RestartCount"`
	State        struct {
		Running    bool   `json:"Running"`
		Restarting bool   `json:"Restarting"`
		ExitCode   int    `json:"ExitCode"`
		OOMKilled  bool   `json:"OOMKilled"`
		StartedAt  string `json:"StartedAt"`
	} `json:"State"`
}

func (d *Docker) Inspect(ctx context.Context, id string) (*ContainerInspect, error) {
	var out ContainerInspect
	err := d.call(ctx, "GET", "/containers/"+url.PathEscape(id)+"/json", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Wait blockiert, bis der Container endet, und liefert den Exit-Code.
// cond "next-exit" wartet auf das nächste Ende (auch wenn der Container noch
// gar nicht gestartet ist), "" = sofort zurück, wenn er gerade nicht läuft.
func (d *Docker) Wait(ctx context.Context, id string, cond ...string) (int, error) {
	var out struct {
		StatusCode int `json:"StatusCode"`
	}
	q := ""
	if len(cond) > 0 && cond[0] != "" {
		q = "?condition=" + cond[0]
	}
	err := d.call(ctx, "POST", "/containers/"+id+"/wait"+q, nil, &out)
	return out.StatusCode, err
}

func (d *Docker) Resize(ctx context.Context, id string, w, h int) error {
	return d.call(ctx, "POST", fmt.Sprintf("/containers/%s/resize?h=%d&w=%d", id, h, w), nil, nil)
}

// Attach hängt sich an stdin/stdout eines Containers (TTY, also ohne Multiplex).
type Attachment struct {
	conn net.Conn
	R    *bufio.Reader
}

func (a *Attachment) Write(p []byte) (int, error) { return a.conn.Write(p) }
func (a *Attachment) Close() error                { return a.conn.Close() }

func (d *Docker) Attach(ctx context.Context, id string) (*Attachment, error) {
	conn, err := d.dial(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(conn, "POST /containers/%s/attach?stream=1&stdin=1&stdout=1&stderr=1 HTTP/1.1\r\nHost: docker\r\nUpgrade: tcp\r\nConnection: Upgrade\r\nContent-Length: 0\r\n\r\n", id)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != 101 && resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		conn.Close()
		return nil, &dockerErr{resp.StatusCode, strings.TrimSpace(string(b))}
	}
	return &Attachment{conn: conn, R: br}, nil
}

// Logs liefert den Ausgabe-Strom (TTY-Container, also roh).
func (d *Docker) Logs(ctx context.Context, id string, follow bool, tail int) (io.ReadCloser, error) {
	q := fmt.Sprintf("/containers/%s/logs?stdout=1&stderr=1&tail=%d", id, tail)
	if tail < 0 {
		q = fmt.Sprintf("/containers/%s/logs?stdout=1&stderr=1&tail=all", id)
	}
	if follow {
		q += "&follow=1"
	}
	resp, err := d.req(ctx, "GET", q, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

type rawStats struct {
	CPU struct {
		Usage struct {
			Total uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		System uint64 `json:"system_cpu_usage"`
		Online int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	Pre struct {
		Usage struct {
			Total uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		System uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	Mem struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
	Net map[string]struct {
		Rx uint64 `json:"rx_bytes"`
		Tx uint64 `json:"tx_bytes"`
	} `json:"networks"`
}

type Stats struct {
	CPU    float64 `json:"cpu"`
	Mem    uint64  `json:"mem"`
	MemMax uint64  `json:"memMax"`
	Rx     uint64  `json:"rx"`
	Tx     uint64  `json:"tx"`
	Disk   uint64  `json:"disk"`
	Uptime int64   `json:"uptime"`
}

// StatsStream ruft fn etwa einmal pro Sekunde mit frischen Werten auf.
func (d *Docker) StatsStream(ctx context.Context, id string, fn func(Stats)) error {
	resp, err := d.req(ctx, "GET", "/containers/"+id+"/stats?stream=1", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	for {
		var r rawStats
		if err := dec.Decode(&r); err != nil {
			return err
		}
		var s Stats
		cpuD := float64(r.CPU.Usage.Total) - float64(r.Pre.Usage.Total)
		sysD := float64(r.CPU.System) - float64(r.Pre.System)
		if sysD > 0 && cpuD > 0 {
			n := r.CPU.Online
			if n == 0 {
				n = 1
			}
			s.CPU = cpuD / sysD * float64(n) * 100
		}
		s.Mem = r.Mem.Usage
		// cgroup v2: inactive_file abziehen, cgroup v1: total_inactive_file / cache.
		if v, ok := r.Mem.Stats["inactive_file"]; ok && v < s.Mem {
			s.Mem -= v
		} else if v, ok := r.Mem.Stats["total_inactive_file"]; ok && v < s.Mem {
			s.Mem -= v
		}
		s.MemMax = r.Mem.Limit
		for _, n := range r.Net {
			s.Rx += n.Rx
			s.Tx += n.Tx
		}
		fn(s)
	}
}

// Gateway der Docker-Bridge, für {{config.docker.interface}} in Eggs.
func (d *Docker) BridgeGateway(ctx context.Context) string {
	var n struct {
		IPAM struct {
			Config []struct {
				Gateway string `json:"Gateway"`
			} `json:"Config"`
		} `json:"IPAM"`
	}
	if d.call(ctx, "GET", "/networks/bridge", nil, &n) == nil {
		for _, c := range n.IPAM.Config {
			if c.Gateway != "" && !strings.Contains(c.Gateway, ":") {
				return c.Gateway
			}
		}
	}
	return "172.17.0.1"
}
