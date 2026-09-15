# Publishing `qint-go`

Go has **no central package registry to push to**. Modules are consumed
directly from their Git source (here, GitHub) and cached/served by the public
[Go module proxy](https://proxy.golang.org). "Publishing" a version therefore
means **pushing a semantic-version tag** — there are no registry credentials
and no `publish` command.

**`v0.1.0` is tagged, pushed and live** — verified against `proxy.golang.org` on
2026-09-15, which reports it as the latest version. Nothing further is needed to
release it; the steps below are for cutting the *next* version.

This is the only one of the five Qint SDKs that is published. The others are
blocked on registry tokens (Node, .NET), a name collision (Python), or an
undecided distribution route (WooCommerce) — see `qint-api/docs/GO-LIVE.md` P1-3.

## Prerequisites

- Push access to `github.com/SwizzX-GmbH/qint-go`.
- A clean, green tree: `go test ./...` passes and `gofmt -l .` is empty.
- The module path in `go.mod` is `github.com/SwizzX-GmbH/qint-go` and matches
  the repo URL exactly (required — the proxy fetches by module path).

## Release steps

1. Make sure `Version` in `doc.go` matches the tag you are about to cut
   (`0.1.0` ⇄ `v0.1.0`).

2. Verify locally:

   ```sh
   go vet ./...
   go test ./...
   gofmt -l .          # must print nothing
   ```

3. Tag with a `v`-prefixed semver tag and push it:

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

   > This step was already performed for `v0.1.0` when the repo was created.
   > For future releases, bump the number (`v0.1.1`, `v0.2.0`, ...) following
   > semantic versioning.

4. (Optional) Warm the module proxy and verify the version is live. This makes
   the version show up on pkg.go.dev and confirms the tag is fetchable:

   ```sh
   GOPROXY=proxy.golang.org go list -m github.com/SwizzX-GmbH/qint-go@v0.1.0
   ```

   You can also request the docs page once to trigger indexing:
   `https://pkg.go.dev/github.com/SwizzX-GmbH/qint-go@v0.1.0`

## What consumers run

```sh
go get github.com/SwizzX-GmbH/qint-go@v0.1.0   # a specific version
go get github.com/SwizzX-GmbH/qint-go@latest    # the newest tag
```

## Notes

- **No secrets required.** Unlike npm / PyPI / NuGet, there is no token to
  hold — access is controlled entirely by push permission on the Git repo.
- **Tags are immutable in the proxy.** Once `v0.1.0` is fetched by the proxy,
  its contents are cached permanently. Never move or re-point a released tag;
  cut a new patch version instead.
- **v1 and beyond.** Releasing `v2.0.0` or higher requires a `/v2` suffix on
  the module path in `go.mod` (Go's semantic-import-versioning rule). Not
  relevant for the `v0.x` / `v1.x` line.
- The public docs currently reference `github.com/qint-dev/qint-go`, which does
  not exist. The canonical, installable module path is
  `github.com/SwizzX-GmbH/qint-go`.
