package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/feeds"
	"github.com/sams96/bookmark-feeder/sync"
)

func handleGet(sc *sync.SyncClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookmarks, err := sc.GetBookmarks()
		if err != nil {
			slog.Error("An error occurred while fetching bookmarks", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bookmarks = sync.FilterBookmarks(bookmarks, "Reading List")

		feed := &feeds.Feed{
			Title:       "Exported Bookmarks Feed",
			Link:        &feeds.Link{Href: "http://example.com/bookmarks.atom"},
			Description: "A collection of bookmarks exported to Atom feed format.",
			Author:      &feeds.Author{Name: "Go Bookmark Exporter"},
			Id:          "urn:uuid:" + uuid.NewString(),
		}

		for _, record := range bookmarks {
			feed.Items = append(feed.Items, &feeds.Item{
				Title:       record.Title,
				Link:        &feeds.Link{Href: record.URI},
				Description: fmt.Sprintf("Bookmark: <a href=\"%s\">%s</a>", record.URI, record.Title),
				Created:     *record.DateAdded,
				Id:          "urn:bookmark:" + record.URI,
			})
		}

		// Generate the Atom XML string
		atomFeedXML, err := feed.ToAtom()
		if err != nil {
			fmt.Println("Error generating Atom feed:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(atomFeedXML))
	})
}

func main() {
	sc, err := sync.NewSyncClient("./firefox-sync-session.json")
	if err != nil {
		slog.Error("Failed to create new SyncClient", "error", err)
		os.Exit(1)
	}

	email := os.Getenv("BOOKMARK_FEEDER_SYNC_EMAIL")
	password := os.Getenv("BOOKMARK_FEEDER_SYNC_PASSWORD")

	err = sc.Login(email, password)
	if err != nil {
		slog.Error("An error occurred during authentication", "error", err)
		os.Exit(1)
	}

	fmt.Println("Successfully logged in and authenticated!")

	s := &http.Server{
		Addr:           ":19928",
		Handler:        handleGet(sc),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
