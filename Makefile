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
	go test ./... -coverprofile cover.out

.PHONY: gotest
gotest:
	 go test $(PKGS)

lint:
	golangci-lint --version
	GOMAXPROCS=2 golangci-lint run --fix --verbose --timeout 300s

### fmt_license: ensure license header is set on all files
fmt_license:
ifneq ($(shell command -v addlicense 2> /dev/null),)
	@echo 'addlicense -v -f license_header.txt **/*.go'
	@addlicense -v -f license_header.txt $$(find . -name '*.go')
else
	$(error addlicense must be installed for this rule: go get -u github.com/google/addlicense)
endif

### check_fmt: Checks the formatting on files in repo
check_fmt:
  ifeq ($(shell command -v goimports 2> /dev/null),)
	  $(error "goimports must be installed for this rule" && exit 1)
  endif
  ifeq ($(shell command -v addlicense 2> /dev/null),)
	  $(error "error addlicense must be installed for this rule: go get -u github.com/google/addlicense")
  endif

	  if [[ $$(find . -not -path '*/\.*' -not -name '*zz_generated*.go' -name '*.go' -exec goimports -l {} \;) != "" ]]; then \
	    echo "Files not formatted; run 'make fmt'"; exit 1 ;\
	  fi ;\
	  if ! addlicense -check -f license_header.txt $$(find . -not -path '*/\.*' -name '*.go'); then \
	    echo "Licenses are not formatted; run 'make fmt_license'"; exit 1 ;\
	  fi \

### gosec - runs the gosec scanner for non-test files in this repo
.PHONY: gosec
gosec:
	# Run this command to install gosec, if not installed:
	# go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -no-fail -fmt=sarif -out=gosec.sarif  ./...