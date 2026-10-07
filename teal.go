// Package teal is a Go client for the Aether Plug-In HTTP API (v1).
//
//	c, err := teal.New("aek_...")
//	if err != nil {
//		return err
//	}
//	app, meta, err := c.App.Get(ctx)
//
// Every endpoint hangs off a service on the client — App, Archive, Vault,
// Actions, Insights and Settings — and every method returns three things: its
// payload, a *Meta and an error.
//
// Meta is what the request cost, read from the X-Aether-* headers: the
// billing mode it was served under, the credits it spent, the balance left.
// It comes back on a refusal too, because the API refunds before it answers,
// so the balance in it is the balance actually left.
//
// Failures are *Error, carrying the API's own code and, where one code covers
// several states, a reason. Branch on those with IsCode and Error.Reason and
// never on the message, which is written for a person reading a log.
//
// The transport retries what can be sent again without doing anything twice:
// a rate refusal and busy, honouring Retry-After, whatever the method —
// neither did any of the work — and a failure on the server's side when the
// call can safely be repeated: reads, deletes and the archive's queries. A
// write is not sent again after a failure, because it may have taken effect
// before the process failed. A spent quota and a locked vault are never
// retried, and neither is unavailable: each turns on a clock longer than a
// call should block on, and Error.RetryAfter or Error.ResetAt says when.
package teal

// Version is this client's version, sent in the User-Agent header.
const Version = "0.2.0"

// DefaultBaseURL is used when no other base URL is configured: the public
// deployment the documentation at https://aether.xchunk.org/docs describes.
const DefaultBaseURL = "https://aether.xchunk.org"
