package teal

import "context"

// InsightsService reads what the archive adds up to.
type InsightsService struct{ base }

// Get returns At a Glance: a handful of sentences about what the archive
// holds — how much of it there is, what is unusual in it, and what is about
// to run out.
//
// GET /v1/insights.
func (s *InsightsService) Get(ctx context.Context) (Insights, *Meta, error) {
	return s.get[Insights](ctx, "v1/insights", nil)
}

// Portrait reads a whole conversation and describes the person on the other
// side of it: the archetype they fall into, the stylometric traits behind
// that, what they write about and when.
//
// A portrait that has not been built yet answers CodeNotFound with a
// Retry-After — the model fits in the background, and nothing is charged for
// an answer that was not given. Treat that as "ask again in
// Meta.RetryAfter", not as a chat that has none.
//
// Under shared limits and hybrid, a chat already portrayed this week is free
// to ask for again: the plan counts chats, not requests. Under credits every
// request is charged, including a repeat of the same chat.
//
// GET /v1/portraits/{chat}.
func (s *InsightsService) Portrait(ctx context.Context, chat int64) (Portrait, *Meta, error) {
	return s.get[Portrait](ctx, "v1/portraits/"+itoa(chat), nil)
}
