dev: version
	FEEDS=$$(cat feeds.json) DEBUG=1 SESSION_KEY=abc ACCESS_TOKEN=dev DB_PATH=db.json PERSIST_INTERVAL=1m go run main.go

dev_example: version
	FEEDS=$$(echo '[\
		{"name":"Ars Technica", "url":"https://feeds.arstechnica.com/arstechnica/index", "type": "xml"}, \
		{"name":"The Verge", "url":"https://www.theverge.com/rss/index.xml", "type": "xml"} \
	]') DEBUG=1 SESSION_KEY=abc ACCESS_TOKEN=dev go run main.go

test:
	go test ./... -cover -coverprofile=/tmp/cover.out -timeout=10s

version:
	git update-index --assume-unchanged ./internal/web/static/version
	git describe --always --dirty > ./internal/web/static/version
	echo "Version set to: $$(cat ./internal/web/static/version)"
