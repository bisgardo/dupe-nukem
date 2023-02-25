# Make default target to nothing.
.PHONY: default
default:

.PHONY: fmt
fmt:
	goimports -local github.com/bisgardo/dupe-nukem -w .

.PHONY: build
build:
	go build ./cmd/dupe-nukem

.PHONY: test
test:
	go test ./...
