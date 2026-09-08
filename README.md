# teal

A Go client for the [Aether Plug-In API v1](http://144.31.187.78:8080/docs) — your
archive, your vault and your account, through an application key of your own.

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
├── archive.go         — /v1/messages/*, /chats, /fields
├── vault.go           — /v1/vault/entries*
├── actions.go         — /v1/actions*
├── insights.go        — /v1/insights, /portraits/{chat}
├── settings.go        — /v1/settings/*
├── example_test.go    — the runnable godoc examples
├── services_test.go   — the endpoints against an httptest server
└── transport_test.go  — the transport itself
```

All 35 documented endpoints are covered.

`transport.go`, `service.go`, `client.go` and `errors.go` are written. The six
service files hold nothing but the service type and the endpoints it is for;
`models.go` holds nothing but `Money`. That is the part left to write.

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

## One thing taken on faith

`GET /v1/usage` with `by_day=true` returns a day-by-day breakdown that the
documentation describes but does not show, so `Usage.Days` and `UsageDay.Day`
are guesses at those two JSON keys. Everything else is built from the response
examples in the docs, and `App.Get` is tested against one of them verbatim.

## Development

```sh
make check   # format, vet and test
make test    # run the test suite
```

## License

MIT. See [LICENSE](LICENSE).
