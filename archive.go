package teal

import (
	"context"
	"io"
	"strconv"

	"go.xchunk.org/anvil/v2/result"
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
// Every call is one query and is billed as one, whatever page it asks for.
// The answer carries a QueryID, and the query's other pages are free through
// SearchPage. Under shared limits a search spends one of the plan's daily
// searches; under hybrid the plan pays until that allowance is spent and
// credits pay after; under credits the allowance is not touched at all.
// Fields lists what a condition may name.
//
// POST /v1/messages/search.
func (s *ArchiveService) Search(ctx context.Context, req SearchRequest) result.Result[Response[SearchResult]] {
	return wrap(s.query[SearchResult](ctx, "v1/messages/search", req))
}

// SearchPage turns to another page of a query Search already ran, named by
// the QueryID its answer carried: the same chat and the same conditions, at
// the page asked for.
//
// It is free on every billing mode and spends no search — the query was paid
// for by its first page — though the plan's rules on the query apply to
// every page as they did to the first.
//
// A query can be paged for 30 minutes after it was run, by the application
// that ran it, which keeps its 20 most recent. After that — or after the
// service restarts — the answer is CodeNotFound, and the way on is to run
// the search again.
//
// POST /v1/messages/search/page.
func (s *ArchiveService) SearchPage(ctx context.Context, req SearchPageRequest) result.Result[Response[SearchResult]] {
	return wrap(s.query[SearchResult](ctx, "v1/messages/search/page", req))
}

// Count answers the same query with the number of matches and nothing else.
//
// It spends no search against the plan, exactly as counting does not in the
// bot. Under credits it is still a statement against the database and is
// billed as the query it is.
//
// POST /v1/messages/count.
func (s *ArchiveService) Count(ctx context.Context, req SearchRequest) result.Result[Response[CountResult]] {
	return wrap(s.query[CountResult](ctx, "v1/messages/count", req))
}

// Export streams every match as one JSON document, for a client to write
// straight to a file. It is the only endpoint that does not answer in the
// envelope, so the body comes back undecoded as Response.Data — and the
// caller closes it.
//
// The number of messages it carries is in the X-Aether-Export-Total header,
// reachable as Response.Meta.Header.Get("X-Aether-Export-Total"), and in the
// document's own Total field.
//
// The whole result is priced and charged before the first byte is written —
// export:run for the export and export:message for every message in it — so
// an application that cannot pay never starts an export. An export is capped
// at 50 MB, and one that would pass the cap is refused whole with
// CodeTooLarge before a byte is sent, and nothing is charged.
//
// The charge becomes final when the document has gone out. One that fails on
// the server's side after it started is refunded and the connection is
// broken off, so reading the body fails rather than ending as if the document
// were whole: a cut-off export never decodes as a complete one. An export the
// caller stops reading is charged if any of it arrived. Decode it into an
// ExportDocument when a file is not what is wanted.
//
// The weekly export allowance and the plan's ceiling on an export's size
// apply under shared limits; under credits neither does, and every message is
// paid for.
//
// POST /v1/messages/export.
func (s *ArchiveService) Export(ctx context.Context, req SearchRequest) result.Result[Response[io.ReadCloser]] {
	return wrap(s.postRaw(ctx, "v1/messages/export", req))
}

// Chats returns one page of the conversations the archive holds, with how
// many messages each carries. A nil request asks for the first page.
//
// GET /v1/chats.
func (s *ArchiveService) Chats(ctx context.Context, req *ChatsRequest) result.Result[Response[ChatList]] {
	return wrap(s.get[ChatList](ctx, "v1/chats", req.query()))
}

// Message reads one message by the archive's own id, which is what a search
// result carries in its ID field. It is not Telegram's message id, which is
// only unique inside a chat.
//
// GET /v1/messages/{id}.
func (s *ArchiveService) Message(ctx context.Context, id int64) result.Result[Response[Message]] {
	return wrap(s.get[Message](ctx, "v1/messages/"+itoa(id), nil))
}

// Versions returns every version of a message's text, latest first. The first
// entry is the text the message carries now; how far back the rest goes is a
// plan limit, and a plan without the edit history refuses with
// CodeForbidden.
//
// GET /v1/messages/{id}/versions.
func (s *ArchiveService) Versions(ctx context.Context, id int64) result.Result[Response[VersionList]] {
	return wrap(s.get[VersionList](ctx, "v1/messages/"+itoa(id)+"/versions", nil))
}

// Fields lists every column a Condition may name, and the match modes each
// accepts. It is generated from the same registry the SQL builder checks
// against, so a field listed here is a field that works and one that is not
// listed never reaches the database.
//
// GET /v1/fields.
func (s *ArchiveService) Fields(ctx context.Context) result.Result[Response[[]Field]] {
	return wrap(s.get[[]Field](ctx, "v1/fields", nil))
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
