package teal

import (
	"net/url"
	"strconv"
	"time"
)

// The payload types, grouped as the API groups its endpoints.
//
// Two conventions run through them. Money arrives as an object, not a number
// — {"credits":N,"display":"$…"} — and is modelled by Money. Timestamps
// arrive either as RFC 3339 strings, which unmarshal into time.Time on their
// own, or as Unix seconds, which are kept as Unix and given an At() helper.
//
// A request type carries the body of one call. Fields the API treats as
// optional are pointers with omitempty, so that leaving one alone and setting
// it to its zero value are different things — it matters wherever zero means
// something, as it does for ttl_seconds.

// Money is the object the API prices things in.
type Money struct {
	Credits Credits `json:"credits"`
	Display string  `json:"display"`
}

// Unix is a timestamp the API sends as seconds since the epoch rather than as
// an RFC 3339 string.
type Unix int64

// At renders the timestamp as a time.Time in UTC.
func (u Unix) At() time.Time { return time.Unix(int64(u), 0).UTC() }

// Quota is one of the account's ceilings, measured. It appears on its own in
// Quotas, and inside Account, ActionList, ActionSettings and Retention.
//
// A quota with no window — the archive's size, the vault's entries — is a
// ceiling rather than an allowance, and carries neither Window nor ResetAt.
type Quota struct {
	Key   string `json:"key"`   // "search:daily", "messages:stored", …
	Label string `json:"label"` // written for a person
	Unit  string `json:"unit"`  // "searches", "messages", …
	// Limit and Remaining arrive as null on an unlimited quota and are zero
	// here: read Unlimited before either of them.
	Limit     int64 `json:"limit"`
	Used      int64 `json:"used"`
	Remaining int64 `json:"remaining"`
	Unlimited bool  `json:"unlimited"`

	Window  string    `json:"window"`   // "day", "week"; empty on a ceiling
	ResetAt time.Time `json:"reset_at"` // when the window turns
}

// --- Application ------------------------------------------------------------

// Application is the application a key was issued for. The key itself is not
// here and is nowhere: it was readable once, when it was issued.
type Application struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Billing     BillingMode `json:"billing"`
	KeyPrefix   string      `json:"key_prefix"` // the twelve characters the bot shows
	KeyIssuedAt time.Time   `json:"key_issued_at"`
	Balance     Money       `json:"balance"`
	Funded      Money       `json:"funded"`
	Spent       Money       `json:"spent"`
	Requests    int64       `json:"requests"`
	LastUsedAt  time.Time   `json:"last_used_at"` // zero for a key never used
	Disabled    bool        `json:"disabled"`
	CreatedAt   time.Time   `json:"created_at"`
	AccountID   int64       `json:"account_id"`
}

// UsageRequest narrows GET /v1/usage.
type UsageRequest struct {
	// Days is how far back to cover, 1 to 365. Zero leaves the API's own
	// default of 30.
	Days int
	// ByDay asks for the day-by-day breakdown as well as the totals.
	ByDay bool
}

func (r *UsageRequest) query() url.Values {
	if r == nil {
		return nil
	}
	q := url.Values{}
	if r.Days > 0 {
		q.Set("days", strconv.Itoa(r.Days))
	}
	if r.ByDay {
		q.Set("by_day", "true")
	}
	return q
}

// Usage is what an application has spent over a window, folded into one row
// per priced operation.
type Usage struct {
	From string `json:"from"` // a date, "2026-08-08"
	To   string `json:"to"`
	// Calls counts operations billed, not requests served: one export bills
	// two operations and is one request.
	Calls   int64     `json:"calls"`
	Credits Money     `json:"credits"`
	Ops     []UsageOp `json:"ops"`
	// Days is the same totals broken down by day and by operation, one row
	// for each operation on each day, newest first. Only with
	// UsageRequest.ByDay; every row carries its Day.
	Days []UsageOp `json:"days"`
}

// UsageOp is one priced operation's share of the spending — over the whole
// window in Usage.Ops, or over one day in Usage.Days.
type UsageOp struct {
	Op      string `json:"op"` // "search:query", "messages:read", …
	Calls   int64  `json:"calls"`
	Units   int64  `json:"units"`
	Credits Money  `json:"credits"`
	// Day is the UTC date the row is counted under, "2026-09-06". It is set
	// on the rows of Usage.Days and empty on the totals.
	Day string `json:"day"`
}

// Account is the account a key opens: the plan behind it, what that plan
// opens, and every quota measured against what has been used of it.
type Account struct {
	AccountID int64 `json:"account_id"`
	Plan      Plan  `json:"plan"`
	// BalanceCents is the account's own balance, and cannot be spent with a
	// key. Moving it onto an application is a bot screen.
	BalanceCents int64   `json:"balance_cents"`
	Quotas       []Quota `json:"quotas"`
	// MeasuredAt is when the report was measured. The API measures it at
	// most once every ten seconds per account, so polling it faster reads
	// the same numbers back — Quotas included.
	MeasuredAt Unix `json:"measured_at"`
}

// Plan is the subscription behind an account.
type Plan struct {
	Tier       string `json:"tier"` // "max", …
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	// Until is when the subscription lapses, zero for one that does not.
	Until time.Time `json:"until"`
	// Permissions are the features the plan opens, by their stable keys —
	// the same keys a forbidden Error names in its Permission.
	Permissions []string `json:"permissions"`
}

// Prices is the price list and the rate ceilings in force on a deployment,
// read from its running configuration.
type Prices struct {
	Prices []Price `json:"prices"`
	// CreditsPerCent is what one cent of account balance buys: 10000.
	CreditsPerCent int64 `json:"credits_per_cent"`
	// Rates is the shield's ceiling for each billing mode.
	Rates map[BillingMode]Rate `json:"rates"`
	// AccountRate is the ceiling every application of one account shares,
	// whatever their modes, and GlobalRate the one every application of
	// every account shares. Either is nil when it is off on this deployment.
	AccountRate *Rate `json:"account_rate"`
	GlobalRate  *Rate `json:"global_rate"`
	// QuotaTTLSeconds is how long a process keeps a plan's quotas in memory
	// before reading them again.
	QuotaTTLSeconds int `json:"quota_ttl_seconds"`
}

// When an operation's price is paid out of the application's balance.
const (
	// ChargingAlways is paid on every call, under credits and hybrid alike.
	ChargingAlways = "always"
	// ChargingOverflow is paid only past what the plan covers: a vault
	// entry or a shortcut beyond the plan's ceiling, say.
	ChargingOverflow = "overflow"
)

// Price is what one operation costs.
type Price struct {
	Op    string `json:"op"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
	Price Money  `json:"price"`
	// Limit names the quota this operation stands in for, if any.
	Limit string `json:"limit"`
	// Charging is ChargingAlways or ChargingOverflow.
	Charging    string `json:"charging"`
	Description string `json:"description"`
}

// Rate is a requests-per-second ceiling and the burst allowed above it. The
// ceiling need not be a whole number.
type Rate struct {
	PerSecond float64 `json:"per_second"`
	Burst     int     `json:"burst"`
}

// --- Archive ----------------------------------------------------------------

// The match modes a Condition may use. GET /v1/fields says which fields
// accept which: matching on a substring rather than a whole value is a plan
// limit.
const (
	MatchContains = "ct"
	MatchEquals   = "eq"
)

// How one Condition joins the one before it.
const (
	ConnAnd = "and"
	ConnOr  = "or"
)

// Condition is one filter of a query. Fields are combined in the order they
// are given, and how many may be combined at once is a plan limit.
type Condition struct {
	// Field names a column, as GET /v1/fields lists them. A field not listed
	// there never reaches the database.
	Field string `json:"field"`
	// Mode is MatchContains or MatchEquals, whichever the field accepts.
	// Empty is MatchEquals.
	Mode string `json:"mode,omitempty"`
	// Conn is ConnAnd or ConnOr, and joins this condition to the one before.
	// It is ignored on the first, and empty is ConnAnd. Joining with ConnOr
	// is a plan limit.
	Conn string `json:"conn,omitempty"`
	// Value is what to match, always written as text and parsed into the
	// field's kind: "42" for a number, "true" for a bool, "2026-09-01" for a
	// time. It is matched exactly as sent, spaces included. A date matches a
	// whole day — a plain date is that day in UTC, and a timestamp with an
	// offset, "2026-09-01T00:30:00+05:00", is that calendar day in its own
	// zone.
	Value string `json:"value"`
}

// SearchRequest is one query against the archive. The same body is taken by
// Search, Count and Export.
type SearchRequest struct {
	// Chat narrows the query to one conversation. Nil searches all of them.
	Chat *int64 `json:"chat,omitempty"`
	// Conditions are the filters, combined in order.
	Conditions []Condition `json:"conditions,omitempty"`
	// Page is which page of the answer Search returns, from zero; Count and
	// Export ignore it. Every Search is billed as one query whatever page it
	// asks for, so the other pages are better turned with SearchPage, which
	// is free.
	Page int `json:"page,omitempty"`
}

// SearchResult is one page of matches.
//
// The total and the page always describe the same moment, but the page
// number is a live position: a message archived between two requests shifts
// the pages after it.
type SearchResult struct {
	Messages []Message `json:"messages"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	Pages    int       `json:"pages"`
	PerPage  int       `json:"per_page"`
	// QueryID names the query for SearchPage, which turns its other pages for
	// nothing for 30 minutes after it was run.
	QueryID string `json:"query_id"`
}

// SearchPageRequest turns to another page of a query already run.
type SearchPageRequest struct {
	// QueryID is what the query's first page carried in SearchResult.QueryID.
	QueryID string `json:"query_id"`
	// Page is which page of matches to return, from zero.
	Page int `json:"page"`
}

// CountResult is the number of matches behind a query, and nothing else.
type CountResult struct {
	Total int64 `json:"total"`
}

// Message is one archived message, in the same shape whether it comes from a
// search, a read by id or an export.
//
// ID is the archive's own id — the one to read a message back by. MessageID
// is Telegram's, which is only unique inside a chat.
type Message struct {
	ID        int64  `json:"id"`
	MessageID int64  `json:"message_id"`
	Chat      int64  `json:"chat"`
	Sender    int64  `json:"sender"`
	Receiver  int64  `json:"receiver"`
	Text      string `json:"text"`
	// Username, FirstName and LastName are the chat's, as Telegram reported
	// them when the message was archived.
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// SenderUsername and SenderName are the sender's.
	SenderUsername string `json:"sender_username"`
	SenderName     string `json:"sender_name"`
	// Deleted marks a message deleted inside Telegram, and DeletedAt says
	// when. The archive keeps it either way.
	Deleted   bool   `json:"deleted"`
	DeletedAt Unix   `json:"deleted_at"`
	MediaType string `json:"media_type"`
	// FileID is only returned when the plan opens the media viewer. Without
	// it the message comes back with everything else and no file id.
	FileID    string    `json:"file_id"`
	CreatedAt time.Time `json:"created_at"`
	// Versions is how many superseded texts the message has, on a search
	// result: zero says asking Versions for its history is not worth a
	// request.
	Versions int `json:"versions"`
	// ReplyToID is the archived message this one answers, readable by the
	// same id as any other. ReplyToMessageID is Telegram's id of it, and is
	// set on its own when the answered message is not in the archive. Both
	// are zero on a message that answers nothing.
	ReplyToID        int64 `json:"reply_to_id"`
	ReplyToMessageID int64 `json:"reply_to_message_id"`
}

// ChatsRequest pages through the conversations in the archive.
type ChatsRequest struct {
	// Page is which page to return, from zero.
	Page int
}

func (r *ChatsRequest) query() url.Values {
	if r == nil || r.Page == 0 {
		return nil
	}
	return url.Values{"page": {strconv.Itoa(r.Page)}}
}

// ChatList is one page of the conversations the archive holds.
type ChatList struct {
	Chats []Chat `json:"chats"`
	Total int64  `json:"total"`
	Page  int    `json:"page"`
}

// Chat is one conversation, with how many messages it carries.
type Chat struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Messages int64  `json:"messages"`
}

// VersionList is every version of a message's text, latest first. How far
// back it goes is a plan limit.
type VersionList struct {
	Versions []MessageVersion `json:"versions"`
}

// MessageVersion is one text a message has carried.
type MessageVersion struct {
	Text string `json:"text"`
	At   Unix   `json:"at"`
	// Current marks the text the message carries now, which is the first.
	Current bool `json:"current"`
}

// Field is a column a Condition may name, and the modes it accepts.
type Field struct {
	Key   string   `json:"key"`
	Kind  string   `json:"kind"` // "string", "int", "int64", "bool", "time"
	Modes []string `json:"modes"`
}

// ExportDocument is what Export streams, for a caller that would rather
// decode it than write it to a file. It is the archive as it stood when the
// export began, and Total is exactly the number of messages it holds.
type ExportDocument struct {
	AccountID  int64     `json:"account_id"`
	ExportedAt time.Time `json:"exported_at"`
	Total      int64     `json:"total"`
	Filters    int       `json:"filters"`
	// Chat is the conversation the export was narrowed to, zero when it
	// covers all of them.
	Chat     int64     `json:"chat"`
	Messages []Message `json:"messages"`
}

// --- Vault ------------------------------------------------------------------

// VaultStoreRequest is one secret to encrypt.
type VaultStoreRequest struct {
	// Passphrase both addresses the entry and derives the key that decrypts
	// it. It is never stored, so nobody — this system included — can open
	// the entry without it or one of its recovery codes.
	Passphrase string `json:"passphrase"`
	Plaintext  string `json:"plaintext"`
}

// VaultRevealRequest opens an entry with its passphrase.
type VaultRevealRequest struct {
	Passphrase string `json:"passphrase"`
}

// VaultRenameRequest moves an entry to another passphrase.
type VaultRenameRequest struct {
	Passphrase    string `json:"passphrase"`
	NewPassphrase string `json:"new_passphrase"`
}

// VaultCodesRequest reissues an entry's recovery codes.
type VaultCodesRequest struct {
	Passphrase string `json:"passphrase"`
}

// VaultRecoverRequest opens an entry with one of its one-time codes.
type VaultRecoverRequest struct {
	// Code is spent by the call. It is read the way a person retypes it:
	// with or without dashes, in either case.
	Code string `json:"code"`
	// NewPassphrase is the passphrase the entry moves to, and is required:
	// the fresh codes exist only in the answer, and an entry its caller can
	// still open with a passphrase of its own choosing is not lost with
	// them if the answer never arrives.
	NewPassphrase string `json:"new_passphrase"`
}

// VaultDeleteRequest destroys the entry a passphrase addresses.
type VaultDeleteRequest struct {
	Passphrase string `json:"passphrase"`
}

// RecoveryCodes are the one-time codes issued with an entry. They are
// readable when they are issued and never again: a client that drops them has
// dropped them for good.
type RecoveryCodes struct {
	// EntryID is the id of the entry Store wrote — what DeleteByID takes. It
	// addresses the entry and opens nothing. ReissueCodes leaves it zero.
	EntryID       int64    `json:"entry_id"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// VaultSecret is a decrypted entry.
type VaultSecret struct {
	Plaintext string `json:"plaintext"`
}

// VaultRecovered is a decrypted entry, opened with a recovery code. The code
// is spent by the call, and every remaining code is replaced with the fresh
// set here — the only set that works from now on.
type VaultRecovered struct {
	Plaintext string `json:"plaintext"`
	// Rekeyed says whether the entry was moved to the new passphrase.
	Rekeyed       bool     `json:"rekeyed"`
	RecoveryCodes []string `json:"recovery_codes"`
	// Revoked is how many of the old codes were thrown away.
	Revoked int `json:"revoked"`
}

// --- Actions ----------------------------------------------------------------

// ActionCreateRequest is one shortcut to store.
type ActionCreateRequest struct {
	// Name is what is typed behind the prefix, and is unique for an account.
	Name string `json:"name"`
	// Body is what the bot writes in its place, placeholders and all. A body
	// using a placeholder the plan does not open is refused rather than
	// rendering as nothing later. It is at most 900 UTF-16 code units — a
	// letter of any alphabet is one, an emoji two — so it always fits a
	// caption.
	Body string `json:"body"`
}

// ActionUpdateRequest changes a shortcut. Only the fields set are written,
// and all of them in one write: a request refused on one field — a rename
// that collides, a body that is refused — changes none of them.
type ActionUpdateRequest struct {
	Name *string `json:"name,omitempty"`
	Body *string `json:"body,omitempty"`
	// ArgsRequired says whether the shortcut fires when it is called with
	// fewer words than its body reads. Off, the missing arguments render as
	// empty; on, nothing fires and the message is sent exactly as typed. A
	// request that only switches it is accepted on every plan.
	ArgsRequired *bool `json:"args_required,omitempty"`
}

// Action is one shortcut.
type Action struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Body string `json:"body"`
	// MediaType and FileID are the attachment the shortcut sends with its
	// body, empty when it sends none.
	MediaType string `json:"media_type"`
	FileID    string `json:"file_id"`
	// MinArgs is how many words typed after the shortcut its body reads,
	// through [[ARG1]] … [[ARG9]] and [[ARGS]]. It is derived from the body
	// and read-only; ArgsRequired is whether a call short of them fires.
	MinArgs      int  `json:"min_args"`
	ArgsRequired bool `json:"args_required"`
	// Uses counts the calls that matched the shortcut, counted before the
	// message was rendered and sent — one that rendered to nothing or that
	// Telegram refused among them. UsedAt is the latest, zero when none.
	Uses      int64     `json:"uses"`
	UsedAt    Unix      `json:"used_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ActionList is every shortcut an account keeps, the prefix they are typed
// behind, and how many more the plan allows.
type ActionList struct {
	Prefix  string   `json:"prefix"`
	Actions []Action `json:"actions"`
	Quota   Quota    `json:"quota"`
}

// Placeholder is a token a shortcut's body may carry, and whether this
// account's plan opens it. A placeholder the plan does not open renders as
// nothing rather than as its token.
type Placeholder struct {
	Token   string `json:"token"` // "[[YOU_FIRST]]"
	Label   string `json:"label"`
	Level   string `json:"level"` // "basic", "advanced" or "exclusive"
	Allowed bool   `json:"allowed"`
	// RequiredPlan names the tier that would open it, when it is not open.
	RequiredPlan string `json:"required_plan"`
}

// --- Insights ---------------------------------------------------------------

// Insights is the whole archive summarised: how much of it there is, what is
// unusual in it, and what is about to run out.
type Insights struct {
	Insights []string `json:"insights"`
}

// Portrait describes the person on the other side of one conversation.
//
// Whether the similar-chats and style-shift sections — Similar and Signals —
// are present depends on the plan.
type Portrait struct {
	ChatID  int64           `json:"chat_id"`
	Subject PortraitSubject `json:"subject"`
	BuiltAt time.Time       `json:"built_at"`
	// Cached says whether the portrait was built for this request or read
	// back. Under credits every request is charged either way.
	Cached    bool              `json:"cached"`
	Model     PortraitModel     `json:"model"`
	Summary   string            `json:"summary"`
	Archetype PortraitArchetype `json:"archetype"`
	Traits    []PortraitTrait   `json:"traits"`
	Topics    []string          `json:"topics"`
	Rhythm    PortraitRhythm    `json:"rhythm"`
	Activity  PortraitActivity  `json:"activity"`
	// Similar are the account's other conversations that sound like this
	// one, and Signals what is worth knowing about the analysis — a style
	// that shifted, say.
	Similar []PortraitSimilar `json:"similar"`
	Signals []PortraitSignal  `json:"signals"`
	// Confidence is how much of the portrait the conversation supports.
	Confidence PortraitConfidence `json:"confidence"`
}

// PortraitSubject is the conversation a portrait was built from.
type PortraitSubject struct {
	Title    string    `json:"title"`
	Username string    `json:"username"`
	Messages int64     `json:"messages"`
	First    time.Time `json:"first"`
	Last     time.Time `json:"last"`
}

// PortraitSimilar is one conversation that sounds like the one portrayed.
type PortraitSimilar struct {
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

// PortraitSignal is one thing worth telling about a portrait's analysis.
type PortraitSignal struct {
	Kind string `json:"kind"`
	// Key identifies the sentence in Text, for a client that would rather
	// write the same finding in a language of its own.
	Key      string `json:"key"`
	Text     string `json:"text"`
	Severity string `json:"severity"` // "info", "notice" or "warning"
}

// PortraitModel is the fitted model a portrait came out of.
type PortraitModel struct {
	Version  int       `json:"version"`
	FittedAt time.Time `json:"fitted_at"`
}

// PortraitArchetype is the archetype a subject falls into, and how well.
type PortraitArchetype struct {
	Key         string  `json:"key"` // "the_essayist", …
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Fit         float64 `json:"fit"`
	// Terms are the terms typical of the archetype — not of this subject.
	Terms []string `json:"terms"`
}

// PortraitTrait is one stylometric trait behind an archetype. Z is how far
// from the mean the subject sits, in standard deviations.
type PortraitTrait struct {
	Key   string  `json:"key"`
	Name  string  `json:"name"`
	Level string  `json:"level"` // "high", "low", …
	Note  string  `json:"note"`
	Score float64 `json:"score"`
	Z     float64 `json:"z"`
}

// PortraitRhythm is when a subject writes.
type PortraitRhythm struct {
	Chronotype string `json:"chronotype"` // "night owl", …
	Tempo      string `json:"tempo"`      // "measured", …
	Week       string `json:"week"`       // "weekdays", …
	// PeakHours are the hours of the day, 0 to 23, they write in most.
	PeakHours      []int   `json:"peak_hours"`
	MeanGapSeconds float64 `json:"mean_gap_seconds"`
	WeekendShare   float64 `json:"weekend_share"`
}

// PortraitActivity is how much a subject writes, and over what span.
type PortraitActivity struct {
	Messages       int64     `json:"messages"`
	Words          int64     `json:"words"`
	UniqueTerms    int64     `json:"unique_terms"`
	First          time.Time `json:"first"`
	Last           time.Time `json:"last"`
	MessagesPerDay float64   `json:"messages_per_day"`
	MediaShare     float64   `json:"media_share"`
}

// PortraitConfidence is how far a portrait can be trusted.
type PortraitConfidence struct {
	Level string  `json:"level"` // "high", "low", …
	Score float64 `json:"score"`
}

// --- Settings ---------------------------------------------------------------

// What a full archive does with a new message.
const (
	// RetentionKeep refuses the newest message.
	RetentionKeep = "keep"
	// RetentionRotate drops the oldest to make room.
	RetentionRotate = "rotate"
)

// Retention is what happens to the archive over time.
type Retention struct {
	Mode string `json:"mode"`
	// TTLSeconds is the window after which a message is dropped whether or
	// not the archive is full. Zero is off.
	TTLSeconds int64 `json:"ttl_seconds"`
	// InChat asks for a message leaving the archive to be deleted from
	// Telegram too.
	InChat bool `json:"in_chat"`
	// CanTTL and CanInChat say whether the plan opens those two settings. A
	// plan that does not refuses the change rather than storing it quietly.
	CanTTL    bool `json:"can_ttl"`
	CanInChat bool `json:"can_in_chat"`
	// TTLChoices are the windows TTLSeconds may be set to, zero — keep for
	// good — among them. It is a closed set: any other number is refused.
	TTLChoices []int64 `json:"ttl_choices"`
	Archive    Quota   `json:"archive"`
}

// RetentionUpdateRequest changes the retention policy. Every field is
// optional, and only the ones set are written — which is why they are
// pointers: zero is a meaningful value for TTLSeconds, where it turns the
// window off. The fields sent are applied together: a request refused on
// one of them changes none.
type RetentionUpdateRequest struct {
	Mode       *string `json:"mode,omitempty"`
	TTLSeconds *int64  `json:"ttl_seconds,omitempty"`
	InChat     *bool   `json:"in_chat,omitempty"`
}

// VaultSettings is how long a decrypted secret stays on screen.
type VaultSettings struct {
	// RevealTTLSeconds is how long a decrypted message stays in the chat,
	// zero when it is not taken back at all.
	RevealTTLSeconds int  `json:"reveal_ttl_seconds"`
	AutoDeletes      bool `json:"auto_deletes"`
	CanRevealTTL     bool `json:"can_reveal_ttl"`
	// RevealTTLChoices are the windows RevealTTLSeconds may be set to besides
	// zero. It is a closed set: any other number is refused.
	RevealTTLChoices []int `json:"reveal_ttl_choices"`
}

// VaultSettingsUpdateRequest sets how long a decrypted vault message stays in
// the Telegram chat before the bot takes it back.
type VaultSettingsUpdateRequest struct {
	// RevealTTLSeconds is the timer: one of VaultSettings.RevealTTLChoices,
	// or zero for the default of 60 seconds. The API requires the field, so
	// it is always sent.
	RevealTTLSeconds int `json:"reveal_ttl_seconds"`
}

// ActionSettings is the prefix shortcuts are typed behind, which families of
// placeholder the plan opens, and how many shortcuts are left.
type ActionSettings struct {
	Prefix       string `json:"prefix"`
	CanUse       bool   `json:"can_use"`
	CanMedia     bool   `json:"can_media"`
	CanAdvanced  bool   `json:"can_advanced"`
	CanExclusive bool   `json:"can_exclusive"`
	Quota        Quota  `json:"quota"`
}

// ActionSettingsUpdateRequest sets the character a shortcut is typed behind,
// so "kiss" is triggered by ".kiss". It is chosen from a closed set: anything
// else is refused.
type ActionSettingsUpdateRequest struct {
	Prefix string `json:"prefix"`
}

// Language is what the account chose, what its Telegram client reports, and
// which of the two is in force.
type Language struct {
	Chosen    string   `json:"chosen"`
	Detected  string   `json:"detected"`
	Effective string   `json:"effective"`
	Supported []string `json:"supported"`
}

// LanguageUpdateRequest sets the language every screen of the bot is drawn
// in.
type LanguageUpdateRequest struct {
	// Language is a code from Language.Supported. An empty string goes back
	// to following the Telegram client. The API refuses a body without the
	// field, so it is always sent.
	Language string `json:"language"`
}
