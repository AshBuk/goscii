# GOSCII

Sci-fi powered TUI for learning Go.

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings).

GOSCII pushes the terminal-first approach further - no external editor required.

Helps you navigate through space by writing real Go code — from variables and goroutines, through clouds of oblivion, to awakening.

## Run

```sh
# on host - requires Go, your code runs directly on your machine
go install github.com/AshBuk/goscii@latest

# in sandbox - requires Docker Compose
docker compose run --rm goscii start

# in sandbox - requires Docker
docker run -it --rm -v ~/.local/share/goscii:/home/goscii/.local/share/goscii ghcr.io/ashbuk/goscii start
```

## Usage

```sh
goscii start                        # open the hub: pick signal, difficulty, mission or adventure
goscii adventure [name]             # run a handcrafted offline adventure directly
goscii progress                     # show solved missions by topic
goscii reset                        # reset adventure checkpoint
goscii reset --all                  # wipe all progress
```
