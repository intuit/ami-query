GO_MIN_VERSION := 10900 # go1.9
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

# dep will eventually provide a cleaner way to do this.
# https://github.com/golang/dep/issues/944
.PHONY: dep
dep: check-go check-dep
	@echo "dep ensure && dep prune"
	@dep ensure && dep prune && find ./vendor -name '*_test.go' -delete

.PHONY: test
test: check-go check-golint
	go test $(TESTARGS) -timeout=30s ./...
	@$(MAKE) vet
	@$(MAKE) lint

.PHONY: vet
vet: check-go
	@echo "go tool vet $(VETARGS)"
	@go list ./... | xargs go list -f '{{.Dir}}' | xargs go tool vet $(VETARGS)

.PHONY: cover
cover: check-go check-cover
	@go test -coverprofile=coverage.out "$(PKG)"
	@go tool cover -html=coverage.out
	@rm -f coverage.out

.PHONY: lint
lint: check-go check-golint
	@echo "golint $(LINTARGS)"
	@for pkg in $(shell go list ./...) ; do \
		golint $(LINTARGS) $$pkg ; \
	done

.PHONY: rpm
rpm: all
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

.PHONY: check-dep
check-dep:
ifeq (,$(shell which dep 2>/dev/null))
	$(error "dep not found in PATH")
endif

.PHONY: check-golint
check-golint:
ifeq (,$(shell which golint 2>/dev/null))
	$(error "golint not found in PATH")
endif

.PHONY: check-cover
check-cover:
ifndef PKG
	$(error PKG must be set to the package to cover, \
		e.g. `make cover PKG=./amicache`)
endif