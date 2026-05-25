# GOSCII

A terminal interface for learning Go with a sci-fi narrative.

GOSCII helps a gophernaut navigate through space by writing real Go code — from variables and goroutines, through clouds of oblivion, to awakening.

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings). GOSCII pushes the CLI-first approach further - no external editor required.

## Run

**on host:**

Requires [Go](https://go.dev/dl/) installed on your machine.

```sh
go install github.com/AshBuk/goscii@latest
```

**in sandbox:**

Requires [Docker](https://docs.docker.com/get-started/get-docker/)

```sh
docker run -it --rm -v ~/.local/share/goscii:/home/goscii/.local/share/goscii ghcr.io/ashbuk/goscii start
```

Progress is saved to `~/.local/share/goscii` volume on your host.

Exiting the TUI stops and removes the container.

## Usage

```sh
goscii start                        # select a topic and generate a mission
goscii start --adventure <name>     # jump to a specific adventure
goscii progress                     # show solved missions by topic
goscii reset                        # reset adventure checkpoint
goscii reset --all                  # wipe all progress
```
