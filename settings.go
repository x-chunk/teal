package teal

import (
	"context"

	"go.xchunk.org/anvil/v2/result"
)

// SettingsService reads and changes the account's settings. Everything here
// is free on every billing mode.
//
// The four PATCH endpoints write only the fields that are set, so their
// request types use pointers wherever leaving a field alone and setting it to
// its zero value are different things. They change things only once the
// key's owner has opened ScopeSettingsWrite to it in the bot — a retention
// policy can delete messages out of real chats in Telegram — and until then
// the answer is CodeScopeRequired. Reading needs no scope.
type SettingsService struct{ base }

// Retention returns the retention policy in force: what a full archive does,
// how long a message stays in it at all, and the windows that may be set.
//
// GET /v1/settings/retention.
func (s *SettingsService) Retention(ctx context.Context) result.Result[Response[Retention]] {
	return wrap(s.get[Retention](ctx, "v1/settings/retention", nil))
}

// UpdateRetention changes the retention policy. The fields set are applied
// together: a request refused on one of them changes none.
//
// The window and the deletion in the chat are paid settings: a plan that does
// not open one refuses the change rather than storing it quietly. The window
// is one of Retention.TTLChoices, and any other number is refused with
// CodeBadRequest. A request that sets nothing is refused too.
//
// PATCH /v1/settings/retention.
func (s *SettingsService) UpdateRetention(ctx context.Context, req RetentionUpdateRequest) result.Result[Response[Retention]] {
	return wrap(s.patch[Retention](ctx, "v1/settings/retention", req))
}

// Vault returns whether a decrypted message is taken off the screen on its
// own, after how long, and the windows that may be set.
//
// GET /v1/settings/vault.
func (s *SettingsService) Vault(ctx context.Context) result.Result[Response[VaultSettings]] {
	return wrap(s.get[VaultSettings](ctx, "v1/settings/vault", nil))
}

// UpdateVault sets how long a decrypted vault message stays in the Telegram
// chat before the bot takes it back: one of VaultSettings.RevealTTLChoices,
// or zero for the default. Any other number is refused with CodeBadRequest,
// and a plan that does not open the setting refuses with CodeForbidden.
//
// PATCH /v1/settings/vault.
func (s *SettingsService) UpdateVault(ctx context.Context, req VaultSettingsUpdateRequest) result.Result[Response[VaultSettings]] {
	return wrap(s.patch[VaultSettings](ctx, "v1/settings/vault", req))
}

// Actions returns the prefix shortcuts are typed behind, which families of
// placeholder the plan opens, and how many shortcuts are left.
//
// GET /v1/settings/actions.
func (s *SettingsService) Actions(ctx context.Context) result.Result[Response[ActionSettings]] {
	return wrap(s.get[ActionSettings](ctx, "v1/settings/actions", nil))
}

// UpdateActions sets the character a shortcut is typed behind.
//
// PATCH /v1/settings/actions.
func (s *SettingsService) UpdateActions(ctx context.Context, req ActionSettingsUpdateRequest) result.Result[Response[ActionSettings]] {
	return wrap(s.patch[ActionSettings](ctx, "v1/settings/actions", req))
}

// Language returns what the account chose, what its Telegram client reports,
// and which of the two is in force.
//
// GET /v1/settings/language.
func (s *SettingsService) Language(ctx context.Context) result.Result[Response[Language]] {
	return wrap(s.get[Language](ctx, "v1/settings/language", nil))
}

// UpdateLanguage sets the language every screen of the bot is drawn in.
//
// PATCH /v1/settings/language.
func (s *SettingsService) UpdateLanguage(ctx context.Context, req LanguageUpdateRequest) result.Result[Response[Language]] {
	return wrap(s.patch[Language](ctx, "v1/settings/language", req))
}
