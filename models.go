package teal

// The payload types. Money arrives as {"credits":N,"display":"$…"} and
// timestamps arrive either as RFC 3339 strings, which unmarshal into
// time.Time on their own, or as Unix seconds, which want an int64.

// Money is the object the API prices things in.
type Money struct {
	Credits Credits `json:"credits"`
	Display string  `json:"display"`
}
