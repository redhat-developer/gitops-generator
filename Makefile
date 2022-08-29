PKGS := $(shell go list  ./... | grep -v test/e2e | grep -v vendor)
FMTPKGS := $(shell go list  ./... | grep -v vendor)

.PHONY: gofmt
gofmt:
	go fmt $(FMTPKGS)

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: test
test:
	go test ./...

.PHONY: gotest
gotest:
	 go test $(PKGS)

lint:
	golangci-lint --version
	GOMAXPROCS=2 golangci-lint run --fix --verbose --timeout 300s

