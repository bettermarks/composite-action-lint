

all: composite-action-lint


composite-action-lint: *.go ./cmd/composite-action-lint/*.go
	go build ./cmd/composite-action-lint/.


.PHONY: clean
clean:
	rm -f composite-action-lint
