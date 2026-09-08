package teal_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/x-chunk/teal"
)

func ExampleNew() {
	c, err := teal.New(os.Getenv("AETHER_KEY"), teal.WithBaseURL("https://your-aether-host"))
	if err != nil {
		log.Fatal(err)
	}

	app, _, err := c.App.Get(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s is on %s with %s left\n", app.Name, app.Billing, app.Balance.Credits)
}

// Searching, and reading what the call cost out of the Meta.
func ExampleArchiveService_Search() {
	c, err := teal.New(os.Getenv("AETHER_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	res, meta, err := c.Archive.Search(context.Background(), teal.SearchRequest{
		Conditions: []teal.Condition{
			{Field: "text", Mode: teal.MatchContains, Value: "invoice"},
			{Field: "created", Mode: teal.MatchEquals, Conn: teal.ConnAnd, Value: "2026-09-01"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d matches over %d pages, cost %s\n", res.Total, res.Pages, meta.Cost)
	if meta.HasBalance {
		fmt.Printf("%s left on the application\n", meta.Balance)
	}
}

// A quota that is spent turns only when its window does, so a client waits
// rather than retries. The transport already retries what is worth retrying.
func ExampleIsCode() {
	c, err := teal.New(os.Getenv("AETHER_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = c.Archive.Search(context.Background(), teal.SearchRequest{})
	switch {
	case teal.IsCode(err, teal.CodeQuotaExhausted):
		var e *teal.Error
		errors.As(err, &e)
		fmt.Printf("%s is spent until %s\n", e.Limit, e.ResetAt.Format(time.RFC3339))
	case teal.IsCode(err, teal.CodeInsufficientCredit):
		fmt.Println("top the application up from the bot")
	case err != nil:
		log.Fatal(err)
	}
}

// Only the fields that are set are written, which is why they are pointers:
// zero turns the window off, and is not the same as leaving it alone.
func ExampleSettingsService_UpdateRetention() {
	c, err := teal.New(os.Getenv("AETHER_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	mode := teal.RetentionRotate
	ttl := int64(30 * 24 * time.Hour / time.Second)

	r, _, err := c.Settings.UpdateRetention(context.Background(), teal.RetentionUpdateRequest{
		Mode:       &mode,
		TTLSeconds: &ttl,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(r.Mode, r.TTLSeconds)
}

// The export is the one endpoint that answers with a document rather than the
// envelope, so it hands back a body for the caller to close.
func ExampleArchiveService_Export() {
	c, err := teal.New(os.Getenv("AETHER_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	body, meta, err := c.Archive.Export(context.Background(), teal.SearchRequest{})
	if err != nil {
		log.Fatal(err)
	}
	defer body.Close()

	f, err := os.Create("export.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if _, err := f.ReadFrom(body); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", meta.Header.Get("X-Aether-Export-Total"), "messages")
}
