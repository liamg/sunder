
test:
	go test -v -race ./...

run:
	go build ./cmd/sunder && ./sunder
