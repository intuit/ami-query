.PHONY: all race test fmt rpms clean clean-rpms clean-all check-gopath

all: check-gopath clean fmt test
	@echo "==> Compiling source code"
	@GO15VENDOREXPERIMENT=1 go build -v -o ./bin/ami-query

race: check-gopath clean fmt test
	@echo "==> Compiling source code with race detection enabled"
	@GO15VENDOREXPERIMENT=1 go build -race -o ./bin/ami-query

test: check-gopath
	@echo "==> Running tests"
	@GO15VENDOREXPERIMENT=1 go test $(TESTARGS) -cover `go list ./... | grep -v vendor`

fmt:
	@echo "==> Formatting source code"
	@GO15VENDOREXPERIMENT=1 go fmt `go list ./... | grep -v vendor`

rpm: all
	@$(MAKE) -C resources

clean:
	@echo "==> Removing previous builds"
	@rm -rf ./bin *.rpm

clean-rpm:
	@$(MAKE) -C resources clean

clean-all: clean clean-rpm

check-gopath:
ifndef GOPATH
	$(error GOPATH is undefined)
endif
