dev:
	FEEDS=$$(echo '[\
		{"name":"Ars Technica", "url":"https://feeds.arstechnica.com/arstechnica/index", "type": "xml"}, \
		{"name":"The Verge", "url":"https://www.theverge.com/rss/index.xml", "type": "xml"} \
	]') DEBUG=1 SESSION_KEY=abc ACCESS_TOKEN=def go run main.go

test:
	go test ./... -cover -coverprofile=/tmp/cover.out
