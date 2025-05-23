# alnvdl/varys

Varys is a barebones feed reader written in Go. It has (practically) no
external dependencies and provides an equally barebones web experience. It is
meant to be self-hosted and used by a single user.

Varys is based on an in-memory feed list that can be configured to be persisted
to the disk and auto-refreshed. It also serves this feed list over an HTTP API,
and provides a static mobile-friendly SPA for reading feeds.

Varys is named after the GoT character who was always up-to-date with his
feeds, which he used to call "little birds".

## Running
To run Varys locally, just install Go 1.24+ and run `make dev_example`.

If you have a list of feeds in `feeds.json` (see the
[Feed list format](#feed-list-format) below), you can run `make dev` to use it.

To access the web interface, go to http://localhost:8080/#token:dev.

## Feed list format
The feed list is a JSON array where each feed is represented as an object:
```jsonc
[
   { /* xml feed */ },
   { /* html feed */ },
   { /* image feed */ }
]
```

These are the supported feed types and their accepted parameters.

### Atom and RSS feeds (type `xml`)
This type of feed can be used with traditional RSS or Atom XML feeds.
```jsonc
{
  "type": "xml",
  "name": "Example XML Feed",
  "url": "https://example.com/rss",
  "params": {
    // max_items is the optional maximum number of items to keep in the feed.
    // Defaults to a number between 100 and 200 based on the feed data.
    "max_items": 50
  }
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
    // encoding is only required if not UTF-8.
    "encoding": "ISO-8859-1",
    // container_tag (required) and container_attrs (optional) are used to
    // define the elements where anchors will be sourced from.
    "container_tag": "div",
    "container_attrs": {
      "class": "news-container"
    },
    // position identifies the position of the title in the extracted content.
    // Cannot be negative.
    "title_pos": 0,
    // base_url is used for resolving relative URLs found in HTML content.
    "base_url": "https://example.com/",
    // allowed_prefixes define the only acceptable prefixes for links identified
    // by the HTML feed parser after resolving them with base_url.
    "allowed_prefixes": [
      "https://example.com/news/"
    ],
    // max_items is the optional maximum number of items to keep in the feed.
    // Defaults to a number between 100 and 200 based on the feed data.
    "max_items": 50
  }
}
```

### Image feeds (type `img`)
This type of feed can be used for images that are updated frequently (e.g.,
hosted webcam images or weather report charts).
```jsonc
{
  "type": "img",
  "name": "Example Image Feed",
  "url": "https://example.com/image.png",
  "params": {
    // title is the titel of the resulting feed items, to which a timestamp
    // will be appended.
    "title": "Example Image",
    // url will be used for representing the URL of the resulting feed items.
    "url": "https://example.com/image.png",
    // mime_type defines the type of image returned by the feed URL.
    "mime_type": "image/png",
    // max_items is the optional maximum number of items to keep in the feed.
    // Defaults to a number between 100 and 200 based on the feed data.
    "max_items": 50
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
- `PERSIST_INTERVAL`: The interval for persisting the feed list to the disk.
   Default is `1m`.
- `REFRESH_INTERVAL`: The interval for refreshing the feeds.
   Default is `5m`.
- `HEALTH_CHECK_INTERVAL`: The interval for checking the health of the service.
   Default is `1m`.
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

### Authentication
The login process is triggered via JavaScript: the page will detect the
`#token:...` hash in the URL and call the `POST /login` endpoint with the
token. That leads to a session cookie being set if the token is correct.

Please note that this design means that the browser may keep the token in the
history. The typical pattern is to actually bookmark the reader with the token.

This is considered a reasonable security trade-off for this project for the
following reasons:
- Varys is design to be self-hosted and used by a single user;
- The only write operation is marking items as read.
- While the list of feeds might be sensitive information, all of the
  information contained in them is usually public on the internet.

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
It is quite easy to deploy and run this application on the Azure App Service
free tier, as long as you have a reasonable number of feeds.

1. Deploy the app in Azure following the
   [quick start guide](https://learn.microsoft.com/en-us/azure/app-service/quickstart-custom-container?tabs=dotnet&pivots=container-linux-azure-portal).
   When selecting the container image, input `ghcr.io` as the registry and
   `alnvdl/varys:main` as the image, leaving the startup command blank.

2. Make sure to set the following environment variables in the deployment:
   | Environment variable                  | Value
   | -                                     | -
   | `ACCESS_TOKEN`                        | A random secret value
   | `SESSION_KEY`                         | Another random secret value
   | `DB_PATH`                             | `/home/db.json`
   | `FEEDS`                               | The JSON feed list
   | `PORT`                                | `80`
   | `PERSIST_INTERVAL`                    | `15m`
   | `REFRESH_INTERVAL`                    | `20m`
   | `WEBSITES_ENABLE_APP_SERVICE_STORAGE` | `true`

   To generate secret random values, you can run `openssl rand 32 | base64`.

3. While not required, you may want to enable log persistence as well by
   following this
   [guide](https://learn.microsoft.com/en-us/azure/app-service/troubleshoot-diagnostic-logs#enable-application-logging-linuxcontainer).

4. You may need to restart the application after the initial setup to make
   sure all settings are picked up.

5. To deploy new versions of the image, just restart the application (assuming
   the deployment is using the `main` tag mentioned in step 1).

## Icons
All icons in this project come from the Bootstrap project:
https://icons.getbootstrap.com/.
