package teal

import "context"

// AppService reads the application, the account behind it and the deployment's
// price list. Everything here is free on every billing mode: an application
// that has run out of credits must still be able to find out that it has.
type AppService struct{ base }

// Get returns the application this key belongs to — its name, its billing
// mode, the balance it has left and what it has spent.
//
// The balance is read at the moment of the call; the X-Aether-Balance header
// on every metered response is the same number, one request fresher.
//
// GET /v1/app.
func (s *AppService) Get(ctx context.Context) (Application, *Meta, error) {
	return s.get[Application](ctx, "v1/app", nil)
}

// Usage returns what this application has spent over a window, folded into
// one row per priced operation. A nil request leaves the API's defaults: the
// last 30 days, without the day-by-day breakdown.
//
// GET /v1/usage.
func (s *AppService) Usage(ctx context.Context, req *UsageRequest) (Usage, *Meta, error) {
	return s.get[Usage](ctx, "v1/usage", req.query())
}

// Account returns the account this key opens: the plan behind it, what it
// opens, when it lapses, the account's own balance and every quota measured
// against what has been used of it.
//
// GET /v1/account.
func (s *AppService) Account(ctx context.Context) (Account, *Meta, error) {
	return s.get[Account](ctx, "v1/account", nil)
}

// Quotas returns the quotas of Account on their own, for a client that polls
// them. Under shared limits they say how much room is left before a call is
// refused; under hybrid, how much room is left before calls start costing
// credits.
//
// GET /v1/quotas.
func (s *AppService) Quotas(ctx context.Context) ([]Quota, *Meta, error) {
	return s.get[[]Quota](ctx, "v1/quotas", nil)
}

// Prices returns what every operation costs, what one cent of account balance
// buys, and the rates the shield enforces. It is read from the running
// configuration, so it is what is actually in force on this deployment.
//
// GET /v1/prices.
func (s *AppService) Prices(ctx context.Context) (Prices, *Meta, error) {
	return s.get[Prices](ctx, "v1/prices", nil)
}
