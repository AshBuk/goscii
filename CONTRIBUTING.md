# Contributing

Thank you for your interest in GOSCII!
A few things to note:

## Code quality

```sh
gofmt -w .          # format
golangci-lint run   # lint
go test ./...       # test
```

CI runs the same checks on every PR.

## Commits

```
prefix: short description
```

Common prefixes: `engine`, `tui`, `ai`, `cmd`, `levels`, `ops`, `docs`.

Use the project's sci-fi voice — see [AGENTS.md](AGENTS.md) for naming conventions and vocabulary.

---

All contributions are appreciated. Adventure stories are especially welcome — handcrafted or AI-generated — if you have a narrative track in mind, open an issue or submit a PR with your levels under `levels/adventures/`. See [docs/ADVENTURE_PROTOCOL.md](docs/ADVENTURE_PROTOCOL.md) for the authoring format.
