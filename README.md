```
╔═══════════════════════════════════════════════╗
║  GOSCII · Sci-fi powered TUI for learning Go  ║
╚═══════════════════════════════════════════════╝
```

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings).

GOSCII pushes the terminal-first approach further - no external editor required.

Helps you navigate through space by writing real Go code — from variables and goroutines, through clouds of oblivion, to awakening.

For folks who love their terminal and want gamified coding sessions that are fun and AI-powered!

## Run

```sh
# On host - requires Go, your code runs directly on your machine
go install github.com/AshBuk/goscii@latest

# In a sandbox - requires Docker.
# -v mounts a host folder into the container so your progress survives between runs.
docker run -it --rm -v ~/.local/share/goscii:/home/goscii/.local/share/goscii ghcr.io/ashbuk/goscii-go start

# From a clone of this repo - same as above, but builds the image from source (compose.yml).
docker compose run --rm goscii-go start
```

## Usage

```sh
goscii start                        # launch the home hub
goscii adventure [name]             # run a handcrafted offline adventure directly
goscii progress                     # show solved missions by topic
goscii reset                        # reset adventure checkpoint
goscii reset --all                  # wipe all progress
```

## Built with

Thanks to [Charm](https://charm.sh) for the cool TUI stuff: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss), and to [Cobra](https://github.com/spf13/cobra) for the CLI.
