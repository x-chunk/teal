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
	CodeForbidden          = "forbidden"            // 403
	CodeAccountBlocked     = "account_blocked"      // 403
	CodeNotFound           = "not_found"            // 404, and a wrong vault passphrase
	CodeConflict           = "conflict"             // 409
	CodeQuotaExhausted     = "quota_exhausted"      // 429
	CodeRateLimited        = "rate_limited"         // 429
	CodeInternal           = "internal"             // 500
	CodeUnavailable        = "unavailable"          // 503
)

// Error is a refusal: the {"ok":false,"error":{…}} body, the status it
// arrived with, and the cost headers that came with it.
type Error struct {
	StatusCode int
	Code       string
	Message    string

	// Set on quota_exhausted: which quota refused, and where it stands.
	Limit      string
	Used       int64
	LimitValue int64
	ResetAt    time.Time

	// Set on rate_limited, from the Retry-After header.
	RetryAfter time.Duration

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
// come back. A failure on the server's side only is when the request can be
// repeated safely — the process may have failed after the work was done, and
// a second vault entry is worse than an error. A spent quota never is: it
// turns when its window does and not before. Neither does unavailable, which
// says the API is not configured on this deployment at all.
func (e *Error) retryable(idempotent bool) bool {
	if e.Code == CodeRateLimited || e.StatusCode == http.StatusTooManyRequests {
		return e.Code != CodeQuotaExhausted
	}
	if e.Code == CodeUnavailable {
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
			Code       string `json:"code"`
			Message    string `json:"message"`
			Limit      string `json:"limit"`
			Used       int64  `json:"used"`
			LimitValue int64  `json:"limit_value"`
			ResetAt    int64  `json:"reset_at"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err == nil && env.Error != nil {
		e.Code = env.Error.Code
		e.Message = env.Error.Message
		e.Limit = env.Error.Limit
		e.Used = env.Error.Used
		e.LimitValue = env.Error.LimitValue
		if env.Error.ResetAt > 0 {
			e.ResetAt = time.Unix(env.Error.ResetAt, 0).UTC()
		}
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
	case http.StatusConflict:
		return CodeConflict
	case http.StatusTooManyRequests:
		return CodeRateLimited
	case http.StatusServiceUnavailable:
		return CodeUnavailable
	default:
		return CodeInternal
	}
}
