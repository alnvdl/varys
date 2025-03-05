dev:
	DEBUG=1 SESSION_KEY=abc ACCESS_TOKEN=def go run main.go

test:
	go test ./... -cover -coverprofile=/tmp/cover.out
