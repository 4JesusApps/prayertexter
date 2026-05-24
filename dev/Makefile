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
OS      := linux
OUTDIR  := bin/$(OS)_$(ARCH)/$(APP)

ifeq ($(APP),prayertexter)
	APPDIR := ../dev/$(APP)
endif

# Version is embedded automatically by `go build` via debug.ReadBuildInfo
# (see internal/buildinfo). No ldflags wiring required.
define BUILD_CMD
	@mkdir -p $(OUTDIR)
	@echo "→ Building $(APP) for $(OS)/$(ARCH)"
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(OUTDIR)/$(BINARY) $(APPDIR)
endef

.PHONY: build clean
build:
	$(call BUILD_CMD,$(BUILD))
	@cp $(OUTDIR)/$(BINARY) $(APPDIR)

clean:
	@rm -rf bin
	@rm -f prayertexter/bootstrap
	@rm -f ../cmd/statecontroller/bootstrap