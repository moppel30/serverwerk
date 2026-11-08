package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

var version = "0.11.4"

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", envOr("SERVERWERK_ADDR", ":8200"), "Adresse der Weboberfläche")
	data := flag.String("daten", envOr("SERVERWERK_DATEN", "/var/lib/serverwerk"), "Datenordner")
	webDir := flag.String("web", "", "Weboberfläche von der Platte lesen (Entwicklung)")
	langFlag := flag.String("lang", envOr("SERVERWERK_LANG", ""), "Sprache bei der Ersteinrichtung: en oder de")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Serverwerk %s\n\n  serverwerk [-addr :8200] [-daten /var/lib/serverwerk]\n  serverwerk passwort <name> [neues-passwort]   Passwort zurücksetzen\n  serverwerk version\n\n", version)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 0 {
		switch flag.Arg(0) {
		case "version":
			fmt.Println(version)
			return
		case "passwort", "password":
			resetPassword(*data, flag.Args()[1:])
			return
		default:
			flag.Usage()
			os.Exit(2)
		}
	}

	for _, d := range []string{"", "server", "backups", "tmp"} {
		os.MkdirAll(filepath.Join(*data, d), 0755)
	}
	os.Chmod(filepath.Join(*data, "backups"), 0700)
	st, err := openStore(*data)
	if err != nil {
		log.Fatalf("Zustand: %v", err)
	}
	// Sprache: einmal festlegen, danach in den Einstellungen änderbar.
	st.Update(func(s *State) error {
		if s.Settings.Language == "" {
			switch {
			case len(s.Users) > 0:
				s.Settings.Language = "de" // bestehende Installationen bleiben deutsch
			case *langFlag == "de" || *langFlag == "en":
				s.Settings.Language = *langFlag
			default:
				s.Settings.Language = "en"
			}
		}
		setLang(s.Settings.Language)
		return nil
	})
	if sub, err := fs.Sub(webFS, "web"); err == nil {
		loadDict(sub)
	}
	ensureSetupCode(st, *data)

	dk := newDocker()
	app := &App{st: st, dk: dk, data: *data, version: version, cat: loadCatalog(*data), icons: newIconCache(*data), arch: loadArchCache(*data)}
	app.m = newManager(st, dk, *data)
	if _, p, err := net.SplitHostPort(*addr); err == nil {
		app.port = p
	} else {
		app.port = "8200"
	}
	go func() {
		// Docker kann nach einem Neustart noch hochfahren: kurz warten.
		for i := 0; i < 30; i++ {
			ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := dk.Version(ctx)
			c()
			if err == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		app.m.Restore()
		app.ensureTunnel()
		app.restoreShims()
		go app.archLoop()
		go app.bootStart()
		go app.m.DiskLoop()
	}()
	go app.scheduleLoop()
	go app.dnsLoop()
	go app.dnsCleanupLoop()
	app.watchSignals()

	mux := http.NewServeMux()
	app.routes(mux)
	var web fs.FS
	if *webDir != "" {
		web = os.DirFS(*webDir)
	} else {
		web, _ = fs.Sub(webFS, "web")
	}
	mux.Handle("/", staticHandler(web))

	srv := &http.Server{Addr: *addr, Handler: securityHeaders(mux), ReadHeaderTimeout: 15 * time.Second}
	log.Printf("Serverwerk %s läuft auf %s (Daten: %s)", version, *addr, *data)
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		h.ServeHTTP(w, r)
	})
}

// staticHandler liefert die Seite aus; index.html bekommt Versionsnummern an CSS/JS.
func staticHandler(web fs.FS) http.Handler {
	files := http.FileServer(http.FS(web))
	hashOf := func(name string) string {
		b, err := fs.ReadFile(web, name)
		if err != nil {
			return "0"
		}
		h := sha256.Sum256(b)
		return hex.EncodeToString(h[:4])
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" || p == "index.html" || !strings.Contains(p, ".") {
			b, err := fs.ReadFile(web, "index.html")
			if err != nil {
				http.Error(w, "index.html fehlt", 500)
				return
			}
			s := string(b)
			s = strings.ReplaceAll(s, "app.css\"", "app.css?v="+hashOf("app.css")+"\"")
			s = strings.ReplaceAll(s, "app.js\"", "app.js?v="+hashOf("app.js")+"\"")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write([]byte(s))
			return
		}
		if r.URL.RawQuery != "" {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		files.ServeHTTP(w, r)
	})
}

func randomPassword() string {
	const chars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := make([]byte, 14)
	rand.Read(raw)
	out := make([]byte, len(raw))
	for i := range out {
		out[i] = chars[int(raw[i])%len(chars)]
	}
	return string(out)
}

func resetPassword(data string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Aufruf: serverwerk passwort <name> [neues-passwort]")
		os.Exit(2)
	}
	st, err := openStore(data)
	if err != nil {
		log.Fatal(err)
	}
	pw := ""
	if len(args) > 1 {
		pw = args[1]
	} else {
		pw = randomPassword()
	}
	if len(pw) < 8 {
		log.Fatal("Passwort braucht mindestens 8 Zeichen")
	}
	found := false
	err = st.Update(func(s *State) error {
		for _, u := range s.Users {
			if strings.EqualFold(u.Name, args[0]) {
				u.Salt = newID(16)
				u.Hash = hashPassword(pw, u.Salt)
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Benutzer %s gibt es nicht", args[0])
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Neues Passwort für %s: %s\n", args[0], pw)
}

// ---------- Werte des Rechners ----------

func readKV(path string) map[string]uint64 {
	out := map[string]uint64{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		fs := strings.Fields(v)
		if len(fs) > 0 {
			n, _ := strconv.ParseUint(fs[0], 10, 64)
			out[k] = n
		}
	}
	return out
}

var lastCPU struct{ idle, total uint64 }

func cpuPercent() float64 {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	fs := strings.Fields(line)
	if len(fs) < 5 {
		return 0
	}
	var total, idle uint64
	for i, f := range fs[1:] {
		n, _ := strconv.ParseUint(f, 10, 64)
		total += n
		if i == 3 || i == 4 {
			idle += n
		}
	}
	dt, di := total-lastCPU.total, idle-lastCPU.idle
	first := lastCPU.total == 0
	lastCPU.total, lastCPU.idle = total, idle
	if first || dt == 0 {
		return 0
	}
	return float64(dt-di) / float64(dt) * 100
}

func hostStats(data string) map[string]any {
	mem := readKV("/proc/meminfo")
	out := map[string]any{
		"memTotal": mem["MemTotal"] * 1024,
		"memUsed":  (mem["MemTotal"] - mem["MemAvailable"]) * 1024,
		"cpu":      cpuPercent(),
	}
	if total, free, ok := diskSpace(data); ok {
		out["diskTotal"] = total
		out["diskFree"] = free
	}
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		out["load"] = strings.Join(strings.Fields(string(b))[:3], " ")
	}
	if b, err := os.ReadFile("/proc/uptime"); err == nil {
		f, _ := strconv.ParseFloat(strings.Fields(string(b))[0], 64)
		out["uptime"] = int64(f)
	}
	return out
}
