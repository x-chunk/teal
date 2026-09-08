package teal

import (
	"context"
	"io"
	"strconv"
)

// ArchiveService reads the archive.
//
// Nothing here writes to it. The archive is what Aether observed in a chat: a
// message enters it by being sent and by no other route, and an API that
// could put one there would make the archive a thing somebody could forge
// rather than a record.
type ArchiveService struct{ base }

// Search runs one query against the archived messages and returns a page of
// matches.
//
// Every call is one query and is billed as one: there is no free paging here,
// because a page is a fresh statement against the database rather than a
// redraw of an answer already paid for. Under shared limits it spends one of
// the plan's daily searches; under credits the allowance is not touched at
// all. Fields lists what a condition may name.
//
// POST /v1/messages/search.
func (s *ArchiveService) Search(ctx context.Context, req SearchRequest) (SearchResult, *Meta, error) {
	return s.query[SearchResult](ctx, "v1/messages/search", req)
}

// Count answers the same query with the number of matches and nothing else.
//
// It spends no search against the plan, exactly as counting does not in the
// bot. Under credits it is still a statement against the database and is
// billed as the query it is.
//
// POST /v1/messages/count.
func (s *ArchiveService) Count(ctx context.Context, req SearchRequest) (CountResult, *Meta, error) {
	return s.query[CountResult](ctx, "v1/messages/count", req)
}

// Export streams every match as one JSON document, for a client to write
// straight to a file. It is the only endpoint that does not answer in the
// envelope, so the body comes back undecoded — and the caller closes it.
//
// The number of messages it carries is in the X-Aether-Export-Total header,
// reachable as Meta.Header.Get("X-Aether-Export-Total"), and in the
// document's own Total field.
//
// The whole result is priced and charged before the first byte is written, so
// an application that cannot pay never starts an export. An export is capped
// at 50 MB; a stream that hits the cap ends early and the charge for it is
// given back. Decode it into an ExportDocument when a file is not what is
// wanted.
//
// POST /v1/messages/export.
func (s *ArchiveService) Export(ctx context.Context, req SearchRequest) (io.ReadCloser, *Meta, error) {
	return s.postRaw(ctx, "v1/messages/export", req)
}

// Chats returns one page of the conversations the archive holds, with how
// many messages each carries. A nil request asks for the first page.
//
// GET /v1/chats.
func (s *ArchiveService) Chats(ctx context.Context, req *ChatsRequest) (ChatList, *Meta, error) {
	return s.get[ChatList](ctx, "v1/chats", req.query())
}

// Message reads one message by the archive's own id, which is what a search
// result carries in its ID field. It is not Telegram's message id, which is
// only unique inside a chat.
//
// GET /v1/messages/{id}.
func (s *ArchiveService) Message(ctx context.Context, id int64) (Message, *Meta, error) {
	return s.get[Message](ctx, "v1/messages/"+itoa(id), nil)
}

// Versions returns every version of a message's text, latest first. The first
// entry is the text the message carries now; how far back the rest goes is a
// plan limit.
//
// GET /v1/messages/{id}/versions.
func (s *ArchiveService) Versions(ctx context.Context, id int64) (VersionList, *Meta, error) {
	return s.get[VersionList](ctx, "v1/messages/"+itoa(id)+"/versions", nil)
}

// Fields lists every column a Condition may name, and the match modes each
// accepts. It is generated from the same registry the SQL builder checks
// against, so a field listed here is a field that works and one that is not
// listed never reaches the database.
//
// GET /v1/fields.
func (s *ArchiveService) Fields(ctx context.Context) ([]Field, *Meta, error) {
	return s.get[[]Field](ctx, "v1/fields", nil)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
