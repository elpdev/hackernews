# Hackernews

A terminal UI for reading Hacker News top stories and extracted article text.

## Features

- Bubble Tea v2 app shell
- Hacker News top stories browser
- In-TUI article reader powered by Trafilatura and Glamour
- Command palette
- Header/sidebar/main/footer layout
- Screen router
- Global keybindings
- Help overlay
- Theme system with Phosphor, Muted Dark, and Miami themes
- Logs/debug screen
- GoReleaser release pipeline

## Requirements

- Go 1.26+
- Python 3
- Trafilatura for article extraction

Install Trafilatura with:

```sh
python3 -m pip install trafilatura
```

The story list uses the official Hacker News API directly. Article reading shells out to local Python and requires the `trafilatura` package to be available.

## Development

```sh
go run ./cmd/hackernews
```

## Test

```sh
go test ./...
```

## Snapshot Release Build

```sh
goreleaser release --snapshot --clean
```

## Docker

Manual publishes build and push multi-arch images to GitHub Container Registry:

```sh
docker run --rm -it ghcr.io/<owner>/hackernews:latest
```

## Release

```sh
git tag v0.1.0
git push origin v0.1.0
```

## Version

```sh
go run ./cmd/hackernews --version
```
