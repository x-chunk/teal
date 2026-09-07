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
├── transport.go       — the HTTP client: Request, Do[T], DoRaw, Meta
├── errors.go          — *Error, the code constants, IsCode/AsError
├── models.go          — the payload types (yours to write)
├── app.go             — /v1/app, /usage, /account, /quotas, /prices
├── archive.go         — /v1/messages/*, /chats, /fields
├── vault.go           — /v1/vault/entries*
├── actions.go         — /v1/actions*
├── insights.go        — /v1/insights, /portraits/{chat}
├── settings.go        — /v1/settings/*
└── transport_test.go  — the transport against an httptest server
```

`transport.go`, `client.go` and `errors.go` are written. The six service files
hold nothing but the service type and the endpoints it is for; `models.go`
holds nothing but `Money`. That is the part left to write.

## Writing an endpoint

Everything goes through one generic function, so a method is three lines:

```go
func (s *AppService) Get(ctx context.Context) (Application, *Meta, error) {
	return Do[Application](ctx, s.c, Request{Method: http.MethodGet, Path: "v1/app"})
}

func (s *ArchiveService) Search(ctx context.Context, req SearchRequest) (SearchPage, *Meta, error) {
	return Do[SearchPage](ctx, s.c, Request{
		Method: http.MethodPost,
		Path:   "v1/messages/search",
		Body:   req,
	})
}
```

`Do[T]` unwraps the `{"ok":true,"data":…}` envelope into a `T`. The export is
the one endpoint that answers with a document instead — `DoRaw` hands its body
back undecoded, for the caller to close.

## What comes back

Every call returns three things: its payload, a `*Meta` and an error.

- **`*Meta`** is what the request cost, read from the `X-Aether-*` headers: the
  billing mode, the credits spent, the balance left, the operations billed.
  It is returned on a refusal too, because the API refunds before it answers,
  so the balance in it is the balance actually left.
- **errors** are `*Error`, carrying the API's own `code`. Branch on the code —
  `teal.IsCode(err, teal.CodeNotFound)` — and never on the message.

Retries cover rate refusals and server failures, honouring `Retry-After`. A
spent quota is never retried: it only turns when its window does.

## Development

```sh
make check   # format, vet and test
make test    # run the test suite
```

## License

MIT. See [LICENSE](LICENSE).
