MOCKERY_VERSION := v3.7.0
GOLANGCI_LINT_VERSION := v2.11.4

.PHONY: test lint generate check-mockery build clean

build:
	go build ./...

test:
	go test ./... -v -count=1

lint:
	docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run ./...

generate: check-mockery
	mockery

check-mockery:
	@command -v mockery >/dev/null 2>&1 || { echo "mockery not found. Install: go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)"; exit 1; }
	@INSTALLED=$$(mockery version 2>&1); \
	if [ "$$INSTALLED" != "$(MOCKERY_VERSION)" ]; then \
		echo "mockery version mismatch: expected $(MOCKERY_VERSION), got $$INSTALLED"; \
		echo "Install: go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)"; \
		exit 1; \
	fi

clean:
	rm -rf internal/mocks/
