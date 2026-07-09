# Chores: align charm.land dependency versions with crush; fix examples/hello path claim

**Severity:** low

## 1. Dependency versions lag crush

a2tea pins slightly older versions of the shared v2 stack than crush
(the intended consumer) uses:

| module                  | a2tea  | crush  |
|-------------------------|--------|--------|
| charm.land/bubbletea/v2 | v2.0.6 | v2.0.7 |
| charm.land/lipgloss/v2  | v2.0.3 | v2.0.4 |
| charm.land/glamour/v2   | v2.0.0 | v2.0.1 |
| charm.land/bubbles/v2   | v2.1.0 | v2.1.0 |

Go's MVS will resolve to the higher version inside crush, so this is not
breaking — but it means a2tea is never developed or tested against the
versions it will actually run with. Bump to match crush and keep them in
lockstep (a small CI job or a note in CONTRIBUTING would do).

## 2. examples/hello samplePath() does not do what its comment says

The comment claims the sample resolves "no matter the caller's cwd", but:

- under `go run ./examples/hello`, `os.Executable()` points into the build
  cache temp dir, where no `sample.json` exists, and
- the fallback is `filepath.Join("examples", "hello", "sample.json")` —
  cwd-relative from the **repo root only**. Running from any other directory
  fails with a read error.

Either embed the sample via `go:embed` (simplest, always works) or resolve
relative to `runtime.Caller(0)` for the `go run` case, and fix the comment.

## 3. render/deps.go blank imports

The blank imports pinning future renderer deps also force every downstream
consumer of `render` to compile glamour, bubbles/list, etc. today. Harmless
for crush (it already depends on all of them) but worth remembering to drop
as each real renderer lands, per the file's own TODO.
