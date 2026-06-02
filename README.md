```
╔═══════════════════════════════════════════════╗
║  GOSCII · Sci-fi powered TUI for learning Go  ║
╚═══════════════════════════════════════════════╝
```

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings).

GOSCII pushes the terminal-first approach further - no external editor required.

Helps you navigate through space by writing real Go code — from variables and goroutines, through clouds of oblivion, to awakening.

For folks who love their terminal and want gamified coding sessions that are fun and AI-powered!

## Two ways to play

- **Offline adventures** — handcrafted mission tracks. No API key needed. You write Go, GOSCII runs it and checks the output.
- **AI missions** — bring your own key (BYOK) for an endless stream of generated exercises by topic and difficulty. Pick a provider on first run.

Everything lives on your machine in `~/.local/share/goscii/` (or `$XDG_DATA_HOME/goscii`):

- `config.json` — AI provider, model, and your API key (`chmod 600`)
- `progress.json` — which missions you've solved

## Run

```sh
# On host - requires Go, your code runs directly on your machine
go install github.com/AshBuk/goscii@latest

# In a sandbox - requires Docker. 
# First run pulls the image automatically.
docker run -it --rm ghcr.io/ashbuk/goscii

# Same, but -v keeps your progress between runs (otherwise --rm wipes it on exit).
docker run -it --rm -v ~/.local/share/goscii:/home/goscii/.local/share/goscii ghcr.io/ashbuk/goscii
```

## Usage

`goscii` subcommands - call directly after `go install`, or append to the Docker run command (e.g. `docker run --rm ghcr.io/ashbuk/goscii progress`):

```sh
goscii start                        # launch the home hub
goscii adventure [name]             # run a handcrafted offline adventure directly
goscii progress                     # show solved missions by topic
goscii reset                        # reset adventure checkpoint
goscii reset --all                  # wipe all progress
```

Under Docker, `progress` and `reset` only mean something with the `-v` mount above.

## Built with

Thanks to [Charm](https://charm.sh) for the cool TUI stuff: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss), and to [Cobra](https://github.com/spf13/cobra) for the CLI.
