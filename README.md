[![CI](https://github.com/AshBuk/goscii/actions/workflows/test.yml/badge.svg)](https://github.com/AshBuk/goscii/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/AshBuk/goscii.svg)](https://pkg.go.dev/github.com/AshBuk/goscii)

<img width="640" height="450" alt="goscii-screencast" src="https://github.com/user-attachments/assets/f2be0304-1df5-46bb-ab71-60929edd5c55" />

**Sci-fi TUI for learning Go.**

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings).

GOSCII pushes the terminal-first approach further: no external editor required.

You wake up on a crashed station, and every broken system is a Go exercise. The station AI, GOSCII, survived the crash too, though not in one piece. It checks your code, and with every repair it remembers a little more of what happened.

Built for fun, with a lot of help from AI.

## Two ways to play

- **Bundled adventures.** Handcrafted or pre-generated mission tracks, offline, with no "Signal" (API key) needed. The onboarding campaign walks through Go basics in 29 missions.
- **AI missions.** Bring your own key (Anthropic, OpenAI, Groq, or any OpenAI-compatible endpoint such as a local model), pick a topic, a mission kind and a difficulty, and get an endless stream of generated exercises wrapped in the same sci-fi narrative.

Every AI mission is test-fired before you see it. GOSCII compiles the reference answer, runs it three times, and takes the expected output from that run, so the model never decides what counts as correct. A draft that fails goes back to the model with the exact compiler or runtime error until it holds.

AI missions come in four kinds:

| Kind | What you do |
|---|---|
| repair | write the missing code |
| fault | find and fix a planted bug in working-looking code |
| decode | read a program and predict its output |
| refit | rewrite working code under a construct rule, such as "no for loop" |

Mixed mode rotates them. Incident mode chains missions on one station system and carries your own code from step to step. Each solved mission raises the integrity of the system it repaired, and finishing a chain decrypts a crew log from the station's past. Press `ctrl+g` in the cockpit to ask GOSCII about your code; on a failed run it reads the diff between expected and actual output.

## Run

```sh
# In a sandbox, requires Docker.
# First run pulls the image automatically.
docker run -it --rm ghcr.io/ashbuk/goscii

# Same, but -v keeps your progress between runs (otherwise --rm wipes it on exit)
# and -e passes your key without storing it.
docker run -it --rm -e ANTHROPIC_API_KEY \
  -v ~/.local/share/goscii:/home/goscii/.local/share/goscii ghcr.io/ashbuk/goscii

# On host, requires Go: your code runs directly on your machine
go install github.com/AshBuk/goscii@latest
```

GOSCII reads the key from `GOSCII_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` or `GROQ_API_KEY` when none is stored. With no config file it picks the provider from whichever variable is set. The default model is `claude-sonnet-5-5`; the Signal screen lists the models your key can reach.

## CLI Usage

`goscii` subcommands: call directly after `go install`, or append to the Docker run command (e.g. `docker run --rm ghcr.io/ashbuk/goscii progress`):

```sh
goscii start                        # launch the home hub
goscii adventure [name]             # run a bundled offline adventure directly
goscii progress                     # solved missions by topic, station integrity
goscii logs                         # crew logs recovered so far
goscii reset                        # reset adventure checkpoint
goscii reset --all                  # wipe all progress

goscii signal models                # models the wired key can reach
goscii signal bench                 # forge a batch of missions and report how the model holds up
goscii forge <track> --topic maps   # forge a draft adventure track for hand editing
```

`signal bench` and `forge` spend tokens on your key. See [docs/ADVENTURE_PROTOCOL.md](docs/ADVENTURE_PROTOCOL.md) for turning a forged track into a bundled adventure.

---

Everything lives on your machine in `~/.local/share/goscii/` (or `$XDG_DATA_HOME/goscii`):

- `config.json`: AI provider, model, and your API key if you chose to store it (`chmod 600`)
- `progress.json`: solved missions, station integrity, recovered crew logs
- `archive/`: every mission GOSCII forged for you, replayable offline from the hub

**Safety:** Docker is the recommended way to play: every release ships a sandboxed image that keeps execution isolated from your filesystem. On the host, GOSCII compiles and runs Go with your own permissions. As a guard it blocks dangerous imports (`os/exec`, `syscall`, `unsafe`, …), runs the program in a throwaway directory, and strips the environment down to `PATH`, `HOME`, `TMPDIR` and `LANG`, so mission code cannot read your API keys. If you don't trust your AI provider's output and would rather not run AI-generated code directly, stick with the Docker image.

## Built with

Thanks to [Charm](https://charm.sh) for the cool TUI stuff: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss), to [Cobra](https://github.com/spf13/cobra) for the CLI, and to the [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go).

### Apache 2.0 [LICENSE](LICENSE)
