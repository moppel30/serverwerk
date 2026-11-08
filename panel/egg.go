package main

// Eggs: Pterodactyl (PTDL_v1, PTDL_v2) und Pelican (PLCN_*, JSON oder YAML)
// werden in ein gemeinsames Format übersetzt.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type EggImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type ConfigFile struct {
	Parser string         `json:"parser"`
	Find   map[string]any `json:"find"`
}

type EggVar struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Env          string `json:"env"`
	Default      string `json:"default"`
	UserViewable bool   `json:"userViewable"`
	UserEditable bool   `json:"userEditable"`
	Rules        string `json:"rules"`
	Sort         int    `json:"sort"`
}

type Egg struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Author       string                `json:"author"`
	Description  string                `json:"description"`
	Format       string                `json:"format"`
	Source       string                `json:"source"`
	UpdateURL    string                `json:"updateUrl"`
	Category     string                `json:"category"`
	Icon         string                `json:"icon,omitempty"`
	Tags         []string              `json:"tags"`
	Features     []string              `json:"features"`
	Images       []EggImage            `json:"images"`
	Startup      string                `json:"startup"`
	Files        map[string]ConfigFile `json:"files"`
	Done         []string              `json:"done"`
	Stop         string                `json:"stop"`
	InstScript   string                `json:"installScript"`
	InstImage    string                `json:"installImage"`
	InstEntry    string                `json:"installEntrypoint"`
	Variables    []EggVar              `json:"variables"`
	FileDenylist []string              `json:"fileDenylist"`
	Imported     time.Time             `json:"imported"`
}

func (e *Egg) HasFeature(f string) bool {
	for _, x := range e.Features {
		if x == f {
			return true
		}
	}
	return false
}

// ParseEgg erkennt JSON oder YAML und alle bekannten Versionen.
func ParseEgg(data []byte) (*Egg, error) {
	var raw map[string]any
	t := strings.TrimSpace(string(data))
	if strings.HasPrefix(t, "{") {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("JSON nicht lesbar: %v", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("YAML nicht lesbar: %v", err)
		}
	}
	if raw == nil {
		return nil, fmt.Errorf("leere Datei")
	}
	e := &Egg{Files: map[string]ConfigFile{}}
	if m, ok := raw["meta"].(map[string]any); ok {
		e.Format = str(m["version"])
		e.UpdateURL = str(m["update_url"])
	}
	if e.Format == "" {
		return nil, fmt.Errorf("kein Egg (meta.version fehlt)")
	}
	e.Name = str(raw["name"])
	e.Author = str(raw["author"])
	e.Description = str(raw["description"])
	e.Tags = strList(raw["tags"])
	if ic := str(raw["icon"]); strings.HasPrefix(ic, "data:image/") || strings.HasPrefix(ic, "https://") {
		e.Icon = ic
	}
	e.Features = strList(raw["features"])
	e.FileDenylist = strList(raw["file_denylist"])

	// Images: PTDL_v1 hat "image"/"images", v2 und Pelican eine Map Name -> Image.
	switch di := raw["docker_images"].(type) {
	case map[string]any:
		for k, v := range di {
			e.Images = append(e.Images, EggImage{Name: k, Image: str(v)})
		}
		// Neueste Version zuerst (Java 25 vor Java 8), sonst alphabetisch.
		sort.SliceStable(e.Images, func(i, j int) bool { return natLess(e.Images[j].Name, e.Images[i].Name) })
	case []any:
		for _, v := range di {
			e.Images = append(e.Images, EggImage{Name: str(v), Image: str(v)})
		}
	}
	if len(e.Images) == 0 {
		for _, v := range strList(raw["images"]) {
			e.Images = append(e.Images, EggImage{Name: v, Image: v})
		}
		if im := str(raw["image"]); im != "" && len(e.Images) == 0 {
			e.Images = append(e.Images, EggImage{Name: im, Image: im})
		}
	}

	e.Startup = str(raw["startup"])
	if sc, ok := raw["startup_commands"].(map[string]any); ok && e.Startup == "" {
		if d, ok := sc["Default"]; ok {
			e.Startup = str(d)
		} else {
			for _, v := range sc {
				e.Startup = str(v)
				break
			}
		}
	}

	if cfg, ok := raw["config"].(map[string]any); ok {
		// Pterodactyl speichert die Unterobjekte als JSON-Text, Pelican als Objekt.
		if files := asObject(cfg["files"]); files != nil {
			for name, v := range files {
				fm, _ := v.(map[string]any)
				if fm == nil {
					continue
				}
				cf := ConfigFile{Parser: str(fm["parser"]), Find: map[string]any{}}
				if f, ok := fm["find"].(map[string]any); ok {
					cf.Find = f
				}
				e.Files[name] = cf
			}
		}
		if su := asObject(cfg["startup"]); su != nil {
			switch d := su["done"].(type) {
			case string:
				if d != "" {
					e.Done = []string{d}
				}
			case []any:
				for _, x := range d {
					if s := str(x); s != "" {
						e.Done = append(e.Done, s)
					}
				}
			}
		} else if s, ok := cfg["startup"].(string); ok && !strings.HasPrefix(strings.TrimSpace(s), "{") && s != "" {
			e.Done = []string{s}
		}
		e.Stop = str(cfg["stop"])
	}
	if e.Stop == "" {
		e.Stop = "^C"
	}

	if sc, ok := raw["scripts"].(map[string]any); ok {
		if in, ok := sc["installation"].(map[string]any); ok {
			e.InstScript = strings.ReplaceAll(str(in["script"]), "\r\n", "\n")
			e.InstImage = str(in["container"])
			e.InstEntry = str(in["entrypoint"])
		}
	}
	if e.InstEntry == "" {
		e.InstEntry = "bash"
	}

	if vars, ok := raw["variables"].([]any); ok {
		for i, v := range vars {
			vm, _ := v.(map[string]any)
			if vm == nil {
				continue
			}
			ev := EggVar{
				Name:         str(vm["name"]),
				Description:  str(vm["description"]),
				Env:          str(vm["env_variable"]),
				Default:      str(vm["default_value"]),
				UserViewable: truthy(vm["user_viewable"]),
				UserEditable: truthy(vm["user_editable"]),
				Sort:         i,
			}
			if s, ok := vm["sort"]; ok {
				if n, err := fmt.Sscan(str(s), &ev.Sort); n == 0 || err != nil {
					ev.Sort = i
				}
			}
			switch r := vm["rules"].(type) {
			case string:
				ev.Rules = r
			case []any:
				ev.Rules = strings.Join(strList(r), "|")
			}
			if ev.Env != "" {
				e.Variables = append(e.Variables, ev)
			}
		}
		sort.SliceStable(e.Variables, func(i, j int) bool { return e.Variables[i].Sort < e.Variables[j].Sort })
	}
	if e.Name == "" || e.Startup == "" || len(e.Images) == 0 {
		return nil, fmt.Errorf("Egg unvollständig (Name, Startbefehl oder Image fehlt)")
	}
	return e, nil
}

func asObject(v any) map[string]any {
	switch x := v.(type) {
	case map[string]any:
		return x
	case string:
		x = strings.TrimSpace(x)
		if x == "" || !strings.HasPrefix(x, "{") {
			return nil
		}
		var m map[string]any
		if json.Unmarshal([]byte(x), &m) == nil {
			return m
		}
	}
	return nil
}

func str(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprint(int64(x))
		}
		return fmt.Sprint(x)
	default:
		return fmt.Sprint(x)
	}
}

func strList(v any) []string {
	var out []string
	switch x := v.(type) {
	case []any:
		for _, s := range x {
			if t := str(s); t != "" {
				out = append(out, t)
			}
		}
	case map[string]any:
		for _, s := range x {
			if t := str(s); t != "" {
				out = append(out, t)
			}
		}
	}
	return out
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case string:
		return x == "1" || x == "true"
	}
	return false
}

// natLess vergleicht Texte mit Zahlen darin natürlich ("Java 8" < "Java 21").
func natLess(a, b string) bool {
	for a != "" && b != "" {
		ca, cb := a[0], b[0]
		if isDigit(ca) && isDigit(cb) {
			na, ra := takeNum(a)
			nb, rb := takeNum(b)
			if na != nb {
				return na < nb
			}
			a, b = ra, rb
			continue
		}
		if ca != cb {
			return ca < cb
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func takeNum(s string) (int, string) {
	n, i := 0, 0
	for i < len(s) && isDigit(s[i]) {
		n = n*10 + int(s[i]-'0')
		i++
	}
	return n, s[i:]
}

// Werte, die der Startbefehl und die Konfigdateien kennen.
func (e *Egg) DefaultEnv() map[string]string {
	m := map[string]string{}
	for _, v := range e.Variables {
		m[v.Env] = v.Default
	}
	return m
}

// ValidateVar prüft die gängigen Laravel-Regeln aus den Eggs.
func ValidateVar(v EggVar, val string) error {
	rules := strings.Split(v.Rules, "|")
	nullable := false
	for _, r := range rules {
		if r == "nullable" {
			nullable = true
		}
	}
	if val == "" {
		for _, r := range rules {
			if r == "required" {
				return fmt.Errorf("%s darf nicht leer sein", v.Name)
			}
		}
		return nil
	}
	_ = nullable
	for _, r := range rules {
		name, arg, _ := strings.Cut(r, ":")
		switch name {
		case "numeric", "integer":
			var f float64
			if _, err := fmt.Sscan(val, &f); err != nil {
				return fmt.Errorf("%s muss eine Zahl sein", v.Name)
			}
		case "boolean":
			if val != "0" && val != "1" && val != "true" && val != "false" {
				return fmt.Errorf("%s muss 0 oder 1 sein", v.Name)
			}
		case "in":
			ok := false
			for _, o := range strings.Split(arg, ",") {
				if o == val {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("%s: erlaubt sind %s", v.Name, arg)
			}
		case "max":
			var n int
			fmt.Sscan(arg, &n)
			if n > 0 && !hasRule(rules, "numeric") && !hasRule(rules, "integer") && len(val) > n {
				return fmt.Errorf("%s: höchstens %d Zeichen", v.Name, n)
			}
		}
	}
	return nil
}

func hasRule(rules []string, n string) bool {
	for _, r := range rules {
		if r == n {
			return true
		}
	}
	return false
}

// Auswahlwerte für Variablen mit "in:a,b,c" oder "boolean".
func (v EggVar) Options() []string {
	for _, r := range strings.Split(v.Rules, "|") {
		if strings.HasPrefix(r, "in:") {
			return strings.Split(r[3:], ",")
		}
	}
	return nil
}
