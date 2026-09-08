package teal

import "context"

// SettingsService reads and changes the account's settings. Everything here
// is free on every billing mode.
//
// The four PATCH endpoints write only the fields that are set, so their
// request types use pointers wherever leaving a field alone and setting it to
// its zero value are different things.
type SettingsService struct{ base }

// Retention returns the retention policy in force: what a full archive does,
// and how long a message stays in it at all.
//
// GET /v1/settings/retention.
func (s *SettingsService) Retention(ctx context.Context) (Retention, *Meta, error) {
	return s.get[Retention](ctx, "v1/settings/retention", nil)
}

// UpdateRetention changes the retention policy.
//
// The window and the deletion in the chat are paid settings: a plan that does
// not open one refuses the change rather than storing it quietly.
//
// PATCH /v1/settings/retention.
func (s *SettingsService) UpdateRetention(ctx context.Context, req RetentionUpdateRequest) (Retention, *Meta, error) {
	return s.patch[Retention](ctx, "v1/settings/retention", req)
}

// Vault returns whether a decrypted message is taken off the screen on its
// own, and after how long.
//
// GET /v1/settings/vault.
func (s *SettingsService) Vault(ctx context.Context) (VaultSettings, *Meta, error) {
	return s.get[VaultSettings](ctx, "v1/settings/vault", nil)
}

// UpdateVault sets how long a decrypted vault message stays in the Telegram
// chat before the bot takes it back.
//
// PATCH /v1/settings/vault.
func (s *SettingsService) UpdateVault(ctx context.Context, req VaultSettingsUpdateRequest) (VaultSettings, *Meta, error) {
	return s.patch[VaultSettings](ctx, "v1/settings/vault", req)
}

// Actions returns the prefix shortcuts are typed behind, which families of
// placeholder the plan opens, and how many shortcuts are left.
//
// GET /v1/settings/actions.
func (s *SettingsService) Actions(ctx context.Context) (ActionSettings, *Meta, error) {
	return s.get[ActionSettings](ctx, "v1/settings/actions", nil)
}

// UpdateActions sets the character a shortcut is typed behind.
//
// PATCH /v1/settings/actions.
func (s *SettingsService) UpdateActions(ctx context.Context, req ActionSettingsUpdateRequest) (ActionSettings, *Meta, error) {
	return s.patch[ActionSettings](ctx, "v1/settings/actions", req)
}

// Language returns what the account chose, what its Telegram client reports,
// and which of the two is in force.
//
// GET /v1/settings/language.
func (s *SettingsService) Language(ctx context.Context) (Language, *Meta, error) {
	return s.get[Language](ctx, "v1/settings/language", nil)
}

// UpdateLanguage sets the language every screen of the bot is drawn in.
//
// PATCH /v1/settings/language.
func (s *SettingsService) UpdateLanguage(ctx context.Context, req LanguageUpdateRequest) (Language, *Meta, error) {
	return s.patch[Language](ctx, "v1/settings/language", req)
}
