package teal

import (
	"context"

	"go.xchunk.org/anvil/v2/result"
)

// InsightsService reads what the archive adds up to.
type InsightsService struct{ base }

// Get returns At a Glance: a handful of sentences about what the archive
// holds — how much of it there is, what is unusual in it, and what is about
// to run out.
//
// The insights are built on workers the bot shares with the API, one answer
// at a time. A request that cannot get its turn within two seconds is
// answered CodeBusy with a RetryAfter, and the transport asks again.
//
// GET /v1/insights.
func (s *InsightsService) Get(ctx context.Context) result.Result[Response[Insights]] {
	return wrap(s.get[Insights](ctx, "v1/insights", nil))
}

// Portrait reads a whole conversation and describes the person on the other
// side of it: the archetype they fall into, the stylometric traits behind
// that, what they write about, when, and which of the account's other
// conversations sound like it.
//
// A portrait that cannot be returned says why in Error.Reason, and nothing is
// charged for an answer that was not given:
//
//   - ReasonModelTraining, under CodeUnavailable: the model is still being
//     fitted. Ask again after Error.RetryAfter; the transport does not wait
//     that long on its own.
//   - ReasonNotEnoughHistory or ReasonEmptyChat, under CodeNotFound: the
//     chat has to grow first.
//   - ReasonPortraitsDisabled, under CodeUnavailable: portraits are switched
//     off on this deployment, and waiting does not help.
//
// Under shared limits and hybrid, a chat already portrayed this week is free
// to ask for again: the plan counts chats, not requests. Under credits every
// request is charged, including a repeat of the same chat.
//
// GET /v1/portraits/{chat}.
func (s *InsightsService) Portrait(ctx context.Context, chat int64) result.Result[Response[Portrait]] {
	return wrap(s.get[Portrait](ctx, "v1/portraits/"+itoa(chat), nil))
}
