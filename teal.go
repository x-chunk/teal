// Package teal is a Go client for the Aether Plug-In HTTP API (v1).
//
//	c, err := teal.New("aek_...", teal.WithBaseURL("https://your-aether-host"))
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
// Failures are *Error, carrying the API's own code. Branch on the code with
// IsCode and never on the message, which is written for a person reading a
// log. The transport retries a rate refusal and a failure on the server's
// side, honouring Retry-After; a spent quota is never retried, because it
// only turns when its window does.
package teal

// Version is this client's version, sent in the User-Agent header.
const Version = "0.1.0"

// DefaultBaseURL is used when no other base URL is configured.
const DefaultBaseURL = "http://144.31.187.78:8080"
