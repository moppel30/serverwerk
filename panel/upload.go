package main

// Große Dateien in Teilen hochladen. Cloudflare (Tunnel) lässt pro Anfrage
// nur 100 MB durch, deshalb schickt der Browser große Dateien in Stücken.
// Die Teile landen außerhalb des Serverordners und werden erst am Ende an
// ihren Platz verschoben, damit der Spielserver nie halbe Dateien sieht.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

const maxChunk = 64 << 20

var uploadIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

func (a *App) uploadDir() string { return filepath.Join(a.data, "uploads") }

// cleanUploads löscht liegengebliebene Teile (abgebrochene Uploads).
func (a *App) cleanUploads() {
	es, _ := os.ReadDir(a.uploadDir())
	for _, e := range es {
		if fi, err := e.Info(); err == nil && time.Since(fi.ModTime()) > 24*time.Hour {
			os.Remove(filepath.Join(a.uploadDir(), e.Name()))
		}
	}
}

// fileChunk: POST ?path=<Ziel-Datei>&upload=<id>&offset=<n>&total=<n>, Inhalt = rohe Bytes.
func (a *App) fileChunk(w http.ResponseWriter, r *http.Request, s *Server) {
	q := r.URL.Query()
	rel, id := q.Get("path"), q.Get("upload")
	offset, err1 := strconv.ParseInt(q.Get("offset"), 10, 64)
	total, err2 := strconv.ParseInt(q.Get("total"), 10, 64)
	if !uploadIDRe.MatchString(id) || err1 != nil || err2 != nil || offset < 0 || total < 0 || offset > total {
		jsonErr(w, 400, "Ungültige Anfrage")
		return
	}
	target, err := safeJoin(a.root(s), rel)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	if a.denied(s, rel) {
		jsonErr(w, 403, "Diese Datei ist gesperrt")
		return
	}
	os.MkdirAll(a.uploadDir(), 0700)
	part := filepath.Join(a.uploadDir(), s.ID+"-"+id+".part")
	if offset == 0 {
		a.cleanUploads()
		os.Remove(part)
	}
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	fi, _ := f.Stat()
	if fi.Size() != offset {
		// Teil fehlt oder doppelt: Browser macht ab dieser Stelle weiter.
		f.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(409)
		fmt.Fprintf(w, `{"error":"offset","size":%d}`, fi.Size())
		return
	}
	f.Seek(offset, io.SeekStart)
	n, err := io.Copy(f, http.MaxBytesReader(w, r.Body, maxChunk))
	if err == nil && offset+n > total {
		err = fmt.Errorf("mehr Daten als angekündigt")
	}
	if err != nil {
		f.Truncate(offset)
		f.Close()
		jsonErr(w, 400, "Teil unvollständig: "+err.Error())
		return
	}
	f.Close()
	if offset+n < total {
		jsonOut(w, map[string]any{"size": offset + n})
		return
	}
	// Fertig: an den Platz verschieben.
	os.MkdirAll(filepath.Dir(target), 0755)
	os.Lchown(filepath.Dir(target), containerUID, containerUID)
	if err := os.Rename(part, target); err != nil {
		if err := copyFile(part, target); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		os.Remove(part)
	}
	os.Chmod(target, 0644)
	os.Lchown(target, containerUID, containerUID)
	jsonOut(w, map[string]any{"size": total, "done": true})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
