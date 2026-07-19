.PHONY: build test vet fmt run-tui clean

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

run-tui:
	go run ./tui

clean:
	rm -f tasktracker
