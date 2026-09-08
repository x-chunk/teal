package teal

// VaultService covers POST /v1/vault/entries, /reveal, /rename, /codes,
// /recover and /delete, and DELETE /v1/vault/entries/{id}.
//
// A passphrase travels in a body and never in a path or a query, which is why
// deleting by passphrase is a POST. A wrong passphrase answers CodeNotFound,
// deliberately the same as one that addresses nothing.
type VaultService struct{ base }
