.PHONY: build test vet fmt run-tui run-mobile clean

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

run-mobile:
	go run ./mobile

clean:
	rm -f tasktracker
