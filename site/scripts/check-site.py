#!/usr/bin/env python3
"""Dependency-free structural checks for Dockly's static marketing site."""
from __future__ import annotations

import re
import sys
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[1]
INDEX = ROOT / "index.html"


class SiteParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.errors: list[str] = []
        self.local_assets: set[str] = set()
        self.ids: set[str] = set()
        self.h1_count = 0

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = dict(attrs)
        element_id = values.get("id")
        if element_id:
            if element_id in self.ids:
                self.errors.append(f"duplicate id: {element_id}")
            self.ids.add(element_id)
        if tag == "h1":
            self.h1_count += 1
        if tag == "img" and "alt" not in values:
            self.errors.append(f"image missing alt: {values.get('src', '<missing src>')}")
        if tag == "a" and not values.get("href"):
            self.errors.append("anchor missing href")
        if tag in {"img", "script"} and values.get("src"):
            self._asset(values["src"] or "")
        if tag == "link" and "stylesheet" in (values.get("rel") or "") and values.get("href"):
            self._asset(values["href"] or "")

    def _asset(self, value: str) -> None:
        parsed = urlsplit(value)
        if not parsed.scheme and not parsed.netloc and parsed.path and not parsed.path.startswith("data:"):
            self.local_assets.add(parsed.path.lstrip("/"))


def main() -> int:
    parser = SiteParser()
    source = INDEX.read_text(encoding="utf-8")
    parser.feed(source)
    if '<link rel="canonical" href="https://usedockly.com/">' not in source:
        parser.errors.append("canonical URL must be https://usedockly.com/")
    if "fonts.googleapis.com" in source or "fonts.gstatic.com" in source:
        parser.errors.append("production HTML must use the self-hosted font")
    if 'src="assets/github-mark-white.svg' not in source:
        parser.errors.append("hero CTA must use the self-hosted GitHub mark")
    if parser.h1_count != 1:
        parser.errors.append(f"expected one h1, found {parser.h1_count}")
    for asset in sorted(parser.local_assets):
        if not (ROOT / asset).is_file():
            parser.errors.append(f"missing local asset: {asset}")

    for css in ROOT.glob("*.css"):
        content = css.read_text(encoding="utf-8")
        for raw in re.findall(r"url\(([^)]+)\)", content):
            value = raw.strip(" \t\r\n\"'")
            parsed = urlsplit(value)
            if parsed.scheme or parsed.netloc or value.startswith(("data:", "#")):
                continue
            target = (css.parent / parsed.path).resolve()
            if not target.is_file():
                parser.errors.append(f"{css.name} references missing asset: {value}")

    if parser.errors:
        print("site check failed:", file=sys.stderr)
        for error in parser.errors:
            print(f"- {error}", file=sys.stderr)
        return 1
    print(f"site check passed: {len(parser.local_assets)} HTML assets, {len(parser.ids)} unique ids")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
