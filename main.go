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
	"github.com/kelseyhightower/envconfig"
	"github.com/sams96/bookmark-feeder/sync"
)

type config struct {
	SyncEmail    string `envconfig:"SYNC_EMAIL"`
	SyncPassword string `envconfig:"SYNC_PASSWORD"`

	ServerAddress string `envconfig:"SERVER_ADDRESS" default:":19928"`

	FeedTitle       string `envconfig:"FEED_TITLE" default:"Bookmarks Feed"`
	FeedLink        string `envconfig:"FEED_LINK"`
	FeedDescription string `envconfig:"FEED_DESC" default:"A collection of bookmarks exported to Atom feed format."`
	FeedAuthor      string `envconfig:"FEED_AUTHOR" default:"Bookmark Feeder"`

	BookmarkFolder string `envconfig:"BOOKMARK_FOLDER"`
}

func handleGet(sc *sync.SyncClient, config config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookmarks, err := sc.GetBookmarks()
		if err != nil {
			slog.Error("An error occurred while fetching bookmarks", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if config.BookmarkFolder != "" {
			bookmarks = sync.FilterBookmarks(bookmarks, config.BookmarkFolder)
		}

		feed := &feeds.Feed{
			Title:       config.FeedTitle,
			Link:        &feeds.Link{Href: config.FeedLink},
			Description: config.FeedDescription,
			Author:      &feeds.Author{Name: config.FeedAuthor},
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
	var config config
	err := envconfig.Process("BOOKMARK_FEEDER", &config)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	sc, err := sync.NewSyncClient("./firefox-sync-session.json")
	if err != nil {
		slog.Error("Failed to create new SyncClient", "error", err)
		os.Exit(1)
	}

	err = sc.Login(config.SyncEmail, config.SyncPassword)
	if err != nil {
		slog.Error("An error occurred during authentication", "error", err)
		os.Exit(1)
	}

	fmt.Println("Successfully logged in and authenticated!")

	s := &http.Server{
		Addr:           config.ServerAddress,
		Handler:        handleGet(sc, config),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
