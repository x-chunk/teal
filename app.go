package teal

// AppService covers GET /v1/app, /v1/usage, /v1/account, /v1/quotas and
// /v1/prices. All free on every billing mode.
type AppService struct{ base }

func 