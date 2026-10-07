package teal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// The error codes the API answers with. Branch on these and never on the
// message, which is written for a person reading a log.
const (
	CodeBadRequest         = "bad_request"          // 400
	CodeUnauthorized       = "unauthorized"         // 401
	CodeInsufficientCredit = "insufficient_credits" // 402
	CodeForbidden          = "forbidden"            // 403, the plan does not open it
	CodeScopeRequired      = "scope_required"       // 403, the key's owner has not opened it
	CodeAccountBlocked     = "account_blocked"      // 403
	CodeNotFound           = "not_found"            // 404, and a wrong vault passphrase
	CodeMethodNotAllowed   = "method_not_allowed"   // 405
	CodeConflict           = "conflict"             // 409
	CodeTooLarge           = "too_large"            // 413, an export past 50 MB
	CodeQuotaExhausted     = "quota_exhausted"      // 429
	CodeRateLimited        = "rate_limited"         // 429
	CodeInternal           = "internal"             // 500
	CodeBusy               = "busy"                 // 503, no room right now
	CodeUnavailable        = "unavailable"          // 503, not as things stand
)

// The reasons an Error narrows its code with, where one code covers states a
// client acts on differently. They are as stable as the codes.
const (
	// ReasonNotConfigured is unavailable on a deployment that does not run
	// Aether Plug-In at all.
	ReasonNotConfigured = "not_configured"
	// ReasonModelTraining is unavailable for a portrait whose model is still
	// being fitted. It comes with a RetryAfter, and is the one reason worth
	// coming back for.
	ReasonModelTraining = "model_training"
	// ReasonPortraitsDisabled is unavailable on a deployment that has
	// portraits switched off. Waiting does not help.
	ReasonPortraitsDisabled = "portraits_disabled"
	// ReasonNotEnoughHistory is not_found for a chat too short to portray
	// yet.
	ReasonNotEnoughHistory = "not_enough_history"
	// ReasonEmptyChat is not_found for a chat whose other side never wrote.
	ReasonEmptyChat = "empty_chat"
	// ReasonVaultLocked is rate_limited for a vault shut to guesses: too many
	// wrong passphrases and codes were tried on it, and its owner has been
	// told. RetryAfter is when the lock runs out.
	ReasonVaultLocked = "vault_locked"
)

// The scopes an Error with CodeScopeRequired names. A key can read without
// any; whatever reaches past reading is opened to it by its owner, on the
// application's screen in the bot, and nothing a request carries opens it.
const (
	// ScopeSettingsWrite is changing the account's settings.
	ScopeSettingsWrite = "settings:write"
	// ScopeVaultWrite is writing to the vault: storing, renaming, reissuing
	// codes, recovering and deleting.
	ScopeVaultWrite = "vault:write"
)

// Error is a refusal: the {"ok":false,"error":{…}} body, the status it
// arrived with, and the cost headers that came with it.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	// Reason narrows Code down, for the codes that cover several states:
	// one of the Reason constants, or empty.
	Reason string

	// Set on quota_exhausted: which quota refused, and where it stands.
	// A refusal from the shield names the quota and its reset only.
	Limit      string
	Used       int64
	LimitValue int64
	ResetAt    time.Time

	// Set on rate_limited, busy, and unavailable when waiting helps — from
	// the Retry-After header, or the body's retry_after when a proxy has
	// dropped the header.
	RetryAfter time.Duration

	// Set on forbidden: the feature the plan does not open, and the
	// cheapest plan that does, when there is one to upgrade to.
	Permission   string
	RequiredPlan string

	// Set on scope_required: ScopeSettingsWrite or ScopeVaultWrite.
	Scope string

	Meta *Meta
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("teal: %s (http %d)", e.Code, e.StatusCode)
	}
	return fmt.Sprintf("teal: %s: %s (http %d)", e.Code, e.Message, e.StatusCode)
}

// IsCode reports whether err is an *Error carrying one of the given codes.
//
//	if teal.IsCode(err, teal.CodeQuotaExhausted, teal.CodeRateLimited) { … }
func IsCode(err error, codes ...string) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	for _, c := range codes {
		if e.Code == c {
			return true
		}
	}
	return false
}

// AsError extracts the *Error from err, if there is one.
func AsError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

// retryable reports whether a request refused this way is worth sending
// again.
//
// A rate refusal always is: nothing was done, and the shield says when to
// come back. So is busy, which is the API having no room for the request
// before any of its work began — the vault's queue answers it only for a
// request it never started. A failure on the server's side only is when the
// request can be repeated safely — the process may have failed after the work
// was done, and a second vault entry is worse than an error.
//
// A spent quota never is: it turns when its window does and not before. A
// locked vault never is either: the lock runs for longer than a call should
// block, and a guess sent the moment it lifts is a guess that locks it again.
// Neither is unavailable, which is a state of the deployment rather than of
// the request — a portrait model still being fitted says how long to wait in
// RetryAfter, and that wait is the caller's to schedule.
func (e *Error) retryable(idempotent bool) bool {
	if e.Code == CodeRateLimited || e.StatusCode == http.StatusTooManyRequests {
		return e.Code != CodeQuotaExhausted && e.Reason != ReasonVaultLocked
	}
	switch e.Code {
	case CodeBusy:
		return true
	case CodeUnavailable:
		return false
	}
	return idempotent && (e.Code == CodeInternal || e.StatusCode >= 500)
}

// decodeError reads a refusal off the wire. The body is drained and closed,
// so the connection can be reused.
func decodeError(resp *http.Response, meta *Meta) *Error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return errorFromBody(raw, resp.StatusCode, meta)
}

// errorFromBody builds the *Error, falling back to the status when the body
// is not the envelope — a proxy in front of the API, say.
func errorFromBody(raw []byte, status int, meta *Meta) *Error {
	e := &Error{
		StatusCode: status,
		Code:       codeForStatus(status),
		Meta:       meta,
	}
	if meta != nil {
		e.RetryAfter = meta.RetryAfter
	}

	var env struct {
		Error *struct {
			Code         string `json:"code"`
			Message      string `json:"message"`
			Reason       string `json:"reason"`
			Limit        string `json:"limit"`
			Used         int64  `json:"used"`
			LimitValue   int64  `json:"limit_value"`
			ResetAt      int64  `json:"reset_at"`
			RetryAfter   int64  `json:"retry_after"`
			Permission   string `json:"permission"`
			RequiredPlan string `json:"required_plan"`
			Scope        string `json:"scope"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err == nil && env.Error != nil {
		e.Code = env.Error.Code
		e.Message = env.Error.Message
		e.Reason = env.Error.Reason
		e.Limit = env.Error.Limit
		e.Used = env.Error.Used
		e.LimitValue = env.Error.LimitValue
		if env.Error.ResetAt > 0 {
			e.ResetAt = time.Unix(env.Error.ResetAt, 0).UTC()
		}
		// The header is what the API sends first and what a proxy may
		// strip; the body says the same and is read when it is all there is.
		if e.RetryAfter == 0 && env.Error.RetryAfter > 0 {
			e.RetryAfter = time.Duration(env.Error.RetryAfter) * time.Second
		}
		e.Permission = env.Error.Permission
		e.RequiredPlan = env.Error.RequiredPlan
		e.Scope = env.Error.Scope
	}
	if meta != nil && e.ResetAt.IsZero() && !meta.QuotaReset.IsZero() {
		e.ResetAt = meta.QuotaReset
	}
	return e
}

func codeForStatus(status int) string {
	if status < 300 {
		// A refusal that arrived with a success status says nothing about
		// itself but what is in its body.
		return CodeInternal
	}
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusPaymentRequired:
		return CodeInsufficientCredit
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case http.StatusConflict:
		return CodeConflict
	case http.StatusRequestEntityTooLarge:
		return CodeTooLarge
	case http.StatusTooManyRequests:
		return CodeRateLimited
	case http.StatusServiceUnavailable:
		return CodeUnavailable
	default:
		return CodeInternal
	}
}
