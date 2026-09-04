#!/usr/bin/env python3
"""Search and render public result pages with Camoufox.

The Go service feeds this script JSON on stdin:
  {"query":"...", "results":[{"title":"...","url":"..."}], "max_pages":2}

The script intentionally does not solve CAPTCHAs, submit forms, use credentials,
or click through access walls. It opens public search/result URLs and extracts
visible text from successfully rendered pages.
"""

from __future__ import annotations

import json
import re
import sys
from urllib.parse import parse_qs, quote_plus, urlparse
from typing import Any


MAX_SNIPPET_CHARS = 1800
SEARCH_URL = "https://duckduckgo.com/html/?q={query}"


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


def normalize_result_url(href: str) -> str:
    href = (href or "").strip()
    if not href:
        return ""
    parsed = urlparse(href)
    if "duckduckgo.com" in parsed.netloc and parsed.path.startswith("/l/"):
        redirected = parse_qs(parsed.query).get("uddg", [""])[0]
        return redirected.strip()
    if parsed.scheme in ("http", "https") and parsed.netloc:
        return href
    return ""


def collect_search_targets(page: Any, query: str, max_pages: int) -> list[dict[str, str]]:
    page.goto(SEARCH_URL.format(query=quote_plus(query)), wait_until="domcontentloaded", timeout=8000)
    anchors = page.locator("a.result__a, a[href]").evaluate_all(
        """els => els.map(a => ({title: (a.innerText || a.textContent || '').trim(), url: a.href || ''}))"""
    )
    targets = []
    seen = set()
    for item in anchors:
        url = normalize_result_url(item.get("url", ""))
        title = clean_text(item.get("title", ""))
        if not url or url in seen or "duckduckgo.com" in urlparse(url).netloc:
            continue
        seen.add(url)
        targets.append({"title": title, "url": url})
        if len(targets) >= max_pages:
            break
    return targets


def main() -> int:
    try:
        request = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        emit({"error": f"invalid JSON request: {exc}"})
        return 1

    results = request.get("results") or []
    max_pages = int(request.get("max_pages") or 3)
    targets: list[dict[str, str]] = []
    seen = set()
    for item in results:
        url = (item.get("url") or "").strip()
        if not url or url in seen:
            continue
        seen.add(url)
        targets.append(item)
        if len(targets) >= max_pages:
            break

    try:
        from camoufox.sync_api import Camoufox
    except Exception as exc:  # pragma: no cover - depends on optional package
        emit({"error": f"camoufox is not installed or could not be imported: {exc}"})
        return 1

    rendered = []
    with Camoufox(headless=True, block_images=True) as browser:
        page = browser.new_page()
        if not targets:
            try:
                targets = collect_search_targets(page, request.get("query", ""), max_pages)
            except Exception as exc:
                emit({"error": f"camoufox search failed: {exc}"})
                return 1

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
