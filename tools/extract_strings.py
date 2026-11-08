"""Sammelt alle deutschen Texte aus Oberfläche (app.js) und Server-Code (*.go),
damit das Wörterbuch panel/web/i18n/en.json vollständig bleibt.

Aufruf:  python tools/extract_strings.py          -> listet Texte ohne Übersetzung
         python tools/extract_strings.py --alle   -> listet alle gefundenen Texte
"""
import json, os, re, sys

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'panel')
LETTERS = re.compile(r'[A-Za-zÄÖÜäöüß]{2}')
GERMANISH = re.compile(r'[äöüÄÖÜß]|\b[A-ZÄÖÜ][a-zäöüß]{2,}\b|\b(?:und|oder|nicht|ist|ein|eine|mit|für|auf|noch|wird|kein|bitte|alle|aus|an|zu|von|bei|nach|vor|schon|nur|gibt|hier|jetzt|frei|belegt|neu|leer)\b')


def js_pieces(src):
    """Liefert Textstücke aus String- und Template-Literalen (ohne ${...})."""
    i, n = 0, len(src)
    pieces = []

    def read_template(i):
        buf = ''
        while i < n:
            c = src[i]
            if c == '\\':
                buf += src[i:i + 2]; i += 2; continue
            if c == '`':
                pieces.append(buf); return i + 1
            if c == '$' and i + 1 < n and src[i + 1] == '{':
                pieces.append(buf); buf = ''
                i = read_code(i + 2, '}')
                continue
            buf += c; i += 1
        pieces.append(buf)
        return i

    def read_code(i, end=None):
        depth = 0
        while i < n:
            c = src[i]
            if c in '\'"':
                q = c; j = i + 1; buf = ''
                while j < n and src[j] != q and src[j] != '\n':
                    if src[j] == '\\':
                        buf += src[j:j + 2]; j += 2; continue
                    buf += src[j]; j += 1
                pieces.append(buf); i = j + 1; continue
            if c == '`':
                i = read_template(i + 1); continue
            if c == '/' and i + 1 < n and src[i + 1] == '/':
                j = src.find('\n', i); i = n if j < 0 else j; continue
            if c == '/' and i + 1 < n and src[i + 1] == '*':
                j = src.find('*/', i + 2); i = n if j < 0 else j + 2; continue
            if c == '/':  # Regex-Literal grob überspringen
                prev = src[:i].rstrip()[-1:] if src[:i].rstrip() else ''
                if prev in '(,=:[!&|?;{}' or src[:i].rstrip().endswith(('return', 'typeof')):
                    j = i + 1; cls = False
                    while j < n and src[j] != '\n':
                        if src[j] == '\\': j += 2; continue
                        if src[j] == '[': cls = True
                        elif src[j] == ']': cls = False
                        elif src[j] == '/' and not cls: break
                        j += 1
                    i = j + 1; continue
            if end and c == '{':
                depth += 1
            elif end and c == '}':
                if depth == 0:
                    return i + 1
                depth -= 1
            i += 1
        return i

    read_code(0)
    return pieces


def clean(t):
    t = t.replace('&nbsp;', ' ').replace('&rsaquo;', '›').replace('&amp;', '&').replace('&lt;', '<').replace('&gt;', '>')
    return t.strip(' \n\t·,()')


def fragments_js(src):
    out = set()
    for piece in js_pieces(src):
        for part in re.split(r'<[^<>]*>|<[^<>]*$|^[^<>]*>', piece):
            t = clean(part)
            if len(t) < 2 or not LETTERS.search(t) or not GERMANISH.search(t):
                continue
            if re.search(r'[{};]|=>|\bvar\(|^\.|^#|^[a-z-]+$|^[a-z]+[A-Z]\w*$', t) and not re.search(r'[äöüß]| [a-zäöü]', t):
                continue
            out.add(t)
    return out


def fragments_go(src):
    out = set()
    for m in re.finditer(r'"((?:[^"\\\n]|\\.)*)"', src):
        s = m.group(1)
        if not GERMANISH.search(s) or not re.search(r' |[äöüß]', s):
            continue
        for part in re.split(r'%[-+# 0-9.]*[vsdqfx%]|\\n', s):
            t = part.strip(' :.,()')
            if len(t) >= 3 and GERMANISH.search(t) and not re.search(r'^[\w./-]+$', t):
                out.add(t)
    return out


def collect():
    frags = set(fragments_js(open(os.path.join(ROOT, 'web', 'app.js'), encoding='utf-8').read()))
    for f in os.listdir(ROOT):
        if f.endswith('.go') and not f.endswith('_test.go'):
            frags |= fragments_go(open(os.path.join(ROOT, f), encoding='utf-8').read())
    return frags


if __name__ == '__main__':
    frags = sorted(collect(), key=lambda x: (-len(x), x))
    dict_path = os.path.join(ROOT, 'web', 'i18n', 'en.json')
    have = json.load(open(dict_path, encoding='utf-8')) if os.path.exists(dict_path) else {}
    show = frags if '--alle' in sys.argv else [f for f in frags if f not in have]
    for f in show:
        print(f)
    print(f'--- {len(show)} von {len(frags)}', file=sys.stderr)
