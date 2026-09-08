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
	Key       string `json:"key"`   // "search:daily", "messages:stored", …
	Label     string `json:"label"` // written for a person
	Unit      string `json:"unit"`  // "searches", "messages", …
	Limit     int64  `json:"limit"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	Unlimited bool   `json:"unlimited"`

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
	LastUsedAt  time.Time   `json:"last_used_at"`
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
	Calls   int64      `json:"calls"`
	Credits Money      `json:"credits"`
	Ops     []UsageOp  `json:"ops"`
	Days    []UsageDay `json:"days"` // only with UsageRequest.ByDay
}

// UsageOp is one priced operation's share of the spending.
type UsageOp struct {
	Op      string `json:"op"` // "search:query", "messages:read", …
	Calls   int64  `json:"calls"`
	Units   int64  `json:"units"`
	Credits Money  `json:"credits"`
}

// UsageDay is one day of the breakdown UsageRequest.ByDay asks for.
type UsageDay struct {
	Day     string    `json:"day"`
	Calls   int64     `json:"calls"`
	Credits Money     `json:"credits"`
	Ops     []UsageOp `json:"ops"`
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
}

// Plan is the subscription behind an account.
type Plan struct {
	Tier        string    `json:"tier"` // "ultra", …
	Name        string    `json:"name"`
	PriceCents  int64     `json:"price_cents"`
	Until       time.Time `json:"until"`
	Permissions []string  `json:"permissions"`
}

// Prices is the price list and the rate ceilings in force on a deployment,
// read from its running configuration.
type Prices struct {
	Prices []Price `json:"prices"`
	// CreditsPerCent is what one cent of account balance buys: 10000.
	CreditsPerCent int64 `json:"credits_per_cent"`
	// Rates is the shield's ceiling for each billing mode.
	Rates map[BillingMode]Rate `json:"rates"`
	// QuotaTTLSeconds is how long a process keeps a plan's quotas in memory
	// before reading them again.
	QuotaTTLSeconds int `json:"quota_ttl_seconds"`
}

// Price is what one operation costs.
type Price struct {
	Op    string `json:"op"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
	Price Money  `json:"price"`
	// Limit names the quota this operation stands in for, if any.
	Limit string `json:"limit"`
	// Charging is when the balance pays: "always", "past the plan", "never".
	Charging    string `json:"charging"`
	Description string `json:"description"`
}

// Rate is a requests-per-second ceiling and the burst allowed above it.
type Rate struct {
	PerSecond int `json:"per_second"`
	Burst     int `json:"burst"`
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
	Mode string `json:"mode"`
	// Conn is ConnAnd or ConnOr, and joins this condition to the one before.
	// It is ignored on the first.
	Conn string `json:"conn,omitempty"`
	// Value is what to match: a string, a number or a date, as the field's
	// kind requires.
	Value any `json:"value"`
}

// SearchRequest is one query against the archive. The same body is taken by
// Search, Count and Export.
type SearchRequest struct {
	// Chat narrows the query to one conversation. Nil searches all of them.
	Chat *int64 `json:"chat,omitempty"`
	// Conditions are the filters, combined in order.
	Conditions []Condition `json:"conditions,omitempty"`
	// Page is which page of the answer to return, from zero. Paging is not
	// free: every call is one query and is billed as one.
	Page int `json:"page,omitempty"`
}

// SearchResult is one page of matches.
type SearchResult struct {
	Messages []Message `json:"messages"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	Pages    int       `json:"pages"`
	PerPage  int       `json:"per_page"`
}

// CountResult is the number of matches behind a query, and nothing else.
type CountResult struct {
	Total int64 `json:"total"`
}

// Message is one archived message.
//
// ID is the archive's own id — the one to read a message back by. MessageID
// is Telegram's, which is only unique inside a chat.
type Message struct {
	ID             int64  `json:"id"`
	MessageID      int64  `json:"message_id"`
	Chat           int64  `json:"chat"`
	Sender         int64  `json:"sender"`
	Receiver       int64  `json:"receiver"`
	SenderUsername string `json:"sender_username"`
	Text           string `json:"text"`
	MediaType      string `json:"media_type"`
	// FileID is only returned when the plan opens the media viewer. Without
	// it the message comes back with everything else and no file id.
	FileID    string    `json:"file_id"`
	Deleted   bool      `json:"deleted"`
	CreatedAt time.Time `json:"created_at"`
	// Versions is how many texts this message has had, on a search result.
	Versions int `json:"versions"`
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
	Kind  string   `json:"kind"` // "string", "int64", "time"
	Modes []string `json:"modes"`
}

// ExportDocument is what Export streams, for a caller that would rather
// decode it than write it to a file.
type ExportDocument struct {
	AccountID  int64     `json:"account_id"`
	ExportedAt time.Time `json:"exported_at"`
	Total      int64     `json:"total"`
	Filters    int       `json:"filters"`
	Chat       int64     `json:"chat"`
	Messages   []Message `json:"messages"`
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
	Code string `json:"code"`
	// NewPassphrase moves the entry on the way, when it is not empty.
	NewPassphrase string `json:"new_passphrase,omitempty"`
}

// VaultDeleteRequest destroys the entry a passphrase addresses.
type VaultDeleteRequest struct {
	Passphrase string `json:"passphrase"`
}

// RecoveryCodes are the one-time codes issued with an entry. They are
// readable when they are issued and never again: a client that drops them has
// dropped them for good.
type RecoveryCodes struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

// VaultSecret is a decrypted entry.
type VaultSecret struct {
	Plaintext string `json:"plaintext"`
}

// VaultRecovered is a decrypted entry, opened with a recovery code. The code
// is spent by the call either way, and every remaining code is replaced with
// the fresh set here.
type VaultRecovered struct {
	Plaintext string `json:"plaintext"`
	// Rekeyed says whether the entry was moved to a new passphrase.
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
	// rendering as nothing later.
	Body string `json:"body"`
}

// ActionUpdateRequest renames a shortcut, changes what it says, or both.
// Only the fields set are written, and the name is applied first, so a rename
// that collides fails before the body is touched.
type ActionUpdateRequest struct {
	Name *string `json:"name,omitempty"`
	Body *string `json:"body,omitempty"`
}

// Action is one shortcut.
type Action struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
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
	Level   string `json:"level"` // "basic", "exclusive", …
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
// Whether the similar-chats and style-shift sections are present depends on
// the plan.
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

// PortraitModel is the fitted model a portrait came out of.
type PortraitModel struct {
	Version  int       `json:"version"`
	FittedAt time.Time `json:"fitted_at"`
}

// PortraitArchetype is the archetype a subject falls into, and how well.
type PortraitArchetype struct {
	Key   string  `json:"key"` // "the_essayist", …
	Label string  `json:"label"`
	Fit   float64 `json:"fit"`
}

// PortraitTrait is one stylometric trait behind an archetype. Z is how far
// from the mean the subject sits, in standard deviations.
type PortraitTrait struct {
	Key   string  `json:"key"`
	Name  string  `json:"name"`
	Level string  `json:"level"` // "high", "low", …
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

// PortraitActivity is how much a subject writes.
type PortraitActivity struct {
	Messages       int64   `json:"messages"`
	Words          int64   `json:"words"`
	UniqueTerms    int64   `json:"unique_terms"`
	MessagesPerDay float64 `json:"messages_per_day"`
	MediaShare     float64 `json:"media_share"`
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
	CanTTL    bool  `json:"can_ttl"`
	CanInChat bool  `json:"can_in_chat"`
	Archive   Quota `json:"archive"`
}

// RetentionUpdateRequest changes the retention policy. Every field is
// optional, and only the ones set are written — which is why they are
// pointers: zero is a meaningful value for TTLSeconds, where it turns the
// window off.
type RetentionUpdateRequest struct {
	Mode       *string `json:"mode,omitempty"`
	TTLSeconds *int64  `json:"ttl_seconds,omitempty"`
	InChat     *bool   `json:"in_chat,omitempty"`
}

// VaultSettings is how long a decrypted secret stays on screen.
type VaultSettings struct {
	RevealTTLSeconds int  `json:"reveal_ttl_seconds"`
	AutoDeletes      bool `json:"auto_deletes"`
	CanRevealTTL     bool `json:"can_reveal_ttl"`
}

// VaultSettingsUpdateRequest sets how long a decrypted vault message stays in
// the Telegram chat before the bot takes it back.
type VaultSettingsUpdateRequest struct {
	// RevealTTLSeconds is the timer. Zero means the plan's own default, so
	// the field is always sent.
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
	// to following the Telegram client, so the field is always sent.
	Language string `json:"language"`
}
