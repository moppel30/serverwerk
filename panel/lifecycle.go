package main

// Herunterfahren und Hochfahren des Rechners:
// - Fährt der Rechner herunter, bekommt jeder laufende Server seinen Stopp-Befehl
//   aus dem Egg, und Serverwerk wartet, bis alle gespeichert haben.
// - Wird nur Serverwerk neu gestartet (z. B. Update), laufen die Server einfach weiter.
// - Nach dem Hochfahren starten alle Server wieder, die vorher liefen.

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const shutdownWait = 3 * time.Minute

// systemShuttingDown erkennt, ob der ganze Rechner herunterfährt oder neu startet.
func systemShuttingDown() bool {
	if v := os.Getenv("SERVERWERK_TEST_SHUTDOWN"); v != "" {
		return v == "1"
	}
	if out, err := exec.Command("systemctl", "is-system-running").Output(); err == nil || len(out) > 0 {
		if strings.TrimSpace(string(out)) == "stopping" {
			return true
		}
	}
	if out, err := exec.Command("systemctl", "list-jobs", "--no-legend").Output(); err == nil {
		s := string(out)
		for _, t := range []string{"shutdown.target", "reboot.target", "poweroff.target", "halt.target", "kexec.target"} {
			if strings.Contains(s, t) {
				return true
			}
		}
	}
	if b, err := os.ReadFile("/run/openrc/softlevel"); err == nil {
		l := strings.TrimSpace(string(b))
		return l == "shutdown" || l == "reboot"
	}
	return false
}

func (a *App) watchSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-ch
		if systemShuttingDown() {
			a.stopAllForShutdown()
		} else {
			log.Printf("Serverwerk wird beendet, Server laufen weiter")
		}
		os.Exit(0)
	}()
}

func (a *App) stopAllForShutdown() {
	var ids []string
	a.st.Read(func(s *State) {
		for _, x := range s.Servers {
			ids = append(ids, x.ID)
		}
	})
	var running []*Instance
	for _, id := range ids {
		in := a.m.Get(id)
		if st := in.State(); st == StRunning || st == StStarting {
			running = append(running, in)
		}
	}
	if len(running) == 0 {
		return
	}
	log.Printf("Rechner fährt herunter: %d Server werden sauber gestoppt", len(running))
	for _, in := range running {
		in.Say("Rechner fährt herunter, Server wird gestoppt und gespeichert ...")
		in.Stop(false) // merkt sich "soll laufen" weiter, damit er nach dem Neustart wieder startet
	}
	deadline := time.Now().Add(shutdownWait)
	for time.Now().Before(deadline) {
		left := 0
		for _, in := range running {
			if in.State() != StOffline {
				left++
			}
		}
		if left == 0 {
			log.Printf("Alle Server sind gestoppt")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Printf("Nicht alle Server haben rechtzeitig gestoppt")
}

// bootStart startet nach dem Hochfahren die Server, die vorher liefen.
func (a *App) bootStart() {
	var list []*Server
	a.st.Read(func(s *State) {
		for _, x := range s.Servers {
			if x.WantRunning && !x.NoBootStart && x.Install == "ok" {
				list = append(list, cloneServer(x))
			}
		}
	})
	for _, s := range list {
		in := a.m.Get(s.ID)
		if in.State() != StOffline {
			continue // läuft noch (nur Serverwerk wurde neu gestartet)
		}
		in.Say("Server lief vor dem Neustart, wird automatisch gestartet.")
		if err := in.Start(); err != nil {
			in.Say("Automatischer Start ging nicht: %v", err)
		}
		time.Sleep(3 * time.Second) // nicht alle gleichzeitig
	}
}
