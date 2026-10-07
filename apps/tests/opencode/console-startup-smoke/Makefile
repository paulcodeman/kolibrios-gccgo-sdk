PROGRAM = opencode-console-startup-smoke
GO_PACKAGE = main
ROOT = ../../../..
PACKAGE_DIRS = kos os embed unsafe
BUILD_CACHE_NAMESPACE = opencode-runtime-check
APP_MAIN_SYMBOL = main.kolibriMain
STARTUP_FIXTURE_DIR = .build/startup-fixture
override APP_SOURCES = $(STARTUP_FIXTURE_DIR)/main.go $(STARTUP_FIXTURE_DIR)/console_kolibrios.go
include $(ROOT)/tooling/kolibri-app.mk

# Keep the unchanged launch adapter and its embedded font in one Go package.
$(STARTUP_FIXTURE_DIR)/main.go: main.go
	mkdir -p $(STARTUP_FIXTURE_DIR)
	cp $< $@

$(STARTUP_FIXTURE_DIR)/console_kolibrios.go: $(ROOT)/apps/opencode/bootstrap/console_kolibrios.go $(STARTUP_FIXTURE_DIR)/console-unifont.kbf
	mkdir -p $(STARTUP_FIXTURE_DIR)
	cp $< $@

$(STARTUP_FIXTURE_DIR)/console-unifont.kbf: $(ROOT)/tooling/build-unifont.py $(ROOT)/third_party/unifont/UPSTREAM.json $(ROOT)/third_party/unifont/unifont_all-16.0.04.hex.gz
	python3 $(ROOT)/tooling/build-unifont.py $@
