// Package teal is a Go client for the Aether Plug-In HTTP API (v1).
//
//	c, err := teal.New("aek_...", teal.WithBaseURL("https://your-aether-host"))
//	if err != nil {
//		return err
//	}
//	app, meta, err := Do[Application](ctx, c, Request{Method: http.MethodGet, Path: "v1/app"})
//
// The transport is in transport.go and is complete: authentication, the
// {"ok":…} envelope, the X-Aether-* cost headers, retries and error decoding.
// Everything else — the payload types in models.go and the service methods in
// app.go, archive.go and the rest — is left to be written.
package teal

// Version is this client's version, sent in the User-Agent header.
const Version = "0.1.0"

// DefaultBaseURL is used when no other base URL is configured.
const DefaultBaseURL = "http://144.31.187.78:8080"
