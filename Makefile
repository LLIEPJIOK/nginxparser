COVERAGE_FILE ?= coverage.out

TARGET ?= parser

.PHONY: build
build:
	@echo "Выполняется go build для таргета ${TARGET}"
	@mkdir -p .bin
	@go build -o ./bin/${TARGET} ./cmd/${TARGET}

## test: run all tests
.PHONY: test
test:
	@go test -coverpkg='github.com/LLIEPJIOK/nginxparser/...' --race -count=1 -coverprofile='$(COVERAGE_FILE)' ./...
	@go tool cover -func='$(COVERAGE_FILE)' | grep ^total | tr -s '\t'
