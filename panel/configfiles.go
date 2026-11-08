package main

// Konfigdateien aus dem Egg (config.files) vor jedem Start anpassen,
// wie es Wings macht: Port, IP, Speicher usw. in server.properties & Co.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var placeholderRe = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

type varCtx struct {
	srv     *Server
	env     map[string]string
	gateway string
}

func (c *varCtx) lookup(key string) (string, bool) {
	k := strings.TrimSpace(key)
	port := 0
	if len(c.srv.Ports) > 0 {
		port = c.srv.Ports[0]
	}
	switch k {
	case "server.build.default.port", "server.allocations.default.port":
		return strconv.Itoa(port), true
	case "server.build.default.ip", "server.allocations.default.ip":
		return "0.0.0.0", true
	case "server.build.memory", "server.build.memory_limit":
		return strconv.Itoa(c.srv.Memory), true
	case "server.build.cpu", "server.build.cpu_limit":
		return strconv.Itoa(c.srv.CPU), true
	case "server.build.disk", "server.build.disk_space":
		return strconv.Itoa(c.srv.Disk), true
	case "config.docker.interface", "config.docker.network.interface":
		return c.gateway, true
	case "server.uuid":
		return c.srv.ID, true
	case "server.name":
		return c.srv.Name, true
	}
	for _, p := range []string{"server.build.env.", "server.environment.", "env."} {
		if strings.HasPrefix(k, p) {
			return c.env[strings.TrimPrefix(k, p)], true
		}
	}
	return "", false
}

func (c *varCtx) expand(s string) string {
	return placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := placeholderRe.FindStringSubmatch(m)[1]
		if v, ok := c.lookup(inner); ok {
			return v
		}
		return m
	})
}

// valueOf wandelt einen "find"-Wert in Text um. Wings erlaubt auch Objekte
// der Form {"alterWert": "neuerWert"}; dort nehmen wir den ersten neuen Wert.
// apply berechnet den neuen Wert für einen "find"-Eintrag wie Wings:
//   - Text: wird immer gesetzt (Platzhalter ersetzt).
//   - Objekt {"alt": "neu"}: nur ersetzen, wenn der jetzige Wert "alt" ist.
//   - Objekt {"regex:MUSTER": "neu"}: nur ersetzen, wenn MUSTER passt; $1, $2 ... aus dem Muster.
func (c *varCtx) apply(v any, cur string, exists bool) (string, bool) {
	m, isMap := v.(map[string]any)
	if !isMap {
		return c.valueOf(v), true
	}
	if !exists {
		return "", false
	}
	for k, rep := range m {
		if pat, ok := strings.CutPrefix(k, "regex:"); ok {
			re, err := regexp.Compile(pat)
			if err != nil || !re.MatchString(cur) {
				continue
			}
			return re.ReplaceAllString(cur, c.expand(str(rep))), true
		}
		if cur == c.expand(k) {
			return c.valueOf(rep), true
		}
	}
	return "", false
}

func (c *varCtx) valueOf(v any) string {
	switch x := v.(type) {
	case string:
		return c.expand(x)
	default:
		return str(x)
	}
}

func applyConfigFiles(root string, e *Egg, c *varCtx) []string {
	var warn []string
	for name, cf := range e.Files {
		p, err := safeJoin(root, name)
		if err != nil {
			warn = append(warn, name+": "+err.Error())
			continue
		}
		os.MkdirAll(filepath.Dir(p), 0755)
		old, _ := os.ReadFile(p)
		var out []byte
		switch strings.ToLower(cf.Parser) {
		case "properties":
			out = editLines(old, cf.Find, c, "=")
		case "ini":
			out = editIni(old, cf.Find, c)
		case "file":
			out = editPrefix(old, cf.Find, c)
		case "yaml", "yml":
			out, err = editYAML(old, cf.Find, c)
		case "json":
			out, err = editJSON(old, cf.Find, c)
		case "xml":
			out = editXML(old, cf.Find, c)
		default:
			err = fmt.Errorf("unbekannter Parser %q", cf.Parser)
		}
		if err != nil {
			warn = append(warn, name+": "+err.Error())
			continue
		}
		if string(out) != string(old) {
			if err := os.WriteFile(p, out, 0644); err != nil {
				warn = append(warn, name+": "+err.Error())
			}
			os.Lchown(p, containerUID, containerUID)
		}
	}
	return warn
}

func splitLines(b []byte) ([]string, string) {
	s := string(b)
	nl := "\n"
	if strings.Contains(s, "\r\n") {
		nl = "\r\n"
	}
	if s == "" {
		return nil, nl
	}
	s = strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	return strings.Split(s, "\n"), nl
}

func joinLines(lines []string, nl string) []byte {
	return []byte(strings.Join(lines, nl) + nl)
}

// key=value, fehlende Schlüssel werden angehängt (nur bei festen Werten).
func editLines(old []byte, find map[string]any, c *varCtx, sep string) []byte {
	lines, nl := splitLines(old)
	for key, v := range find {
		found := false
		for i, l := range lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "!") {
				continue
			}
			k, cur, ok := strings.Cut(t, sep)
			if !ok {
				k2, cur2, ok2 := strings.Cut(t, ":")
				if !ok2 {
					continue
				}
				k, cur = k2, cur2
			}
			if strings.TrimSpace(k) == key {
				found = true
				if nv, ok := c.apply(v, strings.TrimSpace(cur), true); ok {
					lines[i] = key + sep + nv
				}
			}
		}
		if !found {
			if nv, ok := c.apply(v, "", false); ok {
				lines = append(lines, key+sep+nv)
			}
		}
	}
	return joinLines(lines, nl)
}

func editIni(old []byte, find map[string]any, c *varCtx) []byte {
	lines, nl := splitLines(old)
	for full, v := range find {
		section, key := "", full
		if i := strings.LastIndex(full, "."); i > 0 {
			section, key = full[:i], full[i+1:]
		}
		cur := ""
		secEnd := -1
		found := false
		for i, l := range lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
				cur = t[1 : len(t)-1]
				continue
			}
			if cur != section {
				continue
			}
			secEnd = i
			k, val, ok := strings.Cut(t, "=")
			if ok && strings.TrimSpace(k) == key {
				found = true
				if nv, ok := c.apply(v, strings.TrimSpace(val), true); ok {
					lines[i] = key + "=" + nv
				}
			}
		}
		if found {
			continue
		}
		nv, ok := c.apply(v, "", false)
		if !ok {
			continue
		}
		if section == "" {
			lines = append([]string{key + "=" + nv}, lines...)
		} else if secEnd >= 0 {
			lines = append(lines[:secEnd+1], append([]string{key + "=" + nv}, lines[secEnd+1:]...)...)
		} else {
			hasHeader := false
			for _, l := range lines {
				if strings.TrimSpace(l) == "["+section+"]" {
					hasHeader = true
				}
			}
			if !hasHeader {
				lines = append(lines, "", "["+section+"]")
			}
			lines = append(lines, key+"="+nv)
		}
	}
	return joinLines(lines, nl)
}

// Zeilen, die mit dem Schlüssel anfangen, werden komplett ersetzt.
func editPrefix(old []byte, find map[string]any, c *varCtx) []byte {
	lines, nl := splitLines(old)
	for key, v := range find {
		for i, l := range lines {
			if strings.HasPrefix(strings.TrimLeft(l, " \t"), key) {
				if nv, ok := c.apply(v, l, true); ok {
					lines[i] = nv
				}
			}
		}
	}
	return joinLines(lines, nl)
}

func typed(s string) any {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && (s == "0" || !strings.HasPrefix(s, "0")) {
		return n
	}
	return s
}

// splitPath macht aus "listeners[0].host" die Teile listeners, 0, host.
func splitPath(key string) []string {
	var out []string
	for _, p := range strings.Split(key, ".") {
		for {
			i := strings.Index(p, "[")
			j := strings.Index(p, "]")
			if i < 0 || j < i {
				break
			}
			if i > 0 {
				out = append(out, p[:i])
			}
			out = append(out, p[i+1:j])
			p = p[j+1:]
		}
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

type setFn func(cur any, exists bool) (any, bool)

func editJSON(old []byte, find map[string]any, c *varCtx) ([]byte, error) {
	var doc any = map[string]any{}
	if len(strings.TrimSpace(string(old))) > 0 {
		if err := json.Unmarshal(old, &doc); err != nil {
			return nil, err
		}
	}
	for key, v := range find {
		v := v
		fn := func(cur any, exists bool) (any, bool) {
			if s, ok := v.(string); ok {
				return typed(c.expand(s)), true
			}
			if _, isMap := v.(map[string]any); isMap {
				nv, ok := c.apply(v, str(cur), exists)
				if !ok {
					return nil, false
				}
				return typed(nv), true
			}
			return v, true
		}
		doc = setPath(doc, splitPath(key), fn)
	}
	return json.MarshalIndent(doc, "", "    ")
}

func setPath(node any, path []string, fn setFn) any {
	if len(path) == 0 {
		nv, ok := fn(node, node != nil)
		if !ok {
			return node
		}
		return nv
	}
	switch n := node.(type) {
	case map[string]any:
		if path[0] == "*" {
			for k := range n {
				n[k] = setPath(n[k], path[1:], fn)
			}
			return n
		}
		if _, exists := n[path[0]]; !exists && len(path) == 1 {
			if nv, ok := fn(nil, false); ok {
				n[path[0]] = nv
			}
			return n
		}
		n[path[0]] = setPath(n[path[0]], path[1:], fn)
		return n
	case []any:
		if path[0] == "*" {
			for i := range n {
				n[i] = setPath(n[i], path[1:], fn)
			}
			return n
		}
		if i, err := strconv.Atoi(path[0]); err == nil && i >= 0 && i < len(n) {
			n[i] = setPath(n[i], path[1:], fn)
		}
		return n
	default:
		if path[0] == "*" {
			return node
		}
		m := map[string]any{}
		if len(path) == 1 {
			if nv, ok := fn(nil, false); ok {
				m[path[0]] = nv
			}
			return m
		}
		m[path[0]] = setPath(nil, path[1:], fn)
		return m
	}
}

func editYAML(old []byte, find map[string]any, c *varCtx) ([]byte, error) {
	var doc yaml.Node
	if len(strings.TrimSpace(string(old))) > 0 {
		if err := yaml.Unmarshal(old, &doc); err != nil {
			return nil, err
		}
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	for key, v := range find {
		v := v
		setYAML(doc.Content[0], splitPath(key), func(cur string, exists bool) (string, bool) { return c.apply(v, cur, exists) })
	}
	return yaml.Marshal(&doc)
}

func setYAML(n *yaml.Node, path []string, fn func(cur string, exists bool) (string, bool)) {
	if len(path) == 0 {
		return
	}
	if n.Kind == yaml.SequenceNode {
		if i, err := strconv.Atoi(path[0]); err == nil {
			if i >= 0 && i < len(n.Content) {
				ch := n.Content[i]
				if len(path) == 1 {
					if nv, ok := fn(ch.Value, true); ok {
						setScalar(ch, nv)
					}
				} else {
					setYAML(ch, path[1:], fn)
				}
			}
			return
		}
	}
	if n.Kind != yaml.MappingNode {
		if n.Kind == yaml.SequenceNode && path[0] == "*" {
			for _, ch := range n.Content {
				if len(path) == 1 {
					if nv, ok := fn(ch.Value, true); ok {
						setScalar(ch, nv)
					}
				} else {
					setYAML(ch, path[1:], fn)
				}
			}
		}
		return
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if path[0] == "*" || k.Value == path[0] {
			if len(path) == 1 {
				if nv, ok := fn(v.Value, true); ok {
					setScalar(v, nv)
				}
			} else {
				setYAML(v, path[1:], fn)
			}
			if path[0] != "*" {
				return
			}
		}
	}
	if path[0] == "*" {
		return
	}
	// Schlüssel fehlt: nur bei festen Werten anlegen.
	if len(path) == 1 {
		nv, ok := fn("", false)
		if !ok {
			return
		}
		k := &yaml.Node{Kind: yaml.ScalarNode, Value: path[0]}
		v := &yaml.Node{Kind: yaml.ScalarNode}
		setScalar(v, nv)
		n.Content = append(n.Content, k, v)
		return
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Value: path[0]}
	v := &yaml.Node{Kind: yaml.MappingNode}
	n.Content = append(n.Content, k, v)
	setYAML(v, path[1:], fn)
}

func setScalar(n *yaml.Node, val string) {
	n.Kind = yaml.ScalarNode
	n.Content = nil
	n.Value = val
	n.Style = 0
	switch typed(val).(type) {
	case bool:
		n.Tag = "!!bool"
	case int64:
		n.Tag = "!!int"
	default:
		n.Tag = "!!str"
	}
}

// XML: einfacher Ansatz über den letzten Pfadteil, reicht für die üblichen Eggs.
func editXML(old []byte, find map[string]any, c *varCtx) []byte {
	s := string(old)
	for key, v := range find {
		parts := strings.Split(key, ".")
		last := parts[len(parts)-1]
		if strings.HasPrefix(last, "@") || strings.Contains(last, "[@") {
			continue
		}
		re, err := regexp.Compile(`(<` + regexp.QuoteMeta(last) + `(?:\s[^>]*)?>)([^<]*)(</` + regexp.QuoteMeta(last) + `>)`)
		if err != nil {
			continue
		}
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			sm := re.FindStringSubmatch(m)
			nv, ok := c.apply(v, sm[2], true)
			if !ok {
				return m
			}
			return sm[1] + nv + sm[3]
		})
	}
	return []byte(s)
}
