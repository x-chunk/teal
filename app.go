package teal

import (
	"context"
)

// AppService covers GET /v1/app, /v1/usage, /v1/account, /v1/quotas and
// /v1/prices. All free on every billing mode.
type AppService struct{ base }

func (s *AppService) Get(ctx context.Context) (Application, *Meta, error) {
	return s.get[Application](ctx, "v1/app", nil)
}
