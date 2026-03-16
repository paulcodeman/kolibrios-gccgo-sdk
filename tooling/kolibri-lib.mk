PROGRAM ?= lib
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
ASM_COMPILER_FLAGS = -f elf32 $(if $(filter 1,$(DEBUG)),-g -F dwarf,)
NASM_BIN ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/nasm) \
  $(shell command -v nasm 2>/dev/null) \
  nasm)
NASM = $(NASM_BIN) $(ASM_COMPILER_FLAGS)
OBJCOPY ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/objcopy) \
  $(shell command -v objcopy 2>/dev/null) \
  objcopy)
OBJCOPY_COFF := $(firstword \
  $(shell test -x $(ROOT_ABS)/tooling/bin/i386-elf-objcopy && echo $(ROOT_ABS)/tooling/bin/i386-elf-objcopy) \
  $(shell command -v i386-elf-objcopy 2>/dev/null) \
  $(shell test -x /usr/local/binutils-coff/bin/i386-elf-objcopy && echo /usr/local/binutils-coff/bin/i386-elf-objcopy) \
  $(shell command -v i686-w64-mingw32-objcopy 2>/dev/null) \
  $(shell command -v x86_64-w64-mingw32-objcopy 2>/dev/null))
OBJCOPY_TARGETS := $(shell $(OBJCOPY) --help 2>/dev/null | sed -n 's/^.*objcopy: supported targets: //p')
ifeq ($(strip $(filter coff-i386,$(OBJCOPY_TARGETS))),)
ifneq ($(strip $(OBJCOPY_COFF)),)
OBJCOPY := $(OBJCOPY_COFF)
OBJCOPY_TARGETS := $(shell $(OBJCOPY) --help 2>/dev/null | sed -n 's/^.*objcopy: supported targets: //p')
endif
endif
LD ?= $(firstword \
  $(wildcard $(TOOLING_BIN)/ld) \
  $(shell command -v ld 2>/dev/null) \
  ld)
SED = sed
PYTHON ?= python3
OPT_LEVEL ?= -Os
GO_OPT_LEVEL ?= -Os
DEBUG ?= 0
KEEP_PKG ?= 0
KEEP_ABI ?= 0
FAST_PKG ?= 0
OBJ_GC_SECTIONS ?= 1
OBJ_REQUIRE_COFF ?= 1
OBJ_FORMAT ?= coff-i386
OBJ_FORMAT_SUPPORTED := $(if $(filter $(OBJ_FORMAT),$(OBJCOPY_TARGETS)),1,0)
ifeq ($(strip $(OBJCOPY_TARGETS)),)
OBJ_FORMAT_SUPPORTED := 0
endif
ifeq ($(OBJ_FORMAT_SUPPORTED),0)
ifeq ($(OBJ_FORMAT),coff-i386)
ifeq ($(OBJ_REQUIRE_COFF),1)
$(error objcopy does not support coff-i386; install a toolchain with coff-i386 support or set OBJ_REQUIRE_COFF=0 OBJ_FORMAT=pei-i386)
else
$(warning objcopy does not support coff-i386; falling back to pei-i386)
OBJ_FORMAT := pei-i386
endif
else
$(error objcopy does not support $(OBJ_FORMAT))
endif
endif
OBJ_WITH_LIBGCC ?= 0
OBJ_EXTRA_OBJS ?=
OBJ_EXTRA_CLEAN ?=
OBJ_REQUIRE_EXPORTS ?= 1
OBJ_STRIP ?= 1
OBJ_STRIP_SECTIONS ?= .comment .note.GNU-stack .go_export
OBJ_STRIP_FLAGS ?= --strip-debug $(foreach sec,$(OBJ_STRIP_SECTIONS),--remove-section $(sec))
EXPORTS_STUBS ?= 1
EXPORTS_STUBS_MODE ?= direct
EXPORTS_STUBS_STRICT ?= 1
EXPORTS_FILE ?= exports.txt
EXPORTS_GEN = $(MK_DIR)/gen-kolibri-exports.py
OBJ_SHORTEN_SYMBOLS ?= 1
OBJ_SHORTEN_TOOL ?= $(MK_DIR)/shorten-elf-symbols.py
OBJ_SHORTEN_KEEP ?= EXPORTS
OBJ_SHORTEN_OUTPUT = $(BUILD_DIR)/$(PROGRAM).obj.short.elf
COFF_FIX_EXPORTS ?= 1
COFF_FIX_TOOL ?= $(MK_DIR)/fix-coff-exports.py

GO_COMPILER_FLAGS = -m32 -c $(GO_OPT_LEVEL) -nostdlib -nostdinc -fexceptions -fno-stack-protector -fno-split-stack -static -fno-leading-underscore -fno-common -fno-pie $(if $(filter 1,$(DEBUG)),-g,) -ffunction-sections -fdata-sections -I. -I$(ROOT_ABS) -I$(PACKAGE_ARTIFACT_ROOT_ABS)
GCC_COMPILER_FLAGS = -m32 -c $(OPT_LEVEL) -ffunction-sections -fdata-sections -fno-pic -fno-pie -fno-stack-protector -fno-builtin-calloc $(if $(filter 1,$(DEBUG)),-g,)

APP_SOURCES = $(wildcard *.go)
GO_PACKAGE ?= $(strip $(shell $(SED) -n 's/^package[[:space:]]\+\([A-Za-z_][A-Za-z0-9_]*\).*$$/\1/p;q' $(firstword $(APP_SOURCES))))

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
PACKAGE_ARTIFACTS = $(PACKAGE_OBJS) $(PACKAGE_GOXS)
INTERMEDIATE_ARTIFACTS = $(ABI_OBJS) $(APP_OBJ)
OBJ_INTERMEDIATE = $(BUILD_DIR)/$(PROGRAM).obj.elf
OBJ_EXPORTS_PRESENT = 0
OBJ_EXPORTS_SOURCE =
OBJ_EXPORTS_OBJ =
OBJ_EXPORTS_STUBS_SOURCE =
OBJ_EXPORTS_STUBS_OBJ =
ifneq ($(wildcard $(EXPORTS_FILE)),)
OBJ_EXPORTS_PRESENT = 1
OBJ_EXPORTS_SOURCE = $(BUILD_DIR)/$(PROGRAM).exports.S
OBJ_EXPORTS_OBJ = $(BUILD_DIR)/$(PROGRAM).exports.o
ifneq ($(EXPORTS_STUBS),0)
OBJ_EXPORTS_STUBS_SOURCE = $(BUILD_DIR)/$(PROGRAM).exports.c
OBJ_EXPORTS_STUBS_OBJ = $(BUILD_DIR)/$(PROGRAM).exports.stub.o
endif
endif
ifneq ($(OBJ_EXPORTS_PRESENT),0)
ifeq ($(origin OBJ_GC_ROOT), undefined)
OBJ_GC_ROOT := EXPORTS
endif
endif
OBJ_INPUTS = $(ABI_OBJS) $(PACKAGE_OBJS) $(APP_OBJ) $(OBJ_EXPORTS_OBJ) $(OBJ_EXPORTS_STUBS_OBJ) $(OBJ_EXTRA_OBJS)
ifneq ($(OBJ_WITH_LIBGCC),0)
OBJ_INPUTS += $(RUNTIME_LIBS)
endif
INTERMEDIATE_ARTIFACTS += $(OBJ_INTERMEDIATE) $(OBJ_SHORTEN_OUTPUT) $(OBJ_EXPORTS_SOURCE) $(OBJ_EXPORTS_OBJ) $(OBJ_EXPORTS_STUBS_SOURCE) $(OBJ_EXPORTS_STUBS_OBJ)

LD_RELOC_FLAGS = -r -m elf_i386
ifneq ($(OBJ_GC_SECTIONS),0)
ifneq ($(strip $(OBJ_GC_ROOT)),)
LD_RELOC_FLAGS += --gc-sections -u $(OBJ_GC_ROOT)
endif
endif

OBJ_BUILD_GOALS := $(filter obj all,$(MAKECMDGOALS))
ifeq ($(strip $(MAKECMDGOALS)),)
OBJ_BUILD_GOALS := all
endif
ifneq ($(OBJ_BUILD_GOALS),)
ifeq ($(OBJ_REQUIRE_EXPORTS),1)
ifeq ($(OBJ_EXPORTS_PRESENT),0)
$(error obj build requires $(EXPORTS_FILE) or set OBJ_REQUIRE_EXPORTS=0)
endif
endif
endif

.PHONY: all clean obj

all: $(PROGRAM).obj

clean:
	rm -rf $(BUILD_DIR) $(PACKAGE_ARTIFACT_ROOT)
	rm -f $(INTERMEDIATE_ARTIFACTS) $(PACKAGE_ARTIFACTS) $(PROGRAM).obj $(OBJ_EXTRA_CLEAN)

obj: $(PROGRAM).obj

$(BUILD_DIR):
	mkdir -p $@

ifneq ($(OBJ_EXPORTS_PRESENT),0)
$(OBJ_EXPORTS_SOURCE): $(EXPORTS_FILE) $(EXPORTS_GEN) | $(BUILD_DIR)
	$(PYTHON) $(EXPORTS_GEN) --package $(GO_PACKAGE) --input $(EXPORTS_FILE) --output $@ $(if $(filter 0,$(EXPORTS_STUBS)),, --c-output $(OBJ_EXPORTS_STUBS_SOURCE) --stub-mode $(EXPORTS_STUBS_MODE) --stub-strict $(EXPORTS_STUBS_STRICT))

$(OBJ_EXPORTS_OBJ): $(OBJ_EXPORTS_SOURCE)
	$(GCC) $(GCC_COMPILER_FLAGS) $< -o $@

ifneq ($(OBJ_EXPORTS_STUBS_SOURCE),)
$(OBJ_EXPORTS_STUBS_OBJ): $(OBJ_EXPORTS_STUBS_SOURCE)
	$(GCC) $(GCC_COMPILER_FLAGS) $< -o $@
endif
endif

$(PROGRAM).obj: $(OBJ_INPUTS) | $(BUILD_DIR)
	$(LD) $(LD_RELOC_FLAGS) -o $(OBJ_INTERMEDIATE) $(OBJ_INPUTS)
ifneq ($(OBJ_SHORTEN_SYMBOLS),0)
	$(PYTHON) $(OBJ_SHORTEN_TOOL) --input $(OBJ_INTERMEDIATE) --output $(OBJ_SHORTEN_OUTPUT) --objcopy $(OBJCOPY) $(foreach sym,$(OBJ_SHORTEN_KEEP),--keep $(sym))
	mv $(OBJ_SHORTEN_OUTPUT) $(OBJ_INTERMEDIATE)
endif
ifneq ($(OBJ_STRIP),0)
	$(OBJCOPY) $(OBJ_STRIP_FLAGS) $(OBJ_INTERMEDIATE)
endif
	$(OBJCOPY) -O $(OBJ_FORMAT) $(OBJ_INTERMEDIATE) $@
ifneq ($(filter coff-i386,$(OBJ_FORMAT)),)
ifneq ($(COFF_FIX_EXPORTS),0)
	$(PYTHON) $(COFF_FIX_TOOL) --file $@ --symbol EXPORTS --section .data
endif
endif
ifeq ($(KEEP_ABI),0)
	rm -f $(ABI_OBJS)
endif
	rm -f $(APP_OBJ) $(OBJ_INTERMEDIATE) $(OBJ_EXPORTS_SOURCE) $(OBJ_EXPORTS_OBJ) $(OBJ_EXPORTS_STUBS_SOURCE) $(OBJ_EXPORTS_STUBS_OBJ) $(OBJ_EXTRA_CLEAN)
ifeq ($(KEEP_PKG),0)
	rm -f $(PACKAGE_ARTIFACTS)
	rm -rf $(PACKAGE_ARTIFACT_ROOT)
endif
	rm -rf $(BUILD_DIR)

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
