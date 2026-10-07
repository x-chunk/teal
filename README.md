# teal

A Go client for the [Aether Plug-In API v1](https://aether.xchunk.org/docs) — your
archive, your vault and your account, through an application key of your own.

```go
c, err := teal.New(os.Getenv("AETHER_KEY")) // https://aether.xchunk.org by default
if err != nil {
	return err
}
res, meta, err := c.Archive.Search(ctx, teal.SearchRequest{
	Conditions: []teal.Condition{{Field: "text", Mode: teal.MatchContains, Value: "invoice"}},
})
```

A self-hosted deployment is one option away: `teal.WithBaseURL("https://your-host")`.

## Install

```sh
go get github.com/x-chunk/teal
```

## Layout

```
teal/
├── teal.go            — package doc, version, default base URL
├── client.go          — the Client and its options
├── transport.go       — the HTTP client: Request, (*Client).Do[T], DoRaw, Meta
├── service.go         — base, embedded in every service: get/post/patch/del[T]
├── errors.go          — *Error, the code constants, IsCode/AsError
├── models.go          — every payload and request type
├── app.go             — /v1/app, /usage, /account, /quotas, /prices
├── archive.go         — /v1/messages/*, /messages/search/page, /chats, /fields
├── vault.go           — /v1/vault/entries*
├── actions.go         — /v1/actions*
├── insights.go        — /v1/insights, /portraits/{chat}
├── settings.go        — /v1/settings/*
├── example_test.go    — the runnable godoc examples
├── services_test.go   — the endpoints against an httptest server
└── transport_test.go  — the transport itself
```

All 36 endpoints of the public API are covered, and nothing else is: the
internal and admin APIs of a deployment and its gRPC API are not this client's
business.

## Conventions

**One request type per call, named `<Service><Action>Request`.** The body of a
call is a struct and never a bare map or a row of positional arguments, so the
wire format lives in one place, a typo in a field will not compile, and the API
can gain a field without breaking anyone's build.

**Required fields are values, optional fields are pointers with `omitempty`.**
It matters wherever zero means something: `RetentionUpdateRequest.TTLSeconds`
set to zero turns the window off, and leaving it nil leaves it alone. The two
must be different, and only a pointer makes them so.

**Ids from the path stay ordinary arguments** — `Update(ctx, id, req)`. The
struct is exactly the body, so it can be marshalled and compared against the
documentation one to one.

**Query parameters are structs too**, with an unexported `query()` that builds
the `url.Values`. A nil request means the API's own defaults.

**Every method returns `(payload, *Meta, error)`**, and one that answers `{}`
returns just `(*Meta, error)`.

## Adding an endpoint

Write the payload type in `models.go`, then one line in the service file:

```go
func (s *AppService) Quotas(ctx context.Context) ([]Quota, *Meta, error) {
	return s.get[[]Quota](ctx, "v1/quotas", nil)
}
```

The helpers on `base` are `get`, `post`, `patch`, `del` and `postRaw`; `none`
is the payload of an endpoint answering `{}`. When none of them fits, reach for
`s.c.Do[T](ctx, Request{…})` underneath.

## Where the shapes come from

Every type is built from the API's own route table and response shapes, and
the tests decode the documentation's response examples verbatim wherever there
is one. `GET /v1/usage?by_day=true`, which the documentation describes but does
not show, answers `days` as flat rows — one per operation per day, each with
its `day` — so `Usage.Days` is a `[]UsageOp`, not a nested type.

## Retries

The transport sends a call again only when doing so cannot do anything twice:

| Refusal | Retried |
|---|---|
| `rate_limited`, `busy` | always, after `Retry-After` — nothing was done |
| `internal`, other 5xx | reads, deletes and the archive's queries only |
| `quota_exhausted` | never — it turns at `reset_at` |
| `rate_limited` with reason `vault_locked` | never — the lock outlasts a call |
| `unavailable` | never — `Error.Reason` and `Error.RetryAfter` say what to do |

## Development

```sh
make check   # format, vet and test
make test    # run the test suite
```

## License

MIT. See [LICENSE](LICENSE).
