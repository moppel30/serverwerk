package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBungeeEggRules(t *testing.T) {
	dir := t.TempDir()
	cfg := `listeners:
- host: 0.0.0.0:25569
  query_port: 25569
  motd: hallo
servers:
  lobby:
    address: localhost:25566
  pvp:
    address: 172.17.0.1:25567
  extern:
    address: play.example.com:25565
`
	os.WriteFile(filepath.Join(dir, "config.yml"), []byte(cfg), 0644)
	egg := &Egg{Files: map[string]ConfigFile{"config.yml": {Parser: "yaml", Find: map[string]any{
		"listeners[0].query_port": "{{server.build.default.port}}",
		"listeners[0].host":       "0.0.0.0:{{server.build.default.port}}",
		"servers.*.address": map[string]any{
			`regex:^(127\.0\.0\.1|localhost)(:\d{1,5})?$`: "{{config.docker.interface}}$2",
		},
	}}}}
	c := &varCtx{srv: &Server{Ports: []int{25570}}, env: map[string]string{}, gateway: "172.17.0.1"}
	if w := applyConfigFiles(dir, egg, c); len(w) > 0 {
		t.Fatal(w)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "config.yml"))
	out := string(b)
	for _, want := range []string{"host: 0.0.0.0:25570", "query_port: 25570", "address: 172.17.0.1:25566", "address: 172.17.0.1:25567", "address: play.example.com:25565", "motd: hallo"} {
		if !strings.Contains(out, want) {
			t.Errorf("fehlt %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "$2") || strings.Contains(out, "listeners[0]") {
		t.Errorf("kaputt:\n%s", out)
	}
}

func TestPropertiesExact(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.properties"), []byte("server-port=25565\nmotd=x\n"), 0644)
	egg := &Egg{Files: map[string]ConfigFile{"server.properties": {Parser: "properties", Find: map[string]any{
		"server-port": "{{server.build.default.port}}",
		"motd":        map[string]any{"x": "neu"},
		"fehlt":       map[string]any{"a": "b"},
	}}}}
	c := &varCtx{srv: &Server{Ports: []int{25570}}, env: map[string]string{}}
	applyConfigFiles(dir, egg, c)
	b, _ := os.ReadFile(filepath.Join(dir, "server.properties"))
	if string(b) != "server-port=25570\nmotd=neu\n" {
		t.Errorf("unerwartet:\n%s", b)
	}
}

func TestRewriteBackends(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yml"), []byte("servers:\n  a:\n    address: 172.18.0.1:25566\n  b:\n    address: 192.168.178.48:25567\n  c:\n    address: 172.18.0.1:30000\n  d:\n    address: localhost:25566\n"), 0644)
	egg := &Egg{Files: map[string]ConfigFile{"config.yml": {Parser: "yaml"}}}
	ch := rewriteBackends(dir, egg, []string{"172.18.0.1", "192.168.178.48", "127.0.0.1", "localhost"}, map[int]int{25566: 25570, 25567: 25571}, "172.17.0.1")
	b, _ := os.ReadFile(filepath.Join(dir, "config.yml"))
	want := "servers:\n  a:\n    address: 172.17.0.1:25570\n  b:\n    address: 172.17.0.1:25571\n  c:\n    address: 172.18.0.1:30000\n  d:\n    address: 172.17.0.1:25570\n"
	if string(b) != want || len(ch) != 3 {
		t.Errorf("unerwartet (%d Änderungen):\n%s", len(ch), b)
	}
}
