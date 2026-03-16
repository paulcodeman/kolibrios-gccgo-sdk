PROGRAM ?= app
PACKAGE_NAME ?= $(PROGRAM)
ROOT ?= ../..
ROOT_ABS := $(abspath $(ROOT))
STDLIB_DIR_ABS := $(ROOT_ABS)/stdlib
FIRST_PARTY_DIRS ?= platform
THIRD_PARTY_DIRS ?= third_party
FIRST_PARTY_DIRS_ABS := $(foreach dir,$(FIRST_PARTY_DIRS),$(if $(filter /%,$(dir)),$(dir),$(ROOT_ABS)/$(dir)))
THIRD_PARTY_DIRS_ABS := $(foreach dir,$(THIRD_PARTY_DIRS),$(if $(filter /%,$(dir)),$(dir),$(ROOT_ABS)/$(dir)))
PACKAGE_ARTIFACT_ROOT := $(ROOT)/.pkg
PACKAGE_ARTIFACT_ROOT_ABS := $(ROOT_ABS)/.pkg

ABI_DIR = $(ROOT)/platform/abi
MK_DIR = $(ROOT)/tooling
BUILD_DIR = .build

TOOLING_BIN ?= $(ROOT_ABS)/tooling/bin
GO ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/gccgo-15) \
  $(wildcard $(TOOLING_BIN)/gccgo) \
  $(shell command -v gccgo-15 2>/dev/null) \
  $(shell command -v gccgo 2>/dev/null) \
  gccgo-15)
GCC ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/gcc) \
  $(shell command -v gcc 2>/dev/null) \
  gcc)
LD ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/ld) \
  $(shell command -v ld 2>/dev/null) \
  ld)
STRIP ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/strip) \
  $(shell command -v strip 2>/dev/null) \
  strip)
ASM_COMPILER_FLAGS = -g -f elf32 -F dwarf
NASM_BIN ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/nasm) \
  $(shell command -v nasm 2>/dev/null) \
  nasm)
NASM = $(NASM_BIN) $(ASM_COMPILER_FLAGS)
OBJCOPY ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/objcopy) \
  $(wildcard $(TOOLING_BIN)/i386-elf-objcopy) \
  $(shell command -v objcopy 2>/dev/null) \
  objcopy)
SED = sed
OPT_LEVEL ?= -Os
GO_OPT_LEVEL ?= -Os
KEEP_PKG ?= 0
KEEP_ABI ?= 0
FAST_PKG ?= 0
KPACK ?= 0
KPACK_BIN ?= $(ROOT)/tooling/bin/kpack
KPACK_FLAGS ?= --nologo

ENTRYPOINT = go_0$(PACKAGE_NAME).Main
LDSCRIPT_TEMPLATE = $(MK_DIR)/static.lds.in
LDSCRIPT = $(BUILD_DIR)/$(PROGRAM).lds
APP_STACK_RESERVE ?= 0x10000
STARTUP_TEMPLATE = $(MK_DIR)/app-startup.c.in
STARTUP_SOURCE = $(BUILD_DIR)/$(PROGRAM).startup.c
STARTUP_OBJ = $(BUILD_DIR)/$(PROGRAM).startup.o

GO_COMPILER_FLAGS = -m32 -c $(GO_OPT_LEVEL) -nostdlib -nostdinc -fexceptions -fno-stack-protector -fno-split-stack -static -fno-leading-underscore -fno-common -fno-pie -g -ffunction-sections -fdata-sections -I. -I$(ROOT_ABS) -I$(PACKAGE_ARTIFACT_ROOT_ABS)
GCC_COMPILER_FLAGS = -m32 -c $(OPT_LEVEL) -ffunction-sections -fdata-sections -fno-pic -fno-pie -fno-stack-protector -fno-builtin-calloc
LDFLAGS = -n -T $(LDSCRIPT) -m elf_i386 -z noexecstack -z relro -z now --gc-sections --eh-frame-hdr --entry=$(ENTRYPOINT)

APP_SOURCES = $(wildcard *.go)
GO_PACKAGE ?= $(strip $(shell $(SED) -n 's/^package[[:space:]]\+\([A-Za-z_][A-Za-z0-9_]*\).*$$/\1/p;q' $(firstword $(APP_SOURCES))))

ifeq ($(GO_PACKAGE),main)
APP_INIT_SYMBOL ?= __go_init_main
APP_MAIN_SYMBOL ?= main.main
else
APP_INIT_SYMBOL ?= go.$(GO_PACKAGE)..import
APP_MAIN_SYMBOL ?= go_0$(GO_PACKAGE).Main
endif

ENTRYPOINT = runtime_kolibri_app_entry

PACKAGE_DIRS ?= kos ui
DEBUG_PKG ?= 0
AUTO_DEPS ?= 1
ifneq ($(AUTO_DEPS),0)
RESOLVED_PACKAGE_DIRS := $(shell $(MK_DIR)/resolve-packages.py --root $(ROOT_ABS) --stdlib $(STDLIB_DIR_ABS) --first-party "$(FIRST_PARTY_DIRS)" --third-party "$(THIRD_PARTY_DIRS)" --app-dir $(CURDIR) --packages "$(PACKAGE_DIRS)")
PACKAGE_DIRS := $(RESOLVED_PACKAGE_DIRS)
endif
PACKAGE_OBJS =
PACKAGE_GOXS =
PREVIOUS_PACKAGE_GOXS =

define FIND_PACKAGE_DIR
$(strip $(firstword \
  $(wildcard $(ROOT_ABS)/$(1)) \
  $(foreach dir,$(FIRST_PARTY_DIRS_ABS),$(wildcard $(dir)/$(1))) \
  $(foreach dir,$(THIRD_PARTY_DIRS_ABS),$(wildcard $(dir)/$(1))) \
  $(wildcard $(STDLIB_DIR_ABS)/$(1)) \
))
endef

define REGISTER_PACKAGE
$(if $(filter 1,$(DEBUG_PKG)),$(info PKG=$(1) DIR=$(call FIND_PACKAGE_DIR,$(1))))
PACKAGE_SOURCE_DIR_$(1) := $(call FIND_PACKAGE_DIR,$(1))
$$(if $$(PACKAGE_SOURCE_DIR_$(1)),,$$(error package source dir not found for $(1)))
PACKAGE_SOURCES_$(1) := $$(filter-out $$(PACKAGE_SOURCE_DIR_$(1))/%_gc_kolibrios.go $$(PACKAGE_SOURCE_DIR_$(1))/%_test.go,$$(wildcard $$(PACKAGE_SOURCE_DIR_$(1))/*.go))
PACKAGE_SOURCE_FILES_$(1) := $$(notdir $$(PACKAGE_SOURCES_$(1)))
PACKAGE_ARTIFACT_PREFIX_$(1) := $(if $(findstring /,$(1)),$(PACKAGE_ARTIFACT_ROOT)/$(1),$(ROOT)/$(1))
PACKAGE_ARTIFACT_PREFIX_ABS_$(1) := $(if $(findstring /,$(1)),$(PACKAGE_ARTIFACT_ROOT_ABS)/$(1),$(ROOT_ABS)/$(1))
PACKAGE_OBJ_$(1) := $$(PACKAGE_ARTIFACT_PREFIX_$(1)).gccgo.o
PACKAGE_GOX_$(1) := $$(PACKAGE_ARTIFACT_PREFIX_$(1)).gox
ifneq ($(FAST_PKG),0)
PACKAGE_ORDER_DEPS_$(1) := $$(if $$(PREVIOUS_PACKAGE_GOXS),| $$(PREVIOUS_PACKAGE_GOXS),)
else
PACKAGE_ORDER_DEPS_$(1) := $$(PREVIOUS_PACKAGE_GOXS)
endif

PACKAGE_OBJS += $$(PACKAGE_OBJ_$(1))
PACKAGE_GOXS += $$(PACKAGE_GOX_$(1))

$$(PACKAGE_OBJ_$(1)): $$(PACKAGE_SOURCES_$(1)) $$(PACKAGE_ORDER_DEPS_$(1))
	mkdir -p $$(dir $$@)
	cd $$(PACKAGE_SOURCE_DIR_$(1)) && $(GO) $(GO_COMPILER_FLAGS) -o $$(PACKAGE_ARTIFACT_PREFIX_ABS_$(1)).gccgo.o $$(PACKAGE_SOURCE_FILES_$(1))

$$(PACKAGE_GOX_$(1)): $$(PACKAGE_OBJ_$(1))
	mkdir -p $$(dir $$@)
	$(OBJCOPY) -j .go_export $$< $$@

PREVIOUS_PACKAGE_GOXS += $$(PACKAGE_GOX_$(1))
endef

$(foreach pkg,$(PACKAGE_DIRS),$(eval $(call REGISTER_PACKAGE,$(pkg))))

APP_OBJ = $(PROGRAM).gccgo.o

LIBGCC = $(shell $(GCC) -m32 -print-libgcc-file-name)
LIBGCC_EH = $(shell $(GCC) -m32 -print-file-name=libgcc_eh.a)
RUNTIME_LIBS = $(LIBGCC_EH) $(LIBGCC)

ABI_OBJS = $(ABI_DIR)/syscalls_i386.o $(ABI_DIR)/runtime_gccgo.o $(ABI_DIR)/go-unwind.o $(ABI_DIR)/runtime_context_386.o
STARTUP_ARTIFACTS = $(STARTUP_SOURCE) $(STARTUP_OBJ)
OBJS = $(ABI_OBJS) $(PACKAGE_OBJS) $(APP_OBJ)
OBJS += $(STARTUP_OBJ)
PACKAGE_ARTIFACTS = $(PACKAGE_OBJS) $(PACKAGE_GOXS)
INTERMEDIATE_ARTIFACTS = $(ABI_OBJS) $(APP_OBJ) $(LDSCRIPT) $(STARTUP_ARTIFACTS)

.PHONY: all clean link

all: $(PROGRAM).kex

clean:
	rm -rf $(BUILD_DIR) $(PACKAGE_ARTIFACT_ROOT)
	rm -f $(INTERMEDIATE_ARTIFACTS) $(PACKAGE_ARTIFACTS) $(PROGRAM).kex

link: $(PROGRAM).kex

$(BUILD_DIR):
	mkdir -p $@

$(LDSCRIPT): $(LDSCRIPT_TEMPLATE) | $(BUILD_DIR)
	$(SED) -e 's/@ENTRYPOINT@/$(ENTRYPOINT)/g' -e 's/@STACK_RESERVE@/$(APP_STACK_RESERVE)/g' $< > $@

$(STARTUP_SOURCE): $(STARTUP_TEMPLATE) | $(BUILD_DIR)
	$(SED) -e 's/@APP_INIT_SYMBOL@/$(APP_INIT_SYMBOL)/g' -e 's/@APP_MAIN_SYMBOL@/$(APP_MAIN_SYMBOL)/g' $< > $@

$(STARTUP_OBJ): $(STARTUP_SOURCE)
	$(GCC) $(GCC_COMPILER_FLAGS) $< -o $@

$(PROGRAM).kex: $(OBJS) $(PACKAGE_GOXS) $(LDSCRIPT)
	$(LD) $(LDFLAGS) -o $(PROGRAM).kex $(OBJS) $(RUNTIME_LIBS)
	$(STRIP) $(PROGRAM).kex
	$(OBJCOPY) $(PROGRAM).kex -O binary
ifneq ($(KPACK),0)
	$(KPACK_BIN) $(KPACK_FLAGS) $(PROGRAM).kex
endif
ifeq ($(KEEP_ABI),0)
	rm -f $(ABI_OBJS)
endif
	rm -f $(APP_OBJ) $(LDSCRIPT) $(STARTUP_ARTIFACTS)
ifeq ($(KEEP_PKG),0)
	rm -f $(PACKAGE_ARTIFACTS)
	rm -rf $(PACKAGE_ARTIFACT_ROOT)
endif
	rmdir $(BUILD_DIR) 2>/dev/null || true

$(APP_OBJ): $(APP_SOURCES) $(PACKAGE_GOXS)
	$(GO) $(GO_COMPILER_FLAGS) -o $@ $(APP_SOURCES)

$(ABI_DIR)/runtime_gccgo.o: $(ABI_DIR)/runtime_gccgo.c
	$(GCC) $(GCC_COMPILER_FLAGS) -fexceptions $< -o $@

$(ABI_DIR)/syscalls_i386.o: $(ABI_DIR)/syscalls_i386.asm
	$(NASM) $< -o $@

$(ABI_DIR)/go-unwind.o: $(ABI_DIR)/go-unwind.c
	$(GCC) $(GCC_COMPILER_FLAGS) -fexceptions $< -o $@

$(ABI_DIR)/runtime_context_386.o: $(ABI_DIR)/runtime_context_386.S
	$(GCC) $(GCC_COMPILER_FLAGS) $< -o $@
