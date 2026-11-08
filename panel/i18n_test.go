package main

import (
	"io/fs"
	"os"
	"testing"
)

func TestTranslate(t *testing.T) {
	sub, _ := fs.Sub(os.DirFS("."), "web")
	loadDict(sub)
	setLang("en")
	defer setLang("de")
	cases := map[string]string{
		"Port 25570 nutzt schon ein anderer Server":  "Port 25570 is already used by another server",
		"Server ist aus (Exit-Code 0).":              "Server is off (exit code 0).",
		"Konfigurationsdateien werden angepasst ...": "Updating configuration files ...",
		"Name oder Passwort falsch":                  "Wrong name or password",
		"Portal bleibt":                              "Portal bleibt",
	}
	for in, want := range cases {
		if got := T(in); got != want {
			t.Errorf("T(%q) = %q, erwartet %q", in, got, want)
		}
	}
	setLang("de")
	if got := T("Name oder Passwort falsch"); got != "Name oder Passwort falsch" {
		t.Errorf("Deutsch darf nicht übersetzt werden: %q", got)
	}
}
