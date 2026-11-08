package main

// Dateizugriff im Serverordner. Alles wird über safeJoin geprüft, damit
// Symlinks aus dem Container nicht auf das Host-System zeigen können.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// safeJoin liefert einen echten Pfad unterhalb von root oder einen Fehler.
func safeJoin(root, rel string) (string, error) {
	rel = path.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	p := filepath.Join(root, filepath.FromSlash(rel))
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		rootReal = root
	}
	// Den längsten existierenden Teil auflösen.
	cur := p
	rest := ""
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			full := filepath.Join(r, rest)
			if full != rootReal && !strings.HasPrefix(full, rootReal+string(filepath.Separator)) {
				return "", fmt.Errorf("Pfad liegt außerhalb des Servers")
			}
			return full, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("ungültiger Pfad")
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

type FileEntry struct {
	Name     string    `json:"name"`
	Dir      bool      `json:"dir"`
	Size     int64     `json:"size"`
	Mod      time.Time `json:"mod"`
	Link     bool      `json:"link"`
	Mode     string    `json:"mode"`
	Editable bool      `json:"editable"`
}

func listDir(root, rel string) ([]FileEntry, error) {
	p, err := safeJoin(root, rel)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	out := []FileEntry{}
	for _, e := range ents {
		fi, err := e.Info()
		if err != nil {
			continue
		}
		fe := FileEntry{Name: e.Name(), Dir: fi.IsDir(), Size: fi.Size(), Mod: fi.ModTime(), Mode: fi.Mode().Perm().String()}
		if fi.Mode()&os.ModeSymlink != 0 {
			fe.Link = true
			if st, err := os.Stat(filepath.Join(p, e.Name())); err == nil {
				fe.Dir = st.IsDir()
			}
		}
		fe.Editable = !fe.Dir && fe.Size <= 4<<20 && looksText(e.Name())
		out = append(out, fe)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

var binaryExt = map[string]bool{
	".jar": true, ".zip": true, ".gz": true, ".tgz": true, ".tar": true, ".rar": true, ".7z": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true,
	".so": true, ".dll": true, ".exe": true, ".bin": true, ".dat": true, ".mca": true, ".mcr": true,
	".db": true, ".sqlite": true, ".class": true, ".pak": true, ".xz": true, ".bz2": true, ".nbt": true,
	".mp3": true, ".ogg": true, ".wav": true, ".pdf": true, ".lock": false,
}

func looksText(name string) bool {
	return !binaryExt[strings.ToLower(filepath.Ext(name))]
}

func isBinary(b []byte) bool {
	n := len(b)
	if n > 8000 {
		n = 8000
	}
	for _, c := range b[:n] {
		if c == 0 {
			return true
		}
	}
	return false
}

// ---------- Archive ----------

func compressZip(root string, dirRel string, names []string, dest string) error {
	base, err := safeJoin(root, dirRel)
	if err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, n := range names {
		src, err := safeJoin(root, path.Join(dirRel, n))
		if err != nil {
			return err
		}
		err = filepath.Walk(src, func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			if p == dest {
				return nil
			}
			rel, _ := filepath.Rel(base, p)
			rel = filepath.ToSlash(rel)
			if fi.IsDir() {
				if rel != "." {
					zw.Create(rel + "/")
				}
				return nil
			}
			h, _ := zip.FileInfoHeader(fi)
			h.Name = rel
			h.Method = zip.Deflate
			w, err := zw.CreateHeader(h)
			if err != nil {
				return err
			}
			r, err := os.Open(p)
			if err != nil {
				return nil
			}
			defer r.Close()
			_, err = io.Copy(w, r)
			return err
		})
		if err != nil {
			return err
		}
	}
	return zw.Close()
}

// extract entpackt zip, tar, tar.gz in den Zielordner (Pfade werden geprüft).
func extractArchive(root, archiveRel, destRel string) error {
	src, err := safeJoin(root, archiveRel)
	if err != nil {
		return err
	}
	dest, err := safeJoin(root, destRel)
	if err != nil {
		return err
	}
	low := strings.ToLower(src)
	switch {
	case strings.HasSuffix(low, ".zip"):
		return extractZip(src, root, dest)
	case strings.HasSuffix(low, ".tar.gz"), strings.HasSuffix(low, ".tgz"), strings.HasSuffix(low, ".tar"):
		return extractTar(src, root, dest)
	}
	return fmt.Errorf("nur .zip, .tar und .tar.gz werden unterstützt")
}

func targetIn(root, dest, name string) (string, error) {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return "", err
	}
	return safeJoin(root, path.Join(filepath.ToSlash(rel), name))
}

func extractZip(src, root, dest string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		t, err := targetIn(root, dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(t, 0755)
			os.Lchown(t, containerUID, containerUID)
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		os.MkdirAll(filepath.Dir(t), 0755)
		r, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(t, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm()|0600)
		if err != nil {
			r.Close()
			return err
		}
		_, err = io.Copy(w, r)
		r.Close()
		w.Close()
		os.Lchown(t, containerUID, containerUID)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTar(src, root, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if !strings.HasSuffix(strings.ToLower(src), ".tar") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	}
	return untar(r, root, dest)
}

func untar(r io.Reader, root, dest string) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		t, err := targetIn(root, dest, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(t, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(t), 0755)
			w, err := os.OpenFile(t, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode).Perm()|0600)
			if err != nil {
				return err
			}
			_, err = io.Copy(w, tr)
			w.Close()
			if err != nil {
				return err
			}
			os.Chtimes(t, h.ModTime, h.ModTime)
		default:
			continue
		}
		os.Lchown(t, containerUID, containerUID)
	}
}

// tarDir schreibt den Ordner als tar.gz (für Backups). ignore sind Muster.
func tarDir(dir string, w io.Writer, ignore []string) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if ignored(rel, fi.IsDir(), ignore) {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return nil
		}
		h.Name = rel
		if fi.IsDir() {
			h.Name += "/"
		}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return nil
			}
			_, err = io.CopyN(tw, f, h.Size)
			f.Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func ignored(rel string, dir bool, pats []string) bool {
	for _, p := range pats {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		p = strings.TrimPrefix(strings.TrimSuffix(p, "/"), "/")
		if ok, _ := path.Match(p, rel); ok {
			return true
		}
		if ok, _ := path.Match(p, path.Base(rel)); ok && !strings.Contains(p, "/") {
			return true
		}
	}
	return false
}
