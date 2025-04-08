VERSION := $(shell git describe --tags --long --dirty 2>/dev/null)

ifeq ($(VERSION),)
	VERSION := UNKNOWN
endif

# DEFAULTS
ARCH  ?= amd64
BUILD ?= prayertexter

ifeq ($(ARCH),arm64)
	GOARCH := arm64
else ifeq ($(ARCH),amd64)
	GOARCH := amd64
else
$(error Unknown ARCH '$(ARCH)'; valid values are amd64, arm64)
endif

GOOS    := linux
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := bootstrap
OUTDIR  := bin/$(GOOS)_$(GOARCH)

define BUILD_CMD
	@mkdir -p $(OUTDIR)/$(1)
	@echo "→ Building $(1) for $(GOOS)/$(GOARCH) (version=$(VERSION))"
	GOOS=linux GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(OUTDIR)/$(1)/$(BINARY) ./cmd/$(1)
endef

.PHONY: build clean
build:
	$(call BUILD_CMD,$(BUILD))
	@cp $(OUTDIR)/$(BUILD)/$(BINARY) .

clean:
	@rm -rf bin
	@rm -f bootstrap