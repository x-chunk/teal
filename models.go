package teal

import "time"

// The payload types. Money arrives as {"credits":N,"display":"$…"} and
// timestamps arrive either as RFC 3339 strings, which unmarshal into
// time.Time on their own, or as Unix seconds, which want an int64.

// Money is the object the API prices things in.
type Money struct {
	Credits Credits `json:"credits"`
	Display string  `json:"display"`
}

type Application struct {
	ID          uint        `json:"id"`
	Name        string      `json:"name"`
	Billing     BillingMode `json:"billing"`
	KeyPrefix   string      `json:"key_prefix"`
	KeyIssuedAt time.Time   `json:"key_issued_at"`
	Balance     Money       `json:"balance"`
	Funded      Money       `json:"funded"`
	Spent       Money       `json:"spent"`
	Requests    uint        `json:"requests"`
	LastUsedAt  time.Time   `json:"last_used_at"`
	Disabled    bool        `json:"disabled"`
	CreatedAt   time.Time   `json:"created_at"`
	AccountID   int64       `json:"account_id"`
}
