export GOFLAGS=-mod=vendor

BIN := ./bin/ami-query

# Default Go linker flags.
GO_LDFLAGS ?= -ldflags="-s -w"

.PHONY: all
all: test clean
	GOOS=linux GOARCH=amd64 go build $(GO_LDFLAGS) $(BUILDARGS) -o $(BIN)

.PHONY: race
race:
	@$(MAKE) all BUILDARGS=-race

.PHONY: rpm
rpm: clean all
	@$(MAKE) -C resources

.PHONY: clean
clean:
	@rm -rf ./bin *.rpm
	@$(MAKE) -C resources clean

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: test
test:
	go test -race -timeout=30s $(TESTARGS) ./...
	@$(MAKE) vet

.PHONY: vet
vet:
	go vet $(VETARGS) ./...

.PHONY: lint
lint:
	@echo "golint $(LINTARGS)"
	@for pkg in $(shell go list ./...) ; do \
		golint $(LINTARGS) $$pkg ; \
	done

.PHONY: cover
cover:
	@$(MAKE) test TESTARGS="-tags test -coverprofile=coverage.out"
	@go tool cover -html=coverage.out
	@rm -f coverage.out
