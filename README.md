# Bookmark Feeder

Bookmark feeder is a service that loads bookmarks from Firefox Sync and exposes
them as an atom feed.

## Usage

To run with docker-compose

```yaml
version: "3.4"

services:
  bookmark-feeder:
    image: ghcr.io/sams96/bookmark-feeder:latest
    container_name: bookmark-feeder
    restart: unless-stopped
    ports:
      - "19928:19928"
    env_file: ".env"
```

or as a plain docker command

```bash
docker run --name bookmark-feeder --restart unless-stopped -p 19928:19928 --env-file .env ghcr.io/sams96/bookmark-feeder:latest
```

These fields are required in your `.env` file.

```env
BOOKMARK_FEEDER_SYNC_EMAIL=<sync email>
BOOKMARK_FEEDER_SYNC_PASSWORD=<sync password>
BOOKMARK_FEEDER_BOOKMARK_FOLDER=<bookmark folder to extract from>
```

And these are optional, the defaults are included here.

```env
BOOKMARK_FEEDER_SERVER_ADDRESS=":19928"
BOOKMARK_FEEDER_FEED_TITLE="Bookmarks Feed"
BOOKMARK_FEEDER_FEED_LINK=""
BOOKMARK_FEEDER_FEED_DESC="A collection of bookmarks exported to Atom feed format."
BOOKMARK_FEEDER_FEED_AUTHOR="Bookmark Feeder"
```

Once set up, just add `<your server address>:19928` to your feed reader of
choice and enjoy.
