package teal

import (
	"context"

	"go.xchunk.org/anvil/v2/result"
)

// ActionsService manages the shortcuts an account keeps: the text a name
// expands to when it is typed behind the account's prefix.
type ActionsService struct{ base }

// List returns every shortcut the account has, the prefix they are typed
// behind, and how many more the plan allows.
//
// GET /v1/actions.
func (s *ActionsService) List(ctx context.Context) result.Result[Response[ActionList]] {
	return wrap(s.get[ActionList](ctx, "v1/actions", nil))
}

// Create stores one shortcut.
//
// A body using a placeholder the plan does not open is refused here rather
// than surprising somebody mid-conversation — Placeholders says which are
// open. So is a body longer than 900 UTF-16 code units, or one carrying a
// NUL, with CodeBadRequest. Under credits and hybrid a shortcut beyond the
// plan's ceiling is charged once, as actions:entry, when it is created.
//
// POST /v1/actions.
func (s *ActionsService) Create(ctx context.Context, req ActionCreateRequest) result.Result[Response[Action]] {
	return wrap(s.post[Action](ctx, "v1/actions", req))
}

// Placeholders lists every placeholder the system knows and whether this
// account's plan opens it, so a client can refuse to write a body half of
// which would render as nothing.
//
// GET /v1/actions/placeholders.
func (s *ActionsService) Placeholders(ctx context.Context) result.Result[Response[[]Placeholder]] {
	return wrap(s.get[[]Placeholder](ctx, "v1/actions/placeholders", nil))
}

// Get reads one shortcut by its id.
//
// GET /v1/actions/{id}.
func (s *ActionsService) Get(ctx context.Context, id int64) result.Result[Response[Action]] {
	return wrap(s.get[Action](ctx, "v1/actions/"+itoa(id), nil))
}

// Update renames a shortcut, changes what it says, switches whether it needs
// its arguments, or any of them at once. Only the fields set are written, and
// all of them in one write: a request refused on one field — a rename that
// collides, a body that is refused — changes none of them, and one that
// succeeds changes them together, so a body and its ArgsRequired never apply
// apart. The answer is the shortcut as it now stands.
//
// A request that sets nothing is refused with CodeBadRequest.
//
// PATCH /v1/actions/{id}.
func (s *ActionsService) Update(ctx context.Context, id int64, req ActionUpdateRequest) result.Result[Response[Action]] {
	return wrap(s.patch[Action](ctx, "v1/actions/"+itoa(id), req))
}

// Delete destroys one shortcut. The name it held becomes free again
// immediately.
//
// DELETE /v1/actions/{id}.
func (s *ActionsService) Delete(ctx context.Context, id int64) result.Result[Response[struct{}]] {
	return wrap(s.del[none](ctx, "v1/actions/"+itoa(id)))
}
