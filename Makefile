GO_MIN_VERSION := 11000 # go1.10
GO_VERSION_CHECK := \
  $(shell expr \
    $(shell go version | \
      awk '{print $$3}' | \
      cut -do -f2 | \
      sed -e 's/\.\([0-9][0-9]\)/\1/g' -e 's/\.\([0-9]\)/0\1/g' -e 's/^[0-9]\{3,4\}$$/&00/' \
    ) \>= $(GO_MIN_VERSION) \
  )
BIN := ./bin/ami-query

.PHONY: all
all: check-go $(BIN)

$(BIN):
	go build $(BUILDARGS) -o $@

.PHONY: race
race:
	@$(MAKE) all BUILDARGS=-race

.PHONY: dep
dep: check-go
	dep ensure

.PHONY: test
test: check-go
	go test $(TESTARGS) -timeout=30s ./...
	@$(MAKE) vet
	@$(MAKE) lint

.PHONY: vet
vet: check-go
	@echo "go tool vet $(VETARGS)"
	@go list ./... | xargs go list -f '{{.Dir}}' | xargs go tool vet $(VETARGS)

.PHONY: cover
cover: check-go
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out
	@rm -f coverage.out

.PHONY: lint
lint: check-go
	@echo "golint $(LINTARGS)"
	@for pkg in $(shell go list ./...) ; do \
		golint $(LINTARGS) $$pkg ; \
	done

.PHONY: rpm
rpm: clean all
	@$(MAKE) -C resources

.PHONY: clean
clean:
	@rm -rf ./bin *.rpm
	@$(MAKE) -C resources clean

.PHONY: check-go
check-go:
ifeq ($(GO_VERSION_CHECK),0)
	$(error go1.9 or higher is required)
endif
