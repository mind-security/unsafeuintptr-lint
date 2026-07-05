# unsafeuintptr

A Go static analyzer that reports conversions of `unsafe.Pointer` to `uintptr`
that are **not kept alive** — the class of latent memory-safety bug where a
pointer is turned into a bare integer and the referenced object can then be
garbage-collected or moved, leaving the `uintptr` dangling.

It is the **complement** of `go vet`'s
[`unsafeptr`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/unsafeptr)
analyzer, which only checks the reverse direction (`uintptr` → `unsafe.Pointer`).
No existing linter checks the direction this tool does.

## Why

Converting an `unsafe.Pointer` to a `uintptr` produces an integer with no pointer
semantics. Per the [`unsafe` docs](https://pkg.go.dev/unsafe), the garbage
collector will not keep the referenced object alive, and will not update the
value if the object moves. The conversion is only safe in a few specific
patterns; anything else can reference freed or moved memory.

## What it flags

Every `uintptr(<unsafe.Pointer>)` conversion (inline or through a variable),
**unless** it is in a provably-safe context:

1. **Pointer arithmetic converted straight back**, in one expression, with only
   `+ - &^` in between:

   ```go
   p2 := unsafe.Pointer(uintptr(unsafe.Pointer(p)) + offset) // OK
   ```

2. **A call argument to a keep-alive function**:
   - a function in the same package marked `//go:uintptrescapes` or
     `//go:uintptrkeepalive`, or
   - a compiler-recognized syscall function on the built-in allowlist
     (`syscall.Syscall*`, `golang.org/x/sys/unix`, `golang.org/x/sys/windows`,
     and the `NewLazyDLL(...).NewProc(...).Call(...)` methods).

   ```go
   syscall.Syscall(fd, uintptr(unsafe.Pointer(&buf[0])), n, 0)          // OK
   proc.Call(uintptr(unsafe.Pointer(&arg)))                             // OK (Call is //go:uintptrescapes)
   ```

Anything else is reported:

```go
x := uintptr(unsafe.Pointer(p))                 // BAD: stored in a variable
return uintptr(unsafe.Pointer(p))               // BAD: returned
h := holder{addr: uintptr(unsafe.Pointer(p))}   // BAD: stored in a field
ordinaryFunc(uintptr(unsafe.Pointer(p)))        // BAD: callee does not keep it alive
```

The last case is the important one: passing the conversion to an **ordinary**
function (even one that later performs a syscall) is unsafe, because the pointer
crosses the function boundary as a bare integer. A trailing
`runtime.KeepAlive` does **not** make it safe — the correct fix is to pass an
`unsafe.Pointer` through the boundary and convert to `uintptr` only in the
`//go:uintptrescapes` call itself.

## Use it in another repo (no local checkout)

You do not need to clone or build this repo to use it. Pick whichever fits:

**Run it ad hoc** — the Go toolchain fetches, builds, and caches it for you:

```sh
go run github.com/mind-security/unsafeuintptr-lint/cmd/unsafeuintptr@latest ./...
```

**Pin it as a tool dependency** (Go 1.24+) so everyone in the repo runs the same
version, recorded in `go.mod`:

```sh
go get -tool github.com/mind-security/unsafeuintptr-lint/cmd/unsafeuintptr@v0.1.0
go tool unsafeuintptr ./...
```

**Install the binary** once:

```sh
go install github.com/mind-security/unsafeuintptr-lint/cmd/unsafeuintptr@latest
unsafeuintptr ./...
```

**GitHub Actions** — this repo is itself a composite action:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: stable
- uses: mind-security/unsafeuintptr-lint@v0.1.0
  with:
    args: ./... # optional; defaults to ./...
    # allow: pkg.Func  # optional; comma-separated -allow value
```

> **Private module note:** this repo is private, so consuming projects must let
> Go authenticate to fetch it:
>
> ```sh
> export GOPRIVATE=github.com/mind-security
> git config --global url."git@github.com:".insteadOf "https://github.com/"
> ```
>
> In CI, give the job a token/SSH key with read access to this repo.

## Usage

Standalone:

```sh
unsafeuintptr ./...
```

As a `go vet` tool:

```sh
go vet -vettool=$(which unsafeuintptr) ./...
```

### `-allow`

`//go:uintptrescapes` pragmas are not visible across package boundaries, so
keep-alive functions defined in _other_ packages must be listed explicitly.
Add them with `-allow`, as a comma-separated list of fully qualified names —
`pkgpath.Func` or `pkgpath.Type.Method`:

```sh
unsafeuintptr -allow="example.com/x.MySyscall,example.com/x.Handle.Call" ./...
```

## golangci-lint

golangci-lint can't load an arbitrary analyzer at runtime; you compile it in with
the [module plugin system](https://golangci-lint.run/plugins/module-plugins/).
This repo ships the plugin (`plugin/`) and a build config (`.custom-gcl.yml`).

1. Build a custom golangci-lint binary that includes this linter:

   ```sh
   golangci-lint custom   # reads .custom-gcl.yml, writes ./bin/custom-gcl
   ```

   For local development, point the plugin at your checkout instead of a tag by
   swapping `version:` for `path:` in `.custom-gcl.yml`.

2. Enable it in `.golangci.yml` (v2 shown; see `.golangci.example.yml`):

   ```yaml
   version: "2"
   linters:
     enable:
       - unsafeuintptr
     settings:
       custom:
         unsafeuintptr:
           type: module
           description: Reports unsafe.Pointer->uintptr conversions that are not kept alive.
           settings:
             allow: # optional, same as the -allow flag
               - example.com/x.MySyscall
   ```

3. Run the custom binary:

   ```sh
   ./bin/custom-gcl run
   ```

## Known limitations / false positives

- **Address-as-number uses** — e.g. `fmt.Sprintf("%x", uintptr(unsafe.Pointer(p)))`,
  alignment checks, or using an address as a map key — are flagged even though
  they never dereference the pointer later. This is intentional under the strict
  rule; suppress individual sites with a `//nolint:unsafeuintptr` comment (when
  run under golangci-lint) or refactor.
- **Cross-package `//go:uintptrescapes` functions** outside the built-in
  allowlist are flagged, because Go does not expose pragmas across package
  boundaries. Add them via `-allow`.
- Calls through **function values or interface methods** cannot be resolved to a
  declaration and are treated as unsafe.

## Development

```sh
go build ./...
go test ./...
```

Tests use `analysistest` over `unsafeuintptr/testdata/src`.
