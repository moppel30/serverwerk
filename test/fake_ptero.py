"""Schein-Pterodactyl für den Import-Test (nur lokal benutzen).
Zwei Server: "Survival" (Paper, Egg wird per Application-API geliefert)
und "Lobby" (Egg-Name ohne Treffer). Dateien kommen als tar.gz-Archiv.
Start: python3 fake_ptero.py 8765
"""
import io, json, sys, tarfile, urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8765
BIND = sys.argv[2] if len(sys.argv) > 2 else "127.0.0.1"
CK, AK = "ptlc_test", "ptla_test"

def server(ident, uuid, name, egg, ports, mem):
    return {"attributes": {"identifier": ident, "uuid": uuid, "name": name, "description": "aus Pterodactyl",
        "docker_image": "ghcr.io/pelican-eggs/yolks:java_21", "limits": {"memory": mem, "cpu": 200, "disk": 10240},
        "relationships": {
            "allocations": {"data": [{"attributes": {"port": p, "is_default": i == 0}} for i, p in enumerate(ports)]},
            "egg": {"attributes": {"uuid": "egg-" + ident, "name": egg}}}}}

SERVERS = [server("aaaa1111", "11111111-aaaa-4aaa-8aaa-111111111111", "Survival", "Paper", [25566], 3072),
           server("bbbb2222", "22222222-bbbb-4bbb-8bbb-222222222222", "Lobby", "Mein Spezial-Egg", [25999], 1024)]

def archive():
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w:gz") as t:
        for name, data in [("server.properties", b"motd=Aus Pterodactyl\nserver-port=25566\n"), ("world/level.dat", b"x" * 1000), ("eula.txt", b"eula=true\n")]:
            ti = tarfile.TarInfo(name); ti.size = len(data); t.addfile(ti, io.BytesIO(data))
    return buf.getvalue()

class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def out(self, code, obj=None, raw=None, ctype="application/json"):
        b = raw if raw is not None else json.dumps(obj).encode()
        self.send_response(code); self.send_header("Content-Type", ctype); self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def auth(self, key):
        return self.headers.get("Authorization") == "Bearer " + key
    def route(self, method):
        u = urllib.parse.urlparse(self.path); q = urllib.parse.parse_qs(u.query); p = u.path
        if p == "/dl/archive.tar.gz":
            return self.out(200, raw=archive(), ctype="application/gzip")
        if p.startswith("/api/application/"):
            if not self.auth(AK): return self.out(403, {"errors": [{"detail": "nope"}]})
            if p == "/api/application/servers":
                return self.out(200, {"data": [{"attributes": {"egg": 5, "nest": 1}}]} if "11111111" in q.get("filter[uuid]", [""])[0] else {"data": []})
            if p == "/api/application/nests/1/eggs/5":
                return self.out(200, {"attributes": {"name": "Paper (aus Ptero)", "author": "test", "description": "Paper exakt",
                    "docker_images": {"Java 21": "ghcr.io/pelican-eggs/yolks:java_21"}, "startup": "java -jar {{SERVER_JARFILE}}",
                    "config": {"files": {"server.properties": {"parser": "properties", "find": {"server-port": "{{server.build.default.port}}"}}}, "startup": {"done": ")! For help"}, "stop": "stop", "logs": {}},
                    "script": {"install": "echo ok", "entry": "ash", "container": "alpine"},
                    "relationships": {"variables": {"data": [{"attributes": {"name": "Jar", "description": "", "env_variable": "SERVER_JARFILE", "default_value": "server.jar", "user_viewable": True, "user_editable": True, "rules": "required|string"}}]}}}})
            return self.out(404, {"errors": [{"detail": "nicht gefunden"}]})
        if not self.auth(CK): return self.out(401, {"errors": [{"detail": "Unauthenticated."}]})
        if p == "/api/client":
            return self.out(200, {"data": SERVERS, "meta": {"pagination": {"current_page": 1, "total_pages": 1}}})
        parts = p.split("/")  # /api/client/servers/<id>/...
        if len(parts) >= 5 and parts[3] == "servers":
            sub = "/".join(parts[5:])
            if sub == "startup":
                return self.out(200, {"data": [{"attributes": {"env_variable": "SERVER_JARFILE", "server_value": "server.jar"}}], "meta": {"raw_startup_command": "java -Xms128M -jar {{SERVER_JARFILE}}"}})
            if sub == "resources":
                return self.out(200, {"attributes": {"current_state": "offline", "resources": {"disk_bytes": 123456789}}})
            if sub == "files/list":
                return self.out(200, {"data": [{"attributes": {"name": n}} for n in ["server.properties", "world", "eula.txt"]]})
            if sub == "files/compress":
                return self.out(200, {"attributes": {"name": "archive-test.tar.gz"}})
            if sub == "files/download":
                return self.out(200, {"attributes": {"url": f"http://127.0.0.1:{PORT}/dl/archive.tar.gz"}})
            if sub == "files/delete":
                return self.out(204, raw=b"")
        self.out(404, {"errors": [{"detail": "nicht gefunden"}]})
    def do_GET(self): self.route("GET")
    def do_POST(self):
        n = int(self.headers.get("Content-Length") or 0); self.rfile.read(n); self.route("POST")

ThreadingHTTPServer((BIND, PORT), H).serve_forever()
