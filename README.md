# GOSCII

A terminal game for learning Go with a sci-fi narrative.

GOSCII helps a gophernaut navigate through space by writing real Go code — from variables and goroutines, through clouds of oblivion, to awakening.

Inspired by [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/mauricioabreu/golings). GOSCII pushes the CLI-first approach further - no external editor required.

## Install

Requires [Go](https://go.dev/dl/) 1.21 or later.

```sh
go install github.com/AshBuk/goscii@latest
```

```sh
goscii start                        # start or continue the campaign
goscii start --adventure <name>     # jump to a specific adventure
goscii progress                     # show current level
goscii reset                        # reset current level
goscii reset --all                  # wipe all progress
```
