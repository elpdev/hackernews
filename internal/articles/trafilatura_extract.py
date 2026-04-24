#!/usr/bin/env python3
import json
import sys


def fail(message, code=1):
    print(message, file=sys.stderr)
    raise SystemExit(code)


def main():
    if len(sys.argv) != 2:
        fail("usage: trafilatura_extract.py <url>")

    try:
        import trafilatura
    except ImportError:
        fail("Python package 'trafilatura' is not installed. Run: python3 -m pip install trafilatura")

    url = sys.argv[1]
    downloaded = trafilatura.fetch_url(url)
    if not downloaded:
        fail("could not download article URL")

    markdown = trafilatura.extract(
        downloaded,
        url=url,
        output_format="markdown",
        include_comments=False,
        include_tables=False,
    )
    metadata_json = trafilatura.extract(
        downloaded,
        url=url,
        output_format="json",
        with_metadata=True,
        include_comments=False,
        include_tables=False,
    )

    metadata = {}
    if metadata_json:
        try:
            metadata = json.loads(metadata_json)
        except json.JSONDecodeError:
            metadata = {}

    print(json.dumps({
        "title": metadata.get("title") or "",
        "author": metadata.get("author") or "",
        "date": metadata.get("date") or "",
        "url": metadata.get("url") or url,
        "image": metadata.get("image") or "",
        "excerpt": metadata.get("excerpt") or "",
        "markdown": markdown or "",
    }))


if __name__ == "__main__":
    main()
