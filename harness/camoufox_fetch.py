#!/usr/bin/env python3
"""Render Perplexity result pages with Camoufox and return concise snippets.

The Go service feeds this script JSON on stdin:
  {"query":"...", "results":[{"title":"...","url":"..."}], "max_pages":3}

The script intentionally does not solve CAPTCHAs, submit forms, use credentials,
or click through access walls. It only opens public URLs already returned by the
search provider and extracts visible text from successfully rendered pages.
"""

from __future__ import annotations

import json
import re
import sys
from typing import Any


MAX_SNIPPET_CHARS = 1800


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=False))


def clean_text(text: str) -> str:
    text = re.sub(r"\s+", " ", text or "").strip()
    return text[:MAX_SNIPPET_CHARS]


def blocked_text(text: str) -> bool:
    lowered = text.lower()
    signals = (
        "access denied",
        "are you a robot",
        "captcha",
        "enable javascript",
        "forbidden",
        "just a moment",
        "please verify",
        "robot check",
        "unusual traffic",
        "verifique que você",
        "verifique que voce",
    )
    return any(signal in lowered for signal in signals)


def main() -> int:
    try:
        request = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        emit({"error": f"invalid JSON request: {exc}"})
        return 1

    results = request.get("results") or []
    max_pages = int(request.get("max_pages") or 3)
    targets = []
    seen = set()
    for item in results:
        url = (item.get("url") or "").strip()
        if not url or url in seen:
            continue
        seen.add(url)
        targets.append(item)
        if len(targets) >= max_pages:
            break

    if not targets:
        emit({"results": []})
        return 0

    try:
        from camoufox.sync_api import Camoufox
    except Exception as exc:  # pragma: no cover - depends on optional package
        emit({"error": f"camoufox is not installed or could not be imported: {exc}"})
        return 1

    rendered = []
    with Camoufox(headless=True) as browser:
        page = browser.new_page()
        for item in targets:
            url = item.get("url", "")
            try:
                page.goto(url, wait_until="domcontentloaded", timeout=8000)
                try:
                    page.wait_for_load_state("networkidle", timeout=2000)
                except Exception:
                    pass
                title = clean_text(page.title()) or item.get("title", "")
                body = clean_text(page.locator("body").inner_text(timeout=3000))
                if not body or blocked_text(body):
                    continue
                rendered.append(
                    {
                        "title": title,
                        "url": url,
                        "snippet": body,
                    }
                )
            except Exception as exc:
                print(f"camoufox_fetch skipped {url}: {exc}", file=sys.stderr)

    emit({"results": rendered})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
