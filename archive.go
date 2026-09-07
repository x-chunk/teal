package teal

// ArchiveService covers POST /v1/messages/search, /count and /export, and
// GET /v1/chats, /v1/messages/{id}, /v1/messages/{id}/versions and
// /v1/fields.
//
// Nothing here writes: a message enters the archive by being sent and by no
// other route. The export does not answer in the envelope — reach for DoRaw.
type ArchiveService struct{ c *Client }
