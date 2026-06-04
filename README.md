[![CI](https://github.com/AshBuk/goscii/actions/workflows/test.yml/badge.svg)](https://github.com/AshBuk/goscii/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AshBuk/goscii)](https://goreportcard.com/report/github.com/AshBuk/goscii)
[![Go Reference](https://pkg.go.dev/badge/github.com/AshBuk/goscii.svg)](https://pkg.go.dev/github.com/AshBuk/goscii)

<img width="640" height="450" alt="goscii-screencast" src="https://github.com/user-attachments/assets/f2be0304-1df5-46bb-ab71-60929edd5c55" />

**Sci-fi TUI for learning Go.**

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings).

GOSCII pushes the terminal-first approach further - no external editor required.

The narrative: you're a stranded astronaut on a crashed station, and every broken system is a Go exercise. You're not alone: the onboard AI, also called GOSCII, hands you missions, give you hints, read the logs and checks the output; and slowly pieces its own corrupted memory back together as you progress.

## Two ways to play

- **Bundled adventures** — handcrafted or pre-generated mission tracks, offline with no "Signal" (API key) needed.
- **AI missions** — bring your own key (Anthropic, OpenAI, or Groq), pick a topic and difficulty, and get an endless stream of generated exercises wrapped in the same sci-fi narrative.

## Run 

```sh
# In a sandbox - requires Docker. 
# First run pulls the image automatically.
docker run -it --rm ghcr.io/ashbuk/goscii

# Same, but -v keeps your progress between runs (otherwise --rm wipes it on exit).
docker run -it --rm -v ~/.local/share/goscii:/home/goscii/.local/share/goscii ghcr.io/ashbuk/goscii

# On host - requires Go, your code runs directly on your machine
go install github.com/AshBuk/goscii@latest
```

## CLI Usage

`goscii` subcommands - call directly after `go install`, or append to the Docker run command (e.g. `docker run --rm ghcr.io/ashbuk/goscii progress`):

```sh
goscii start                        # launch the home hub
goscii adventure [name]             # run a bundled offline adventure directly
goscii progress                     # show solved missions by topic
goscii reset                        # reset adventure checkpoint
goscii reset --all                  # wipe all progress
```
---

Everything lives on your machine in `~/.local/share/goscii/` (or `$XDG_DATA_HOME/goscii`):

- `config.json` — AI provider, model, and your API key (`chmod 600`)
- `progress.json` — which missions you've solved

**Safety:** Docker is the recommended way to play — every release ships a sandboxed image that keeps execution isolated from your filesystem. On the host, GOSCII compiles and runs Go with your own permissions; as a guard it blocks dangerous imports (`os/exec`, `syscall`, `unsafe`, …) and uses a throwaway working directory. If you don't trust your AI provider's output and would rather not run AI-generated code directly, stick with the Docker image.

## Built with

Thanks to [Charm](https://charm.sh) for the cool TUI stuff: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss), and to [Cobra](https://github.com/spf13/cobra) for the CLI.

### Apache 2.0 [LICENSE](LICENSE)

For folks who love their terminal and want gamified coding sessions that are fun and AI-powered!

If GOSCII helped you drop a ⭐ for others to find it.
