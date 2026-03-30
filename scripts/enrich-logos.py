#!/usr/bin/env python3
import csv
import glob
import json
import sys
import urllib.request

LOGOS_URL = "https://raw.githubusercontent.com/iptv-org/database/master/data/logos.csv"

print("Fetching logos from iptv-org/database...")
with urllib.request.urlopen(LOGOS_URL) as r:
    content = r.read().decode("utf-8")

logos = {}
for row in csv.DictReader(content.splitlines()):
    ch, url = row["channel"].strip(), row["url"].strip()
    if ch and url and ch not in logos:
        logos[ch] = url

print(f"Loaded {len(logos)} logos")

n_files, n_ch = 0, 0
for path in sorted(glob.glob("plugins/*/channels.json")):
    with open(path) as f:
        chs = json.load(f)
    changed = False
    for ch in chs:
        xid = ch.get("xmltv_id", "")
        if not xid or ch.get("logo"):
            continue
        logo = logos.get(xid) or logos.get(xid.split("@")[0])
        if logo:
            ch["logo"] = logo
            n_ch += 1
            changed = True
    if changed:
        with open(path, "w") as f:
            json.dump(chs, f, indent=2, ensure_ascii=False)
            f.write("\n")
        n_files += 1

print(f"Updated {n_ch} channels across {n_files} files")
