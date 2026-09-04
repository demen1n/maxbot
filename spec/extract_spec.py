#!/usr/bin/env python3
"""Извлекает актуальную OpenAPI-спеку MAX Bot API из бандла dev.max.ru.

Отдельным файлом MAX спеку не публикует: она вшита как JSON.parse('...') в один
из JS-чанков документации. Скрипт находит нужный чанк по странице /docs-api,
распаковывает JS-литерал и сохраняет результат в spec/max-bot-api-<version>.json.

Запуск: python3 spec/extract_spec.py [--out-dir spec]
"""

import argparse
import json
import os
import re
import urllib.request

BASE = "https://dev.max.ru"
PAGE = BASE + "/docs-api"
UA = {"User-Agent": "Mozilla/5.0"}


def fetch(url):
    return urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=60).read().decode(
        "utf-8", "replace"
    )


def find_spec_chunk():
    page = fetch(PAGE)
    chunks = sorted(set(re.findall(r'src="(/_next/static/chunks/[^"]+)"', page)))
    for path in chunks:
        body = fetch(BASE + path)
        if '"openapi"' in body and "JSON.parse('" in body:
            return path, body
    raise SystemExit("spec chunk not found: docs bundle layout changed")


def unescape_js_string(lit):
    """JS-литерал в одинарных кавычках -> текст JSON."""
    out, i = [], 0
    while i < len(lit):
        c = lit[i]
        if c == "\\" and i + 1 < len(lit):
            n = lit[i + 1]
            if n == "\\":
                out.append("\\")
            elif n == "'":
                out.append("'")
            elif n == "x":
                out.append(chr(int(lit[i + 2 : i + 4], 16)))
                i += 4
                continue
            else:
                # \n, \" и прочее — валидные JSON-escape, оставляем как есть
                out.append(c)
                out.append(n)
            i += 2
            continue
        out.append(c)
        i += 1
    return "".join(out)


def extract(body):
    start = body.find("JSON.parse('")
    if start < 0:
        raise SystemExit("JSON.parse literal not found")
    start += len("JSON.parse('")
    i = start
    while i < len(body):
        if body[i] == "\\":
            i += 2
            continue
        if body[i] == "'":
            break
        i += 1
    return json.loads(unescape_js_string(body[start:i]))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out-dir", default=os.path.dirname(os.path.abspath(__file__)))
    args = ap.parse_args()

    path, body = find_spec_chunk()
    spec = extract(body)
    version = spec["info"]["version"]
    out = os.path.join(args.out_dir, f"max-bot-api-{version}.json")
    with open(out, "w", encoding="utf-8") as fh:
        json.dump(spec, fh, ensure_ascii=False, indent=1)
        fh.write("\n")
    print(f"chunk:  {path}")
    print(f"spec:   {out}")
    print(f"info:   version={version} paths={len(spec['paths'])} schemas={len(spec['components']['schemas'])}")


if __name__ == "__main__":
    main()
