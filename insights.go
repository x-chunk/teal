package teal

// InsightsService covers GET /v1/insights and GET /v1/portraits/{chat}.
//
// A portrait that has not been built yet answers CodeNotFound with a
// Retry-After: the model fits in the background and nothing is charged for an
// answer that was not given.
type InsightsService struct{ base }
