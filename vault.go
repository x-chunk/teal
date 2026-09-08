package teal

import "context"

// VaultService stores and opens secrets.
//
// A passphrase is never stored: a hash of it addresses an entry, and the key
// that decrypts the entry is derived from it. Nobody, this system included,
// can open an entry without its passphrase or one of its recovery codes.
//
// Two consequences run through the whole service. A passphrase travels in a
// body and never in a path or a query, because a path is written into the
// access log of every proxy there has ever been — which is why deleting by
// passphrase is a POST. And a passphrase that addresses nothing answers
// CodeNotFound, deliberately the same answer as one that is wrong.
//
// Every call here waits its turn in the vault's queue, at the class the
// account's plan buys.
type VaultService struct{ base }

// Store encrypts one secret under a passphrase and returns the one-time
// recovery codes issued with it. The codes are readable here and never again.
//
// Under shared limits the plan's ceiling on entries applies. Under credits
// and hybrid an entry beyond that ceiling is charged once, as vault:entry,
// when it is written.
//
// POST /v1/vault/entries.
func (s *VaultService) Store(ctx context.Context, req VaultStoreRequest) (RecoveryCodes, *Meta, error) {
	return s.post[RecoveryCodes](ctx, "v1/vault/entries", req)
}

// Reveal decrypts the entry a passphrase addresses and returns its plaintext.
//
// POST /v1/vault/entries/reveal.
func (s *VaultService) Reveal(ctx context.Context, req VaultRevealRequest) (VaultSecret, *Meta, error) {
	return s.post[VaultSecret](ctx, "v1/vault/entries/reveal", req)
}

// Rename re-wraps an entry's key under a new passphrase. The secret itself is
// not re-encrypted and its data key does not change — only the door onto it
// does.
//
// POST /v1/vault/entries/rename.
func (s *VaultService) Rename(ctx context.Context, req VaultRenameRequest) (*Meta, error) {
	_, meta, err := s.post[none](ctx, "v1/vault/entries/rename", req)
	return meta, err
}

// ReissueCodes throws away whatever recovery codes an entry had and issues a
// fresh set. It is what to call when a set may have leaked: the old ones are
// worthless the moment this returns.
//
// POST /v1/vault/entries/codes.
func (s *VaultService) ReissueCodes(ctx context.Context, req VaultCodesRequest) (RecoveryCodes, *Meta, error) {
	return s.post[RecoveryCodes](ctx, "v1/vault/entries/codes", req)
}

// Recover opens an entry with one of its one-time codes, optionally moving it
// to a new passphrase on the way. The code is spent by this call either way,
// and every remaining code is replaced with a fresh set.
//
// Spending a code raises an alert to the account's owner in Telegram. It is
// published unconditionally: nothing on this path can suppress it.
//
// POST /v1/vault/entries/recover.
func (s *VaultService) Recover(ctx context.Context, req VaultRecoverRequest) (VaultRecovered, *Meta, error) {
	return s.post[VaultRecovered](ctx, "v1/vault/entries/recover", req)
}

// Delete destroys the entry a passphrase addresses. The delete is a real one:
// the row is destroyed rather than marked, which is what frees the passphrase
// to be used again.
//
// Free on every billing mode — nothing should ever make somebody think twice
// about deleting their own secret.
//
// POST /v1/vault/entries/delete.
func (s *VaultService) Delete(ctx context.Context, req VaultDeleteRequest) (*Meta, error) {
	_, meta, err := s.post[none](ctx, "v1/vault/entries/delete", req)
	return meta, err
}

// DeleteByID destroys one entry by the id it was stored under, for a client
// that kept it.
//
// DELETE /v1/vault/entries/{id}.
func (s *VaultService) DeleteByID(ctx context.Context, id int64) (*Meta, error) {
	_, meta, err := s.del[none](ctx, "v1/vault/entries/"+itoa(id))
	return meta, err
}
