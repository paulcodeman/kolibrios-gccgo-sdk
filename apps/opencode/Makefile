PROGRAM = opencode
GO_PACKAGE = main
ROOT = ../..
ROOT_ABS := $(abspath $(ROOT))
PYTHON ?= python3
HOST_GO ?= go
PORT := $(PYTHON) $(ROOT_ABS)/tooling/opencode-port.py
OPENCODE_STAGE := $(ROOT_ABS)/.build-cache/opencode
APP_SOURCE_DIR = $(OPENCODE_STAGE)/app
PACKAGE_DIRS =
BUILD_CACHE_NAMESPACE = opencode
APP_MAIN_SYMBOL = main.kolibriMain
KPACK ?= 1
DEBUG_ELF = $(OPENCODE_STAGE)/opencode.kex.debug.elf
VENDOR_DIR = $(OPENCODE_STAGE)/vendor
KOLIBRI_COMPAT_RECORD = $(OPENCODE_STAGE)/dependency-compat.json
THIRD_PARTY_DIRS = $(OPENCODE_STAGE)/third_party $(ROOT_ABS)/third_party

# Stage before the common makefile resolves imports. Cleaning must also work
# before the upstream checkout has been prepared.
ifeq ($(strip $(filter-out clean clean-cache distclean prepare audit probe smoke print-output,$(MAKECMDGOALS))),)
ifneq ($(MAKECMDGOALS),)
AUTO_DEPS = 0
KOLIBRI_COMPAT = 0
VENDOR_DIR =
endif
endif
ifneq ($(AUTO_DEPS),0)
OPENCODE_STAGE_RESULT := $(shell $(PORT) stage --host-go $(HOST_GO) 2>&1)
ifneq ($(.SHELLSTATUS),0)
$(error OpenCode preparation failed: $(OPENCODE_STAGE_RESULT))
endif
endif

include $(ROOT)/tooling/kolibri-app.mk

.PHONY: prepare audit probe build smoke assets
all: assets
assets:
	+$(MAKE) -C $(ROOT_ABS)/apps/tools/gosh KPACK=$(KPACK)
	$(PYTHON) $(ROOT_ABS)/tooling/stage-opencode-assets.py --destination $(CURDIR)/assets
build: all
prepare:
	$(PORT) prepare --host-go $(HOST_GO)
audit:
	$(PORT) audit
probe:
	$(PORT) probe --compiler $(GO)
smoke:
	$(PORT) smoke --compiler $(GO)
