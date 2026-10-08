package teal

import (
	"context"

	"go.xchunk.org/anvil/v2/result"
)

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
// account's plan buys. A queue with no room answers CodeBusy before any of
// the work began, and the transport sends the call again after RetryAfter,
// writes included.
//
// Everything but Reveal writes, and a key writes to the vault only once its
// owner has opened ScopeVaultWrite to it in the bot; until then the answer is
// CodeScopeRequired and nothing is charged.
//
// Too many wrong passphrases and codes shut the vault to guesses for a while:
// the answer is CodeRateLimited with ReasonVaultLocked and a RetryAfter for
// when the lock runs out, and it is not retried — a guess sent the moment the
// lock lifts is a guess that may lock it again.
type VaultService struct{ base }

// Store encrypts one secret under a passphrase and returns the id of the new
// entry with the one-time recovery codes issued with it. The codes are
// readable here and never again; the id is what DeleteByID takes, and opens
// nothing.
//
// Under shared limits the plan's ceiling on entries applies. Under credits
// and hybrid an entry beyond that ceiling is charged once, as vault:entry,
// when it is written.
//
// POST /v1/vault/entries.
func (s *VaultService) Store(ctx context.Context, req VaultStoreRequest) result.Result[Response[RecoveryCodes]] {
	return wrap(s.post[RecoveryCodes](ctx, "v1/vault/entries", req))
}

// Reveal decrypts the entry a passphrase addresses and returns its plaintext.
//
// POST /v1/vault/entries/reveal.
func (s *VaultService) Reveal(ctx context.Context, req VaultRevealRequest) result.Result[Response[VaultSecret]] {
	return wrap(s.post[VaultSecret](ctx, "v1/vault/entries/reveal", req))
}

// Rename re-wraps an entry's key under a new passphrase. The secret itself is
// not re-encrypted and its data key does not change — only the door onto it
// does. Renaming is a plan feature: a plan that does not open it refuses with
// CodeForbidden.
//
// POST /v1/vault/entries/rename.
func (s *VaultService) Rename(ctx context.Context, req VaultRenameRequest) result.Result[Response[struct{}]] {
	return wrap(s.post[none](ctx, "v1/vault/entries/rename", req))
}

// ReissueCodes throws away whatever recovery codes an entry had and issues a
// fresh set. It is what to call when a set may have leaked: the old ones are
// worthless the moment this returns.
//
// POST /v1/vault/entries/codes.
func (s *VaultService) ReissueCodes(ctx context.Context, req VaultCodesRequest) result.Result[Response[RecoveryCodes]] {
	return wrap(s.post[RecoveryCodes](ctx, "v1/vault/entries/codes", req))
}

// Recover opens an entry with one of its one-time codes and moves it to the
// new passphrase the request carries, which is required. The code is spent by
// this call, every remaining code is replaced with a fresh set, and from then
// on the entry opens with the new passphrase.
//
// Spending a code raises an alert to the account's owner in Telegram. It is
// published unconditionally: nothing on this path can suppress it.
//
// POST /v1/vault/entries/recover.
func (s *VaultService) Recover(ctx context.Context, req VaultRecoverRequest) result.Result[Response[VaultRecovered]] {
	return wrap(s.post[VaultRecovered](ctx, "v1/vault/entries/recover", req))
}

// Delete destroys the entry a passphrase addresses. The delete is a real one:
// the row is destroyed rather than marked, which is what frees the passphrase
// to be used again.
//
// Free on every billing mode — nothing should ever make somebody think twice
// about deleting their own secret.
//
// POST /v1/vault/entries/delete.
func (s *VaultService) Delete(ctx context.Context, req VaultDeleteRequest) result.Result[Response[struct{}]] {
	return wrap(s.post[none](ctx, "v1/vault/entries/delete", req))
}

// DeleteByID destroys one entry by the id Store answered with, in
// RecoveryCodes.EntryID. Free on every billing mode, like Delete.
//
// DELETE /v1/vault/entries/{id}.
func (s *VaultService) DeleteByID(ctx context.Context, id int64) result.Result[Response[struct{}]] {
	return wrap(s.del[none](ctx, "v1/vault/entries/"+itoa(id)))
}
