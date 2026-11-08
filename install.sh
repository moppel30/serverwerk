#!/bin/sh
# Serverwerk – one-command installer / Installation mit einem Befehl:
#   curl -fsSL https://github.com/moppel30/serverwerk/releases/latest/download/install.sh | sudo sh
#
# Options (after "sh -s --"):  --lang en|de   --port 8200   --uninstall
# Works on Debian, Ubuntu, Raspberry Pi OS, Mint, Fedora, RHEL/Rocky/Alma,
# CentOS, Arch/Manjaro, openSUSE, Alpine and Void. Run again = update.
set -eu

QUELLE="${SERVERWERK_URL:-https://github.com/moppel30/serverwerk/releases/latest/download}"
PORT="${SERVERWERK_PORT:-}"
SPRACHE="${SERVERWERK_LANG:-}"
DATEN="/var/lib/serverwerk"
BIN="/usr/local/bin/serverwerk"
ENTFERNEN=0

while [ $# -gt 0 ]; do
  case "$1" in
    --port) PORT="$2"; shift 2 ;;
    --port=*) PORT="${1#*=}"; shift ;;
    --lang|--sprache) SPRACHE="$2"; shift 2 ;;
    --lang=*|--sprache=*) SPRACHE="${1#*=}"; shift ;;
    --uninstall|--entfernen) ENTFERNEN=1; shift ;;
    --source|--quelle) QUELLE="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 2 ;;
  esac
done

if [ -t 1 ]; then A="$(printf '\033[38;2;217;199;167m')"; G="$(printf '\033[2m')"; R="$(printf '\033[31m')"; N="$(printf '\033[0m')"; else A=""; G=""; R=""; N=""; fi
# m "deutsch" "english": Text in der gewählten Sprache (Standard: Englisch)
m() { if [ "$SPRACHE" = de ]; then printf '%s' "$1"; else printf '%s' "$2"; fi; }
schritt() { printf '%s$%s %s\n' "$A" "$N" "$*"; }
info() { printf '  %s%s%s\n' "$G" "$*" "$N"; }
fehler() { printf '%s%s:%s %s\n' "$R" "$(m Fehler Error)" "$N" "$*" >&2; exit 1; }
hat() { command -v "$1" >/dev/null 2>&1; }
tty_ok() { [ -z "${SERVERWERK_NONINTERACTIVE:-}" ] && (exec </dev/tty) 2>/dev/null; }

if [ "$(id -u)" -ne 0 ]; then
  if hat sudo; then
    info "$(m 'Brauche root-Rechte, starte mit sudo neu ...' 'Root privileges needed, restarting with sudo ...')"
    if [ -f "$0" ] && [ "$0" != "sh" ] && [ "$0" != "bash" ] && [ "$0" != "-sh" ]; then exec sudo sh "$0" "$@"; fi
    fehler "$(m 'Bitte so starten:' 'Please run it like this:')  curl -fsSL $QUELLE/install.sh | sudo sh"
  fi
  fehler "$(m 'Bitte als root ausführen.' 'Please run as root.')"
fi

[ "$(uname -s)" = "Linux" ] || fehler "$(m 'Dieses Skript ist für Linux.' 'This script is for Linux.')"

# ---------- Vorhandene Installation ----------
dienst_datei() {
  for f in /etc/systemd/system/serverwerk.service /etc/init.d/serverwerk /etc/sv/serverwerk/run; do
    [ -f "$f" ] && { echo "$f"; return; }
  done
}
ALTDATEI="$(dienst_datei || true)"
alter_wert() { # alter_wert addr|lang
  [ -n "$ALTDATEI" ] || return 0
  case "$1" in
    addr) sed -n 's/.*-addr :\([0-9][0-9]*\).*/\1/p' "$ALTDATEI" | head -n1 ;;
    lang) sed -n 's/.*-lang \([a-z][a-z]\).*/\1/p' "$ALTDATEI" | head -n1 ;;
  esac
}

# ---------- Sprache ----------
if [ -z "$SPRACHE" ]; then
  SPRACHE="$(alter_wert lang)"
  if [ -z "$SPRACHE" ] && [ -z "$ALTDATEI" ] && tty_ok; then
    printf '\n  %sServer%swerk%s\n\n  Language / Sprache:  1) English   2) Deutsch\n  [1]: ' "" "$A" "$N"
    read -r W </dev/tty || W=""
    case "$W" in 2|de|DE|d|D|deutsch|Deutsch) SPRACHE=de ;; *) SPRACHE=en ;; esac
  fi
fi
case "$SPRACHE" in de) ;; *) SPRACHE=en ;; esac

# ---------- System erkennen ----------
ID=""; ID_LIKE=""; PRETTY_NAME="Linux"
[ -r /etc/os-release ] && . /etc/os-release
FAMILIE="$ID $ID_LIKE"
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) fehler "$(m "Prozessor $(uname -m) wird nicht unterstützt (nur x86_64 und arm64)." "CPU $(uname -m) is not supported (only x86_64 and arm64).")" ;;
esac
INIT=none
if hat systemctl && [ -d /run/systemd/system ]; then INIT=systemd
elif hat rc-service; then INIT=openrc
elif hat sv && [ -d /etc/sv ]; then INIT=runit
fi

printf '\n  %sServer%swerk%s  %s\n  %s%s, %s, %s%s\n\n' "" "$A" "$N" "$(m Installation Installer)" "$G" "$PRETTY_NAME" "$ARCH" "$INIT" "$N"

# ---------- Entfernen ----------
if [ "$ENTFERNEN" = 1 ]; then
  schritt "$(m 'serverwerk wird entfernt' 'Removing serverwerk')"
  case "$INIT" in
    systemd) systemctl disable --now serverwerk 2>/dev/null || true; rm -f /etc/systemd/system/serverwerk.service; systemctl daemon-reload ;;
    openrc) rc-service serverwerk stop 2>/dev/null || true; rc-update del serverwerk 2>/dev/null || true; rm -f /etc/init.d/serverwerk ;;
    runit) rm -f /var/service/serverwerk; rm -rf /etc/sv/serverwerk ;;
    *) pkill -f "$BIN" 2>/dev/null || true ;;
  esac
  if hat docker; then
    ids="$(docker ps -aq --filter label=serverwerk.role 2>/dev/null || true)"
    [ -n "$ids" ] && docker rm -f $ids >/dev/null 2>&1 || true
  fi
  rm -f "$BIN"
  info "$(m "Programm entfernt. Deine Server-Dateien liegen weiter in $DATEN" "Program removed. Your server files are still in $DATEN")"
  info "$(m 'Ganz löschen:' 'Delete completely:')  rm -rf $DATEN"
  exit 0
fi

# ---------- Port für die Weboberfläche ----------
port_frei() { # port_frei PORT -> 0, wenn niemand darauf lauscht
  p="$1"
  if hat ss; then
    ! ss -Htln 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]$p\$"
  elif hat netstat; then
    ! netstat -tln 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]$p\$"
  else
    hex="$(printf '%04X' "$p")"
    ! cat /proc/net/tcp /proc/net/tcp6 2>/dev/null | awk '$4=="0A"{print $2}' | grep -qi ":$hex\$"
  fi
}
gueltig() { case "$1" in ''|*[!0-9]*) return 1 ;; esac; [ "$1" -ge 1 ] && [ "$1" -le 65535 ]; }

if [ -z "$PORT" ]; then
  ALTPORT="$(alter_wert addr)"
  if [ -n "$ALTPORT" ]; then
    PORT="$ALTPORT"
    info "$(m "Weboberfläche bleibt auf Port $PORT" "Web interface stays on port $PORT")"
  else
    FREI=""
    for p in 8200 8080 8000 8888 9000 3000 8100 8300 8500 8800 10000; do
      port_frei "$p" && FREI="$FREI $p"
    done
    FREI="$(echo $FREI)"
    VORSCHLAG="${FREI%% *}"
    [ -z "$VORSCHLAG" ] && VORSCHLAG=8200
    # Auch bei "curl ... | sudo sh" fragen: direkt vom Terminal lesen.
    if tty_ok; then
      printf '  %s %s%s%s\n' "$(m 'Freie Ports für die Weboberfläche:' 'Free ports for the web interface:')" "$A" "${FREI:-$(m 'keiner der üblichen' 'none of the usual ones')}" "$N"
      n=0
      while :; do
        printf '  %s [%s]: ' "$(m 'Welchen Port soll das Panel nutzen?' 'Which port should the panel use?')" "$VORSCHLAG"
        read -r ANTWORT </dev/tty || ANTWORT=""
        ANTWORT="${ANTWORT:-$VORSCHLAG}"
        if ! gueltig "$ANTWORT"; then
          printf '  %s%s%s\n' "$R" "$(m 'Bitte eine Zahl von 1 bis 65535.' 'Please enter a number from 1 to 65535.')" "$N"
        elif ! port_frei "$ANTWORT"; then
          printf '  %s%s%s\n' "$R" "$(m "Port $ANTWORT ist schon belegt." "Port $ANTWORT is already in use.")" "$N"
        else
          PORT="$ANTWORT"; break
        fi
        n=$((n+1)); [ $n -ge 5 ] && fehler "$(m 'Kein gültiger Port gewählt.' 'No valid port chosen.')"
      done
      printf '\n'
    else
      PORT="$VORSCHLAG"
      info "$(m "Port $PORT gewählt (ändern: sh -s -- --port 9000)" "Port $PORT chosen (change: sh -s -- --port 9000)")"
    fi
  fi
fi
gueltig "$PORT" || fehler "$(m 'Ungültiger Port:' 'Invalid port:') $PORT"

# ---------- Pakete ----------
paket() {
  if hat apt-get; then DEBIAN_FRONTEND=noninteractive apt-get install -y -q "$@" >/dev/null
  elif hat dnf; then dnf install -y -q "$@" >/dev/null
  elif hat yum; then yum install -y -q "$@" >/dev/null
  elif hat pacman; then pacman -Sy --noconfirm --needed "$@" >/dev/null
  elif hat zypper; then zypper -n -q install "$@" >/dev/null
  elif hat apk; then apk add -q "$@" >/dev/null
  elif hat xbps-install; then xbps-install -Sy "$@" >/dev/null
  else return 1; fi
}

if ! hat curl && ! hat wget; then
  schritt "$(m 'curl wird installiert' 'Installing curl')"
  hat apt-get && apt-get update -q >/dev/null
  paket curl || fehler "$(m 'Bitte curl installieren.' 'Please install curl.')"
fi
laden() { # laden URL ZIEL
  if hat curl; then curl -fsSL --retry 3 -o "$2" "$1"; else wget -q -O "$2" "$1"; fi
}

# ---------- Docker ----------
docker_starten() {
  case "$INIT" in
    systemd) systemctl enable --now docker >/dev/null 2>&1 || systemctl start docker ;;
    openrc) rc-update add docker default >/dev/null 2>&1 || true; rc-service docker start >/dev/null 2>&1 || true ;;
    runit) [ -e /var/service/docker ] || ln -s /etc/sv/docker /var/service/ ;;
    *) if ! docker info >/dev/null 2>&1; then
         info "$(m 'Kein Dienst-Manager gefunden, starte dockerd im Hintergrund' 'No service manager found, starting dockerd in the background')"
         nohup dockerd >/var/log/dockerd.log 2>&1 &
       fi ;;
  esac
  i=0
  while ! docker info >/dev/null 2>&1; do
    i=$((i+1)); [ $i -gt 30 ] && fehler "$(m 'Docker startet nicht. Ausgabe:' 'Docker does not start. Output:') journalctl -u docker"
    sleep 1
  done
}

if hat docker && docker info >/dev/null 2>&1; then
  schritt "$(m 'Docker ist da' 'Docker found') ($(docker version --format '{{.Server.Version}}' 2>/dev/null))"
else
  if ! hat docker; then
    schritt "$(m 'Docker wird installiert (dauert ein paar Minuten)' 'Installing Docker (takes a few minutes)')"
    case "$FAMILIE" in
      *arch*|*manjaro*|*endeavouros*) paket docker ;;
      *alpine*) paket docker ;;
      *suse*) paket docker ;;
      *void*) paket docker ;;
      *rocky*|*almalinux*|*ol*|*centos*|*rhel*)
        if hat dnf; then
          dnf -y -q install dnf-plugins-core >/dev/null 2>&1 || true
          dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo >/dev/null 2>&1 ||
            dnf config-manager addrepo --from-repofile=https://download.docker.com/linux/centos/docker-ce.repo >/dev/null 2>&1 || true
          dnf -y -q install docker-ce docker-ce-cli containerd.io >/dev/null
        else
          laden https://get.docker.com /tmp/get-docker.sh && sh /tmp/get-docker.sh >/dev/null
        fi ;;
      *)
        # Debian, Ubuntu, Mint, Raspberry Pi OS, Fedora, SLES: offizielles Skript von Docker.
        laden https://get.docker.com /tmp/get-docker.sh || fehler "$(m 'get.docker.com nicht erreichbar' 'get.docker.com not reachable')"
        sh /tmp/get-docker.sh >/tmp/get-docker.log 2>&1 || {
          info "$(m 'Offizielles Skript ging nicht (siehe /tmp/get-docker.log), versuche Paketquelle der Distro' 'Official script failed (see /tmp/get-docker.log), trying the distribution package')"
          hat apt-get && apt-get update -q >/dev/null
          paket docker.io || paket docker || fehler "$(m 'Docker konnte nicht installiert werden.' 'Docker could not be installed.')"
        } ;;
    esac
    rm -f /tmp/get-docker.sh
  fi
  schritt "$(m 'Docker wird gestartet' 'Starting Docker')"
  docker_starten
fi

# ---------- Programm ----------
schritt "$(m 'serverwerk wird geladen' 'Downloading serverwerk')"
TMP="$(mktemp)"
LOKAL="$(dirname "$0" 2>/dev/null)/serverwerk-linux-$ARCH"
if [ -n "${SERVERWERK_DATEI:-}" ]; then cp "$SERVERWERK_DATEI" "$TMP"
elif [ -f "$LOKAL" ]; then cp "$LOKAL" "$TMP"; info "$(m nehme using) $LOKAL"
else laden "$QUELLE/serverwerk-linux-$ARCH" "$TMP" || fehler "$(m "Download von $QUELLE fehlgeschlagen" "Download from $QUELLE failed")"
fi
chmod 755 "$TMP"
"$TMP" version >/dev/null 2>&1 || fehler "$(m 'Geladene Datei ist kein gültiges Programm.' 'Downloaded file is not a valid program.')"
NEU="$("$TMP" version)"
ALT=""; [ -x "$BIN" ] && ALT="$("$BIN" version 2>/dev/null || true)"

# Server-Prozesse in den Containern laufen als UID 988 (wie bei Pterodactyl).
if ! getent passwd 988 >/dev/null 2>&1; then
  if hat useradd; then useradd --system --no-create-home --uid 988 --shell /usr/sbin/nologin serverwerk 2>/dev/null || true
  elif hat adduser; then adduser -S -D -H -u 988 -s /sbin/nologin serverwerk 2>/dev/null || true; fi
fi

mkdir -p "$DATEN"
dienst_stopp() {
  case "$INIT" in
    systemd) systemctl stop serverwerk 2>/dev/null || true ;;
    openrc) rc-service serverwerk stop 2>/dev/null || true ;;
    runit) sv stop serverwerk 2>/dev/null || true ;;
    *) pkill -f "^$BIN" 2>/dev/null || true ;;
  esac
}
[ -n "$ALT" ] && dienst_stopp
mv -f "$TMP" "$BIN"

# ---------- Dienst ----------
ARGS="-addr :$PORT -daten $DATEN -lang $SPRACHE"
schritt "$(m "Dienst wird eingerichtet (Port $PORT)" "Setting up the service (port $PORT)")"
case "$INIT" in
  systemd)
    cat >/etc/systemd/system/serverwerk.service <<EOF
[Unit]
Description=Serverwerk Panel
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
ExecStart=$BIN $ARGS
Restart=always
RestartSec=3
LimitNOFILE=65536
# Beim Herunterfahren stoppt Serverwerk alle Spielserver sauber (bis zu 3 Minuten).
TimeoutStopSec=240
KillMode=mixed

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable serverwerk >/dev/null 2>&1
    systemctl restart serverwerk ;;
  openrc)
    cat >/etc/init.d/serverwerk <<EOF
#!/sbin/openrc-run
description="Serverwerk Panel"
command="$BIN"
command_args="$ARGS"
command_background=true
pidfile="/run/serverwerk.pid"
output_log="/var/log/serverwerk.log"
error_log="/var/log/serverwerk.log"
depend() { need docker net; }
retry="TERM/240/KILL/5"
EOF
    chmod 755 /etc/init.d/serverwerk
    rc-update add serverwerk default >/dev/null 2>&1
    rc-service serverwerk restart >/dev/null 2>&1 || rc-service serverwerk start ;;
  runit)
    mkdir -p /etc/sv/serverwerk
    printf '#!/bin/sh\nexec %s %s 2>&1\n' "$BIN" "$ARGS" >/etc/sv/serverwerk/run
    chmod 755 /etc/sv/serverwerk/run
    [ -e /var/service/serverwerk ] || ln -s /etc/sv/serverwerk /var/service/
    sv restart serverwerk 2>/dev/null || true ;;
  *)
    info "$(m 'Kein Dienst-Manager: starte im Hintergrund (nach Neustart des Rechners erneut ausführen)' 'No service manager: starting in the background (run again after a reboot)')"
    nohup "$BIN" $ARGS >/var/log/serverwerk.log 2>&1 & ;;
esac

# Firewall für die Weboberfläche öffnen, falls eine aktiv ist.
if hat ufw && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "$PORT/tcp" >/dev/null 2>&1 && info "ufw: $(m "Port $PORT freigegeben" "port $PORT opened")"
elif hat firewall-cmd && firewall-cmd --state >/dev/null 2>&1; then
  firewall-cmd --permanent --add-port="$PORT/tcp" >/dev/null 2>&1 && firewall-cmd --reload >/dev/null 2>&1 && info "firewalld: $(m "Port $PORT freigegeben" "port $PORT opened")"
fi

# ---------- Fertig ----------
i=0
until (hat curl && curl -fs -o /dev/null "http://127.0.0.1:$PORT/") || (! hat curl && wget -q -O /dev/null "http://127.0.0.1:$PORT/"); do
  i=$((i+1)); [ $i -gt 20 ] && fehler "$(m 'Panel antwortet nicht. Log:' 'Panel does not respond. Log:') journalctl -u serverwerk -n 50"
  sleep 1
done

IP="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')"
[ -z "$IP" ] && IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -z "$IP" ] && IP="localhost"

printf '\n  %s%s%s %s\n\n' "$A" "$(m 'Fertig.' 'Done.')" "$N" "$(m "Serverwerk $NEU läuft." "Serverwerk $NEU is running.")"
printf '  %-10s %shttp://%s:%s%s\n' "$(m 'Adresse:' 'Address:')" "$A" "$IP" "$PORT" "$N"
if [ -f "$DATEN/einrichtung.txt" ]; then
  CODE="$(tr -d ' \r\n' < "$DATEN/einrichtung.txt")"
  printf '\n  %s\n' "$(m 'Jetzt im Browser öffnen und dein Admin-Konto anlegen:' 'Now open this in your browser and create your admin account:')"
  printf '  %shttp://%s:%s/#/einrichten/%s%s\n' "$A" "$IP" "$PORT" "$CODE" "$N"
  printf '  %s(%s)%s\n' "$G" "$(m "Einrichtungs-Code: $CODE, gilt nur bis das erste Konto angelegt ist" "Setup code: $CODE, only valid until the first account is created")" "$N"
elif [ -n "$ALT" ]; then
  printf '  %-10s %s -> %s, %s\n' "Update:" "$ALT" "$NEU" "$(m 'Server und Daten bleiben erhalten' 'servers and data are kept')"
fi
printf '\n  %s%s  sudo serverwerk %s admin%s\n\n' "$G" "$(m 'Passwort vergessen:' 'Forgot password:')" "$(m passwort password)" "$N"
