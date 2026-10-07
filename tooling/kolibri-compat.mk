# Shared dependency setup for applications and libraries, before import discovery.
PYTHON ?= python3
VENDOR_DIR ?= $(if $(wildcard $(CURDIR)/vendor/modules.txt),$(CURDIR)/vendor,)
KOLIBRI_COMPAT ?= 1
SDK_VENDOR_ROOT :=

ifneq ($(strip $(VENDOR_DIR)),)
SDK_VENDOR_ROOT := $(abspath $(VENDOR_DIR))
ifeq ($(wildcard $(SDK_VENDOR_ROOT)/.),)
$(error Vendor directory does not exist: $(SDK_VENDOR_ROOT))
endif
ifneq ($(KOLIBRI_COMPAT),0)
KOLIBRI_COMPAT_RECORD ?= $(SDK_VENDOR_ROOT)/kolibrios-compat-record.json
KOLIBRI_COMPAT_MODULES ?=
KOLIBRI_COMPAT_STAGE_RESULT := $(shell $(PYTHON) "$(ROOT_ABS)/tooling/stage-kolibrios-compat.py" --vendor "$(SDK_VENDOR_ROOT)" --record "$(KOLIBRI_COMPAT_RECORD)" $(foreach module,$(KOLIBRI_COMPAT_MODULES),--module "$(module)") 2>&1)
ifneq ($(.SHELLSTATUS),0)
$(error Shared KolibriOS dependency setup failed: $(KOLIBRI_COMPAT_STAGE_RESULT))
endif
endif
endif
