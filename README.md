# alnvdl/varys

Varys is a barebones RSS reader written in Go. It has (practically) no external
dependencies and provides an equally barebones web experience. It is meant to
be self-hosted and used by a single user.

Varys is based on an in-memory feed list that can be configured to be persisted
to the disk and auto-refreshed. It also serves this feed list over an HTTP API,
and provides a static mobile-friendly SPA for reading feeds.

Varys is named after the GoT character who apparently was always up-to-date
with his RSS feeds.

## Running
To run Varys locally, just install Go 1.24+ and run `make dev_example`.

If you have a list of feeds in `feeds.json` (see the
[Feed list format](#feed-list-format) below), you can run `make dev` to use it.

## Feed list format
The feed list is a JSON array where each feed is represented as an object:
```jsonc
[
   { /* xml feed */ }
   { /* html feed */ }
   { /* image feed */ }
]
```

These are the supported feed types and their accepted parameters.

### Atom and RSS feeds (type `xml`)
This type of feed can be used with traditional RSS or Atom XML feeds.
```json
{
  "type": "xml",
  "name": "Example XML Feed",
  "url": "https://example.com/rss"
}
```

### HTML pages (type `html`)
This type of feed can be used to simulate feeds based on the content of HTML
pages.
```jsonc
{
  "type": "html",
  "name": "Example HTML Feed",
  "url": "https://example.com/news",
  "params": {
    "encoding": "ISO-8859-1", // Only required if not UTF-8.
    // container_tag and container_attrs are used to define the elements where
    // anchors will be sourced from.
    "container_tag": "div",
    "container_attrs": {
      "class": "news-container"
    },
    "title_pos": 0, // Identifies the position of the title in the extracted
                    // content.
    "base_url": "https://example.com/", // The base URL to use for resolving
    //                                 // relative URLs.
    "allowed_prefixes": [
      // Only items with URLs starting with these prefixes (after resolving
      // them with base_url) will be picked up.
      "https://example.com/news/"
    ]
  }
}
```

### Image feeds (type `image`)
This type of feed can be used for images that are updated frequently (e.g.,
hosted webcam images or weather report charts).
```json
{
  "type": "img",
  "name": "Example Image Feed",
  "url": "https://example.com/image.png",
  "params": {
    "title": "Example Image", // The title of the items, to which a timestamp
                              // will be appended.
    "url": "https://example.com/image.png", // The URL the user should be taken
                                            // to when opening an item from
                                            // this feed.
    "mime_type": "image/png" // The type of image returned by the feed URL.
  }
}
```

## Environment variables
The following environment variables can be used to configure Varys:

- `ACCESS_TOKEN`: A random secret value used for authentication.
   This variable is required.
- `SESSION_KEY`: A random secret value used for signing session cookies.
   If not provided, a random key will be generated on every initialization.
- `DB_PATH`: The path to the database file. Default is `db.json`.
- `FEEDS`: The JSON content of your feed list.
   This is optional, but it is somewhat pointless not to have one.
- `PORT`: The port on which the server will run.
   Default is `8080`.
- `PERSIST_INTERVAL`: The interval at which the feed list is persisted to disk.
   Default is `1m`.
- `REFRESH_INTERVAL`: The interval at which the feeds are refreshed.
   Default is `5m`.
- `HEALTH_CHECK_INTERVAL`: The interval at which the server health is checked.
Default is `3m`.

## API

### `POST /login`
Authenticates the user with the provided token.

**Request body**:
```json
{
  "token": "your-access-token"
}
```

**Authenticated**: no

**Responses**:
- `200`: (empty body, sets a session cookie)
- `401`:
   ```json
   {
      "code": "401",
      "name": "Unauthorized",
      "message": "unauthorized"
   }
   ```

### `GET /api/feeds`
Returns a summary of all feeds.

**Request body**: none

**Authenticated**: yes

**Responses**:
- `200`:
   ```jsonc
   [
      {
         "uid": "feed1",
         "name": "Feed 1",
         "url": "http://example.com/feed1",
         "item_count": 1,
         "read_count": 0,
         "last_updated": 1633024800,
         "last_error": "",
         "items": [ /* item summaries without contents */ ]
      }
   ]
   ```
- `401`:
   ```json
   {
      "code": "401",
      "name": "Unauthorized",
      "message": "unauthorized"
   }
   ```

### `GET /api/feeds/{fuid}`
Returns a summary of the specified feed.

**Request body**: none

**Authenticated**: yes

**Responses**:
- `200`:
   ```json
   {
      "uid": "feed1",
      "name": "Feed 1",
      "url": "http://example.com/feed1",
      "item_count": 1,
      "read_count": 0,
      "last_updated": 1633024800,
      "last_error": "",
      "items": []
   }
   ```
- `404`:
   ```json
   {
      "code": "404",
      "name": "Not Found",
      "message": "feed not found"
   }
   ```
- `401`:
   ```json
   {
      "code": "401",
      "name": "Unauthorized",
      "message": "unauthorized"
   }
   ```

### `GET /api/feeds/{fuid}/items/{iuid}`
Returns a summary of the specified item.

**Request body**: none

**Authenticated**: yes

**Responses**:
- `200`:
   ```json
   {
      "uid": "item1",
      "feed_uid": "feed1",
      "feed_name": "Feed 1",
      "url": "http://example.com/item1",
      "title": "Item 1",
      "timestamp": 1633024800,
      "authors": "",
      "read": false,
      "content": "HTML content of item 1 (sanitized)"
   }
   ```
- `404`:
   ```json
   {
      "code": "404",
      "name": "Not Found",
      "message": "item not found"
   }
   ```
- `401`:
   ```json
   {
      "code": "401",
      "name": "Unauthorized",
      "message": "unauthorized"
   }
   ```

### `POST /api/feeds/{fuid}/read`
Marks all items in the specified feed as read up to the given timestamp.

**Request body**:
```json
{
  "before": 1633024800
}
```
**Authenticated**: yes
**Responses**:
- `200`: (empty body)
- `404`:
   ```json
   {
      "code": "404",
      "name": "Not Found",
      "message": "item or feed not found"
   }
   ```
- `401`:
   ```json
   {
      "code": "401",
      "name": "Unauthorized",
      "message": "unauthorized"
   }
   ```

### `POST /api/feeds/{fuid}/items/{iuid}/read`
Marks the specified item as read.

**Request body**: none

**Authenticated**: yes

**Responses**:
- `200`: (empty body)
- `404`:
   ```json
   {
      "code": "404",
      "name": "Not Found",
      "message": "item or feed not found"
   }
   ```
- `401`:
   ```json
   {
      "code": "401",
      "name": "Unauthorized",
      "message": "unauthorized"
   }
   ```

### `GET /status`
Returns the status and version of the application.

**Request body**: none

**Authenticated**: no

**Responses**:
- `200`: `{ "status": "ok", "version": "1.0.0" }`
- `500`:
   ```json
   {
      "code": "500",
      "name": "Internal Server Error",
      "message": "cannot read version file"
   }
   ```

### Error Format
All error responses follow this format:
```json
{
  "code": "HTTP status code",
  "name": "HTTP status text",
  "message": "Detailed error message"
}
```

## Deploying in Azure App Service

1. Deploy the app in Azure following the [quick start guide](https://learn.microsoft.com/en-us/azure/app-service/quickstart-custom-container?tabs=dotnet&pivots=container-linux-azure-portal).
   When selecting the container, input `ghcr.io` as the registry and `alnvdl/varys:main` as the image, leaving the startup command blank.

2. Make sure to set the following environment variables in the deployment:
   | Environment variable                  | Value
   | -                                     | -
   | `ACCESS_TOKEN`                        | A random secret value.
   | `SESSION_KEY`                         | Another random secret value
   | `DB_PATH`                             | `/home/db.json`
   | `FEEDS`                               | The JSON content of your feedlist.
   | `PORT`                                | `80`
   | `PERSIST_INTERVAL`                    | `15m`
   | `REFRESH_INTERVAL`                    | `20m`
   | `WEBSITES_ENABLE_APP_SERVICE_STORAGE` | `true`

   To generate secret random values, you can run `openssl rand 32 | base64`.

3. While not being required, you may want to enable log persistence as well by following this [guide](https://learn.microsoft.com/en-us/azure/app-service/troubleshoot-diagnostic-logs#enable-application-logging-linuxcontainer).

4. You may need to restart the application to make sure it works well.
