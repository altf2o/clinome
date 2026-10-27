# Contributing

Bug reports and pull requests are welcome. For anything larger than a small
fix, please open an issue first so we can agree on the approach.

## Development

```sh
go build -o clinome .     # build
go test -race ./...        # unit tests
go vet ./...               # static checks
go test -run='^$' -fuzz=FuzzDecodeWAV -fuzztime=60s .   # fuzz the WAV decoder
```

CI also runs `gofmt`, [staticcheck](https://staticcheck.dev) and
[govulncheck](https://go.dev/doc/security/vuln/) on every pull request.
To run them locally:

```sh
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Guidelines

- Keep the code `gofmt`-formatted and dependency-light.
- Audio code in `Engine.Read` runs on the audio thread under a lock: don't
  allocate heavily, block, or call back into the UI there.
- Add a test for bug fixes where practical. The engine can be tested
  offline by calling `Read` directly (see `engine_test.go`).
- Update `README.md` when you change flags or key bindings, and add an entry
  to `CHANGELOG.md` under **Unreleased**.

## Releasing

Pushing a tag like `v0.2.0` runs GoReleaser, which builds binaries for
macOS, Linux and Windows (amd64 and arm64) and publishes a GitHub release.
To try it locally: `goreleaser release --snapshot --clean`.
