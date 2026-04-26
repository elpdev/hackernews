# Hackernews

Hackernews is a terminal UI for browsing Hacker News, reading extracted article text, viewing comment threads, and saving articles for later.

The app uses the official Hacker News Firebase API for story and comment data. Article reading is handled locally by Python and Trafilatura, then rendered in the TUI as Markdown.

## Features

- Browse Top, New, Best, Ask HN, Show HN, and Jobs feeds.
- Page through up to 500 stories, loaded 100 at a time.
- Search the current feed by title, author, URL, and metadata.
- Sort the current feed by default HN order, recency, or points.
- Persist theme, sidebar, default screen, sort, and hide-read preferences.
- Track read stories locally and optionally hide them from feeds.
- Search across stories already loaded from all feeds.
- Read extracted article text inside the terminal.
- View article images inline when the terminal supports it.
- Browse comment threads with collapse/expand and parent navigation.
- Save and remove articles from a local saved list.
- Tag saved articles for later filtering and organization.
- Open stories in the system browser and copy URLs to the clipboard.
- Use a command palette, settings screen, doctor diagnostics, help overlay, sidebar navigation, and selectable themes.

## Requirements

- Go 1.26 or newer.
- Python 3.
- Trafilatura for article extraction.
- Optional browser opener support: `xdg-open` on Linux, `open` on macOS, or the Windows URL handler.
- Optional clipboard support: `wl-copy`, `xclip`, or `xsel` on Linux; `pbcopy` on macOS; `clip` on Windows.

Install Trafilatura as either a Python package or a command-line tool:

```sh
python3 -m pip install trafilatura
# or
brew install trafilatura
```

## Install

Homebrew:

```sh
brew install elpdev/tap/hackernews
```

Arch Linux via AUR with yay:

```sh
yay -S hackernews-bin
```

Or install from the AUR manually:

```sh
git clone https://aur.archlinux.org/hackernews-bin.git
cd hackernews-bin
makepkg -si
```

## Run

From the repository:

```sh
go run ./cmd/hackernews
```

Build a local binary:

```sh
go build ./cmd/hackernews
./hackernews
```

Show version metadata:

```sh
go run ./cmd/hackernews --version
```

## Controls

Global controls:

| Key | Action |
| --- | --- |
| `ctrl+k` | Open command palette |
| `?` | Open help |
| `tab` | Switch focus between main content and sidebar |
| `esc` | Close overlay or go back |
| `q` | Quit |
| `ctrl+c` | Quit |

Story list controls:

| Key | Action |
| --- | --- |
| `up` / `k`, `down` / `j` | Move selection |
| `pgup`, `pgdown` | Move by page |
| `left` / `p`, `right` / `n` | Previous or next 100 stories |
| `/` | Search current feed |
| `ctrl+u` | Clear search |
| `o` | Open selected story in browser |
| `O` | Cycle sort mode |
| `h` | Hide or show read stories |
| `enter` | Extract and read the selected article |
| `c` | Open comments for the selected story |
| `s` | Save or unsave the selected story/article |
| `y` | Copy story URL |
| `r` | Refresh feed |

Article reader controls:

| Key | Action |
| --- | --- |
| `up` / `k`, `down` / `j` | Scroll line by line |
| `pgup`, `pgdown` | Scroll by page |
| `left` / `p`, `right` / `n` | Jump between paragraphs |
| `[` / `]` | Jump between paragraphs |
| `o` | Open article URL in browser |
| `y` | Copy article URL |
| `s` | Save or unsave article |
| `esc` | Return to story list |

Comment controls:

| Key | Action |
| --- | --- |
| `up` / `k`, `down` / `j` | Move between comments |
| `left` / `p`, `right` / `n` | Previous or next top-level thread/search match |
| `space` / `enter` | Collapse or expand comment thread |
| `P` | Jump to parent comment |
| `a` | Collapse or expand all comment threads |
| `/` | Search comments |
| `ctrl+u` | Clear comment search |
| `g`, `G` | Jump to top or bottom |
| `o` | Open HN item in browser |
| `y` | Copy HN item URL |
| `r` | Refresh comments |
| `esc` | Return to the previous feed |

## Saved Articles

Saved screen controls:

| Key | Action |
| --- | --- |
| `/` | Search saved articles |
| `ctrl+u` | Clear search |
| `O` | Cycle sort by saved date, story date, or title |
| `t` | Edit comma-separated tags for the selected article |
| `o` | Open selected article in browser |
| `left` / `p`, `right` / `n` | Jump between paragraphs while reading |

Saved articles are stored locally at:

```text
~/.hackernews/saved.json
```

The file is written with user-only permissions. Saved entries include the HN story metadata, extracted article content, and optional tags so they can be revisited from the Saved screen.

Read history and preferences are stored locally at `~/.hackernews/history.json` and `~/.hackernews/config.json`.

## Settings And Doctor

Open Settings from the sidebar or command palette to change existing preferences such as theme, sidebar visibility, default feed, sort mode, hide-read, and sync enabled state. Long sync fields are configured through `Setup Sync` in the command palette.

Run `Doctor` from the command palette to check local storage paths and optional dependencies such as Python, Trafilatura, browser opener, clipboard tools, and Git sync setup. Press `r` on the Doctor screen to rerun checks.

## Manual Git Sync

Saved articles and read history can be synced manually through a private Git repository.

1. Create a private Git repository on your Git host.
2. Open the command palette with `ctrl+k`.
3. Run `Setup Sync`.
4. Enter the Git remote, branch, and local sync directory.
5. Run `Sync Now` whenever you want to pull, merge, commit, and push state.

The setup command writes sync settings to `~/.hackernews/config.json`. Git authentication is handled by your local Git setup, so SSH remotes such as `git@github.com:you/hackernews-sync.git` work without storing tokens in Hackernews.

The sync repository stores:

```text
saved.json
history.json
deleted_saved.json
```

`deleted_saved.json` prevents articles deleted on one computer from reappearing after syncing from another computer.

## Development

Run tests:

```sh
go test ./...
```

Run a build check:

```sh
go build ./cmd/hackernews
rm -f hackernews
```

The main packages are:

| Path | Purpose |
| --- | --- |
| `cmd/hackernews` | CLI entrypoint and version flag |
| `internal/app` | Root Bubble Tea model, routing, global keys, commands |
| `pkg/screens` | Importable feed, reader, comments, and saved screens |
| `pkg/hn` | Hacker News API client and item types |
| `pkg/articles` | Trafilatura article extraction |
| `pkg/saved` | Local saved article store |
| `pkg/history` | Local read history store |
| `pkg/theme` | Importable theme definitions and styles |
| `pkg/commands` | Command registry and command palette model |
| `internal/browser` | System browser integration |
| `internal/clipboard` | Clipboard integration |

## Releases

Release builds are configured with GoReleaser for Linux, macOS, and Windows on amd64 and arm64.

Create a local snapshot release:

```sh
goreleaser release --snapshot --clean
```

Push a version tag to run the release workflow:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The publish workflow can also create a version tag, run GoReleaser, and publish multi-arch Docker images to GitHub Container Registry.

## Notes

Article extraction first tries the `trafilatura` Python package through `python3`, then falls back to the `trafilatura` command-line tool. This supports both `python3 -m pip install trafilatura` and `brew install trafilatura` setups.
