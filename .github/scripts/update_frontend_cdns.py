#!/usr/bin/env python3
"""Bump the pinned front-end CDN versions in layout.tmpl and recompute SRI.

Writes a human-readable summary of any changes to /tmp/frontend-cdn-summary.txt.
"""
import base64
import hashlib
import json
import os
import re
import sys
import urllib.request

TMPL = os.environ.get("LAYOUT_TMPL", "cmd/server/static/layout.tmpl")
SUMMARY = os.environ.get("SUMMARY_FILE", "/tmp/frontend-cdn-summary.txt")

# name: npm package; marker: unique text preceding the version in the template;
# sri_url: how to fetch the file to hash (None means no SRI, version-only).
DEPS = [
    {"name": "daisyui", "marker": "daisyui@",
     "sri_url": lambda v: f"https://cdn.jsdelivr.net/npm/daisyui@{v}"},
    {"name": "@tailwindcss/browser", "marker": "@tailwindcss/browser@",
     "sri_url": lambda v: f"https://cdn.jsdelivr.net/npm/@tailwindcss/browser@{v}"},
    {"name": "web-vitals", "marker": "web-vitals@", "sri_url": None},
]

VERSION_RE = r"[0-9][0-9A-Za-z.+-]*"


def fetch(url):
    with urllib.request.urlopen(url) as r:  # noqa: S310 (trusted CDN/registry)
        return r.read()


def latest(pkg):
    return json.loads(fetch(f"https://registry.npmjs.org/{pkg}/latest"))["version"]


def sri(url):
    return "sha384-" + base64.b64encode(hashlib.sha384(fetch(url)).digest()).decode()


def main():
    src = open(TMPL).read()
    changes = []

    for dep in DEPS:
        m = re.search(re.escape(dep["marker"]) + f"({VERSION_RE})", src)
        if not m:
            sys.exit(f"marker not found in template: {dep['marker']}")
        cur = m.group(1)
        new = latest(dep["name"])
        if not re.fullmatch(VERSION_RE, new):
            sys.exit(f"unexpected version for {dep['name']}: {new!r}")
        if new == cur:
            continue

        src = src.replace(f"{dep['marker']}{cur}", f"{dep['marker']}{new}")
        if dep["sri_url"]:
            new_hash = sri(dep["sri_url"](new))
            idx = src.index(f"{dep['marker']}{new}")
            head, tail = src[:idx], src[idx:]
            tail = re.sub(r'integrity="sha384-[A-Za-z0-9+/=]*"',
                          f'integrity="{new_hash}"', tail, count=1)
            src = head + tail
        changes.append(f"- {dep['name']}: {cur} -> {new}")

    with open(SUMMARY, "w") as f:
        f.write("\n".join(changes) + ("\n" if changes else ""))

    if changes:
        open(TMPL, "w").write(src)
    print("\n".join(changes) if changes else "up to date")


if __name__ == "__main__":
    main()
