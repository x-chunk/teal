package teal

import "context"

// ActionsService manages the shortcuts an account keeps: the text a name
// expands to when it is typed behind the account's prefix.
type ActionsService struct{ base }

// List returns every shortcut the account has, the prefix they are typed
// behind, and how many more the plan allows.
//
// GET /v1/actions.
func (s *ActionsService) List(ctx context.Context) (ActionList, *Meta, error) {
	return s.get[ActionList](ctx, "v1/actions", nil)
}

// Create stores one shortcut.
//
// A body using a placeholder the plan does not open is refused here rather
// than surprising somebody mid-conversation — Placeholders says which are
// open. Under credits and hybrid a shortcut beyond the plan's ceiling is
// charged once, as actions:entry, when it is created.
//
// POST /v1/actions.
func (s *ActionsService) Create(ctx context.Context, req ActionCreateRequest) (Action, *Meta, error) {
	return s.post[Action](ctx, "v1/actions", req)
}

// Placeholders lists every placeholder the system knows and whether this
// account's plan opens it, so a client can refuse to write a body half of
// which would render as nothing.
//
// GET /v1/actions/placeholders.
func (s *ActionsService) Placeholders(ctx context.Context) ([]Placeholder, *Meta, error) {
	return s.get[[]Placeholder](ctx, "v1/actions/placeholders", nil)
}

// Get reads one shortcut by its id.
//
// GET /v1/actions/{id}.
func (s *ActionsService) Get(ctx context.Context, id int64) (Action, *Meta, error) {
	return s.get[Action](ctx, "v1/actions/"+itoa(id), nil)
}

// Update renames a shortcut, changes what it says, or both. Only the fields
// set are written, and the name is applied first, so a rename that collides
// fails before the body is touched.
//
// PATCH /v1/actions/{id}.
func (s *ActionsService) Update(ctx context.Context, id int64, req ActionUpdateRequest) (Action, *Meta, error) {
	return s.patch[Action](ctx, "v1/actions/"+itoa(id), req)
}

// Delete destroys one shortcut. The name it held becomes free again
// immediately.
//
// DELETE /v1/actions/{id}.
func (s *ActionsService) Delete(ctx context.Context, id int64) (*Meta, error) {
	_, meta, err := s.del[none](ctx, "v1/actions/"+itoa(id))
	return meta, err
}
