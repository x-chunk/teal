# Changelog

All notable changes to this module are documented here. The module is still
v0, so a minor version may break the API.

## v0.5.0 — 2026-10-20

### BREAKING CHANGE: methods answer with `(*Response[T], error)`

Every service method, `Client.Do` and `Client.DoRaw` now answer with
`(*Response[T], error)` instead of `result.Result[Response[T]]`. On an error
the `*Response` is nil. `Response`, `Meta` and `*Error` are unchanged, and a
refusal still carries its Meta in `Error.Meta`.

The module no longer depends on `go.xchunk.org/anvil/v2`.

### Added

- `*DecodeError`: an answer that was served and charged but could not be read
  or decoded now keeps its Meta in `DecodeError.Meta` instead of losing it.
  Match it with `errors.As`.

Before (v0.4.0):

```go
r, err := c.App.Get(ctx).Value()
if err != nil {
	return err
}
fmt.Println(r.Data.Name, r.Meta.Balance)

total := c.Archive.Count(ctx, req).
	Map(func(r teal.Response[teal.CountResult]) int64 { return r.Data.Total }).
	UnwrapOr(0)

err = c.Vault.Delete(ctx, teal.VaultDeleteRequest{Passphrase: p}).Error()
```

After (v0.5.0):

```go
r, err := c.App.Get(ctx)
if err != nil {
	return err
}
fmt.Println(r.Data.Name, r.Meta.Balance)

var total int64
if r, err := c.Archive.Count(ctx, req); err == nil {
	total = r.Data.Total
}

_, err = c.Vault.Delete(ctx, teal.VaultDeleteRequest{Passphrase: p})
```

To migrate:

- `x.Value()` → drop the call: the method returns the pair itself.
- `x.Error()` → `_, err := …`.
- `Map`/`UnwrapOr` chains → an ordinary `if err != nil` branch.
- Signatures that named `result.Result[teal.Response[T]]` →
  `(*teal.Response[T], error)`.

## v0.4.0

- Every method answers with `result.Result[Response[T]]` from
  `go.xchunk.org/anvil/v2/result`.

## v0.3.0

- The module moved from `github.com/x-chunk/teal` to `go.xchunk.org/teal`.
