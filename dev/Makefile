# Default cli named parameter values
ARCH ?= amd64
APP  ?= prayertexter

# Allowed cli named parameter values
VALID_ARCH := amd64 arm64
VALID_APP  := prayertexter statecontroller announcer

# Validate that parameter values are acceptable
ifeq ($(filter $(ARCH),$(VALID_ARCH)),)
  $(error Unknown ARCH '$(ARCH)'; valid values are $(VALID_ARCH))
endif

ifeq ($(filter $(APP),$(VALID_APP)),)
  $(error Unknown APP '$(APP)'; valid values are $(VALID_APP))
endif

APPDIR  := ../cmd/$(APP)
BINARY  := bootstrap
VERSION := $(shell git describe --tags --long --dirty 2>/dev/null)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
OS      := linux
OUTDIR  := bin/$(OS)_$(ARCH)/$(APP)

ifeq ($(VERSION),)
	VERSION := UNKNOWN
endif

ifeq ($(APP),prayertexter)
	APPDIR := ../dev/$(APP)
endif

define BUILD_CMD
	@mkdir -p $(OUTDIR)
	@echo "→ Building $(APP) for $(OS)/$(ARCH) (version=$(VERSION))"
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -tags lambda.norpc $(LDFLAGS) -o $(OUTDIR)/$(BINARY) $(APPDIR)
endef

.PHONY: build clean
build:
	$(call BUILD_CMD,$(BUILD))
	@cp $(OUTDIR)/$(BINARY) $(APPDIR)

clean:
	@rm -rf bin
	@rm -f prayertexter/bootstrap
	@rm -f ../cmd/statecontroller/bootstrap