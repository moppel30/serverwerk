#!/bin/sh
# Serverwerk – Installation mit einem Befehl:
#   curl -fsSL https://github.com/moppel30/serverwerk/releases/latest/download/install.sh | sudo sh
#
# Optionen (hinter "sh -s --"):  --port 8200   --entfernen
# Läuft auf Debian, Ubuntu, Raspberry Pi OS, Mint, Fedora, RHEL/Rocky/Alma,
# CentOS, Arch/Manjaro, openSUSE, Alpine und Void. Erneut ausführen = Update.
set -eu

QUELLE="${SERVERWERK_URL:-https://github.com/moppel30/serverwerk/releases/latest/download}"
PORT="${SERVERWERK_PORT:-}"
DATEN="/var/lib/serverwerk"
BIN="/usr/local/bin/serverwerk"
ENTFERNEN=0

while [ $# -gt 0 ]; do
  case "$1" in
    --port) PORT="$2"; shift 2 ;;
    --port=*) PORT="${1#*=}"; shift ;;
    --entfernen|--uninstall) ENTFERNEN=1; shift ;;
    --quelle) QUELLE="$2"; shift 2 ;;
    *) echo "Unbekannte Option: $1"; exit 2 ;;
  esac
done

if [ -t 1 ]; then A="$(printf '\033[38;2;217;199;167m')"; G="$(printf '\033[2m')"; R="$(printf '\033[31m')"; N="$(printf '\033[0m')"; else A=""; G=""; R=""; N=""; fi
schritt() { printf '%s$%s %s\n' "$A" "$N" "$*"; }
info() { printf '  %s%s%s\n' "$G" "$*" "$N"; }
fehler() { printf '%sFehler:%s %s\n' "$R" "$N" "$*" >&2; exit 1; }
hat() { command -v "$1" >/dev/null 2>&1; }

if [ "$(id -u)" -ne 0 ]; then
  if hat sudo; then
    info "Brauche root-Rechte, starte mit sudo neu ..."
    if [ -f "$0" ] && [ "$0" != "sh" ] && [ "$0" != "bash" ] && [ "$0" != "-sh" ]; then exec sudo sh "$0" "$@"; fi
    fehler "Bitte so starten:  curl -fsSL $QUELLE/install.sh | sudo sh"
  fi
  fehler "Bitte als root ausführen."
fi

[ "$(uname -s)" = "Linux" ] || fehler "Dieses Skript ist für Linux. Für Windows kommt später eine eigene .exe."

# ---------- System erkennen ----------
ID=""; ID_LIKE=""; PRETTY_NAME="Linux"
[ -r /etc/os-release ] && . /etc/os-release
FAMILIE="$ID $ID_LIKE"
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) fehler "Prozessor $(uname -m) wird nicht unterstützt (nur x86_64 und arm64)." ;;
esac
INIT=none
if hat systemctl && [ -d /run/systemd/system ]; then INIT=systemd
elif hat rc-service; then INIT=openrc
elif hat sv && [ -d /etc/sv ]; then INIT=runit
fi

printf '\n  %sServer%swerk%s  Installation\n  %s%s, %s, %s%s\n\n' "" "$A" "$N" "$G" "$PRETTY_NAME" "$ARCH" "$INIT" "$N"

# ---------- Entfernen ----------
if [ "$ENTFERNEN" = 1 ]; then
  schritt "serverwerk wird entfernt"
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
  info "Programm entfernt. Deine Server-Dateien liegen weiter in $DATEN"
  info "Ganz löschen:  rm -rf $DATEN"
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
alter_port() { # Port einer vorhandenen Installation
  for f in /etc/systemd/system/serverwerk.service /etc/init.d/serverwerk /etc/sv/serverwerk/run; do
    [ -f "$f" ] && sed -n 's/.*-addr :\([0-9][0-9]*\).*/\1/p' "$f" | head -n1 && return
  done
}
gueltig() { case "$1" in ''|*[!0-9]*) return 1 ;; esac; [ "$1" -ge 1 ] && [ "$1" -le 65535 ]; }

if [ -z "$PORT" ]; then
  ALTPORT="$(alter_port || true)"
  if [ -n "$ALTPORT" ]; then
    PORT="$ALTPORT"
    info "Weboberfläche bleibt auf Port $PORT"
  else
    FREI=""
    for p in 8200 8080 8000 8888 9000 3000 8100 8300 8500 8800 10000; do
      port_frei "$p" && FREI="$FREI $p"
    done
    FREI="$(echo $FREI)"
    VORSCHLAG="${FREI%% *}"
    [ -z "$VORSCHLAG" ] && VORSCHLAG=8200
    # Auch bei "curl ... | sudo sh" fragen: direkt vom Terminal lesen.
    if [ -z "${SERVERWERK_NONINTERACTIVE:-}" ] && (exec </dev/tty) 2>/dev/null; then
      printf '  Freie Ports für die Weboberfläche: %s%s%s\n' "$A" "${FREI:-keiner der üblichen}" "$N"
      n=0
      while :; do
        printf '  Welchen Port soll das Panel nutzen? [%s]: ' "$VORSCHLAG"
        read -r ANTWORT </dev/tty || ANTWORT=""
        ANTWORT="${ANTWORT:-$VORSCHLAG}"
        if ! gueltig "$ANTWORT"; then
          printf '  %sBitte eine Zahl von 1 bis 65535.%s\n' "$R" "$N"
        elif ! port_frei "$ANTWORT"; then
          printf '  %sPort %s ist schon belegt.%s\n' "$R" "$ANTWORT" "$N"
        else
          PORT="$ANTWORT"; break
        fi
        n=$((n+1)); [ $n -ge 5 ] && fehler "Kein gültiger Port gewählt."
      done
      printf '\n'
    else
      PORT="$VORSCHLAG"
      info "Port $PORT gewählt (ändern: sh -s -- --port 9000)"
    fi
  fi
fi
gueltig "$PORT" || fehler "Ungültiger Port: $PORT"

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
  schritt "curl wird installiert"
  hat apt-get && apt-get update -q >/dev/null
  paket curl || fehler "Bitte curl installieren."
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
         info "Kein Dienst-Manager gefunden, starte dockerd im Hintergrund"
         nohup dockerd >/var/log/dockerd.log 2>&1 &
       fi ;;
  esac
  i=0
  while ! docker info >/dev/null 2>&1; do
    i=$((i+1)); [ $i -gt 30 ] && fehler "Docker startet nicht. Ausgabe: journalctl -u docker"
    sleep 1
  done
}

if hat docker && docker info >/dev/null 2>&1; then
  schritt "Docker ist da ($(docker version --format '{{.Server.Version}}' 2>/dev/null))"
else
  if ! hat docker; then
    schritt "Docker wird installiert (dauert ein paar Minuten)"
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
        laden https://get.docker.com /tmp/get-docker.sh || fehler "get.docker.com nicht erreichbar"
        sh /tmp/get-docker.sh >/tmp/get-docker.log 2>&1 || {
          info "Offizielles Skript ging nicht (siehe /tmp/get-docker.log), versuche Paketquelle der Distro"
          hat apt-get && apt-get update -q >/dev/null
          paket docker.io || paket docker || fehler "Docker konnte nicht installiert werden."
        } ;;
    esac
    rm -f /tmp/get-docker.sh
  fi
  schritt "Docker wird gestartet"
  docker_starten
fi

# ---------- Programm ----------
schritt "serverwerk wird geladen"
TMP="$(mktemp)"
LOKAL="$(dirname "$0" 2>/dev/null)/serverwerk-linux-$ARCH"
if [ -n "${SERVERWERK_DATEI:-}" ]; then cp "$SERVERWERK_DATEI" "$TMP"
elif [ -f "$LOKAL" ]; then cp "$LOKAL" "$TMP"; info "nehme $LOKAL"
else laden "$QUELLE/serverwerk-linux-$ARCH" "$TMP" || fehler "Download von $QUELLE fehlgeschlagen"
fi
chmod 755 "$TMP"
"$TMP" version >/dev/null 2>&1 || fehler "Geladene Datei ist kein gültiges Programm."
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
schritt "Dienst wird eingerichtet (Port $PORT)"
case "$INIT" in
  systemd)
    cat >/etc/systemd/system/serverwerk.service <<EOF
[Unit]
Description=Serverwerk Panel
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
ExecStart=$BIN -addr :$PORT -daten $DATEN
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
command_args="-addr :$PORT -daten $DATEN"
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
    printf '#!/bin/sh\nexec %s -addr :%s -daten %s 2>&1\n' "$BIN" "$PORT" "$DATEN" >/etc/sv/serverwerk/run
    chmod 755 /etc/sv/serverwerk/run
    [ -e /var/service/serverwerk ] || ln -s /etc/sv/serverwerk /var/service/
    sv restart serverwerk 2>/dev/null || true ;;
  *)
    info "Kein Dienst-Manager: starte im Hintergrund (nach Neustart des Rechners erneut ausführen)"
    nohup "$BIN" -addr ":$PORT" -daten "$DATEN" >/var/log/serverwerk.log 2>&1 & ;;
esac

# Firewall für die Weboberfläche öffnen, falls eine aktiv ist.
if hat ufw && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "$PORT/tcp" >/dev/null 2>&1 && info "ufw: Port $PORT freigegeben"
elif hat firewall-cmd && firewall-cmd --state >/dev/null 2>&1; then
  firewall-cmd --permanent --add-port="$PORT/tcp" >/dev/null 2>&1 && firewall-cmd --reload >/dev/null 2>&1 && info "firewalld: Port $PORT freigegeben"
fi

# ---------- Fertig ----------
i=0
until (hat curl && curl -fs -o /dev/null "http://127.0.0.1:$PORT/") || (! hat curl && wget -q -O /dev/null "http://127.0.0.1:$PORT/"); do
  i=$((i+1)); [ $i -gt 20 ] && fehler "Panel antwortet nicht. Log: journalctl -u serverwerk -n 50"
  sleep 1
done

IP="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')"
[ -z "$IP" ] && IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -z "$IP" ] && IP="localhost"

printf '\n  %sFertig.%s Serverwerk %s läuft.\n\n' "$A" "$N" "$NEU"
printf '  Adresse:   %shttp://%s:%s%s\n' "$A" "$IP" "$PORT" "$N"
if [ -f "$DATEN/einrichtung.txt" ]; then
  CODE="$(tr -d ' \r\n' < "$DATEN/einrichtung.txt")"
  printf '\n  Jetzt im Browser öffnen und dein Admin-Konto anlegen:\n'
  printf '  %shttp://%s:%s/#/einrichten/%s%s\n' "$A" "$IP" "$PORT" "$CODE" "$N"
  printf '  %s(Einrichtungs-Code: %s, gilt nur bis das erste Konto angelegt ist)%s\n' "$G" "$CODE" "$N"
elif [ -n "$ALT" ]; then
  printf '  Update:    %s -> %s, Server und Daten bleiben erhalten\n' "$ALT" "$NEU"
fi
printf '\n  %sPasswort vergessen:  sudo serverwerk passwort admin%s\n\n' "$G" "$N"
