package teal

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// base is embedded in every service and folds one call down to one line:
//
//	func (s *AppService) Get(ctx context.Context) (Application, *Meta, error) {
//		return s.get[Application](ctx, "v1/app", nil)
//	}
//
// The type is always written out, because T is in the result and Go infers
// only from arguments.
type base struct{ c *Client }

// query is post for the archive's reads, which are POSTs only because a query
// does not fit in a URL. They may be retried; the writes post covers may not.
func (b base) query[T any](ctx context.Context, path string, body any) (T, *Meta, error) {
	return b.c.Do[T](ctx, Request{Method: http.MethodPost, Path: path, Body: body, Idempotent: true})
}

func (b base) get[T any](ctx context.Context, path string, q url.Values) (T, *Meta, error) {
	return b.c.Do[T](ctx, Request{Method: http.MethodGet, Path: path, Query: q, Idempotent: true})
}

func (b base) post[T any](ctx context.Context, path string, body any) (T, *Meta, error) {
	return b.c.Do[T](ctx, Request{Method: http.MethodPost, Path: path, Body: body})
}

func (b base) patch[T any](ctx context.Context, path string, body any) (T, *Meta, error) {
	return b.c.Do[T](ctx, Request{Method: http.MethodPatch, Path: path, Body: body})
}

// del is spelled short because delete is a builtin. Deleting twice leaves
// the same nothing behind, so it is retried like a read.
func (b base) del[T any](ctx context.Context, path string) (T, *Meta, error) {
	return b.c.Do[T](ctx, Request{Method: http.MethodDelete, Path: path, Idempotent: true})
}

// postRaw is for the export, the one endpoint that answers with a document
// instead of the envelope. The caller closes the body.
func (b base) postRaw(ctx context.Context, path string, body any) (io.ReadCloser, *Meta, error) {
	return b.c.DoRaw(ctx, Request{Method: http.MethodPost, Path: path, Body: body, Idempotent: true})
}

// none is the payload of an endpoint that answers with {}: reach for it when
// only the Meta and the error are worth returning.
//
//	_, meta, err := s.post[none](ctx, "v1/vault/entries/rename", body)
type none = struct{}
