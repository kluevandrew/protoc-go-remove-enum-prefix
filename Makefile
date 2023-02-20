BIN_DIR := $(CURDIR)/.bin

LINTER_VERSION = 1.50.1

.ONESHELL: install-linter
.PHONY: install-linter
install-linter:
	@type $(BIN_DIR)/golangci-lint < /dev/null 2>&1 && exit 0 || ( \
	(mkdir -p $(BIN_DIR) || true) && \
	export GOBIN=$(BIN_DIR) && \
		go install -ldflags '-w -s' github.com/golangci/golangci-lint/cmd/golangci-lint@v${LINTER_VERSION} \
	)

clean:
	rm -rf $(BIN_DIR) protoc-go-remove-enum-prefix

.PHONY: install-linter
lint: install-linter
	export GOBIN=$(BIN_DIR) && \
	export PATH=$$GOBIN:$$PATH && \
	golangci-lint run -v

.PHONY: fix
fix: install-linter
	export GOBIN=$(BIN_DIR) && \
	export PATH=$$GOBIN:$$PATH && \
	golangci-lint run -v --fix

.PHONY: test
test:
	export GOBIN=$(BIN_DIR) && \
	export PATH=$$GOBIN:$$PATH && \
	go test ./...

build:
	go build -o protoc-go-remove-enum-prefix .