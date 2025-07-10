BIN := blocc
BINPATH := $(shell go env GOPATH)/bin
DISTPATH := ./dist/blocc_darwin_amd64_v1/$(BIN)

clean:
	-rm $(BINPATH)/$(BIN)

build: clean
	goreleaser release --snapshot --clean
	cp $(DISTPATH) $(BINPATH)

test:
	go test -v $(shell go list ./... | grep -v /integration)

test-integration:
	go test -v ./integration/...

test-all:
	go test -v ./...

install:
	go install ./cmd/blocc

.PHONY: build clean test test-integration test-all install