PROGRAM ?= app
PACKAGE_NAME ?= $(PROGRAM)
ROOT ?= ../..
ROOT_ABS := $(abspath $(ROOT))
STDLIB_DIR_ABS := $(ROOT_ABS)/stdlib
FIRST_PARTY_DIRS ?= platform
THIRD_PARTY_DIRS ?= third_party
RUNTIME_FLAVOR ?= bootstrap
ifeq ($(RUNTIME_FLAVOR),libgo)
FIRST_PARTY_DIRS := $(FIRST_PARTY_DIRS) native/libgo/staging/go
BUILD_CACHE_NAMESPACE ?= libgo
endif
FIRST_PARTY_DIRS_ABS := $(foreach dir,$(FIRST_PARTY_DIRS),$(if $(filter /%,$(dir)),$(dir),$(ROOT_ABS)/$(dir)))
THIRD_PARTY_DIRS_ABS := $(foreach dir,$(THIRD_PARTY_DIRS),$(if $(filter /%,$(dir)),$(dir),$(ROOT_ABS)/$(dir)))
BUILD_CACHE_ROOT ?= $(ROOT)/.build-cache
BUILD_CACHE_NAMESPACE ?=
BUILD_CACHE_BASE := $(BUILD_CACHE_ROOT)$(if $(strip $(BUILD_CACHE_NAMESPACE)),/$(BUILD_CACHE_NAMESPACE),)
BUILD_CACHE_BASE_ABS := $(abspath $(BUILD_CACHE_BASE))
PACKAGE_ARTIFACT_ROOT := $(BUILD_CACHE_BASE)/pkg
PACKAGE_ARTIFACT_ROOT_ABS := $(BUILD_CACHE_BASE_ABS)/pkg
ABI_ARTIFACT_ROOT := $(BUILD_CACHE_BASE)/abi

ABI_DIR = $(ROOT)/platform/abi
MK_DIR = $(ROOT)/tooling
BUILD_DIR = .build
GOOS_TARGET ?= kolibrios
GOARCH_TARGET ?= 386
BUILD_TAGS ?= gccgo
SELECT_GO_FILES := $(MK_DIR)/select-go-files.py

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
GCC_TOOL_PREFIX ?= $(if $(wildcard $(TOOLING_BIN)/as),-B$(TOOLING_BIN),)
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
KEEP_PKG ?= 1
KEEP_ABI ?= 1
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

GO_COMPILER_FLAGS = -m32 $(GCC_TOOL_PREFIX) -c $(GO_OPT_LEVEL) -nostdlib -nostdinc -fexceptions -fno-stack-protector -fno-split-stack -static -fno-leading-underscore -fno-common -fno-pie -g -ffunction-sections -fdata-sections -I. -I$(ROOT_ABS) -I$(PACKAGE_ARTIFACT_ROOT_ABS)
GCC_COMPILER_FLAGS = -m32 $(GCC_TOOL_PREFIX) -c $(OPT_LEVEL) -ffunction-sections -fdata-sections -fno-pic -fno-pie -fno-stack-protector -fno-builtin-calloc
PACKAGE_NATIVE_INCLUDE_FLAGS :=
ABI_RUNTIME_CPPFLAGS :=
ABI_RUNTIME_INCLUDE_FLAGS :=
ABI_RUNTIME_SOURCE := $(ABI_DIR)/runtime_gccgo.c
ABI_RUNTIME_DEPS :=
ABI_EXTRA_RUNTIME_OBJS :=
LIBGO_RUNTIME_GLOBALIZE_SYMBOLS :=
ifeq ($(RUNTIME_FLAVOR),libgo)
PACKAGE_NATIVE_INCLUDE_FLAGS += -I$(ROOT_ABS)/native/libgo/overlay/include -I$(ROOT_ABS)/native/libgo/staging/runtime
ABI_RUNTIME_CPPFLAGS += -DKOLIBRI_USE_LIBGO_RUNTIME=1
ABI_RUNTIME_INCLUDE_FLAGS += -I$(ROOT_ABS)/native/libgo/staging/runtime
ABI_RUNTIME_SOURCE := $(ABI_DIR)/runtime_libgo_bridge.c
LIBGO_RUNTIME_GLOBALIZE_SYMBOLS := runtime.writeBarrier runtime.pointerequal..f runtime.memequal0..f runtime.memequal8..f runtime.memequal16..f runtime.memequal32..f runtime.memequal64..f runtime.memequal128..f runtime.f32equal..f runtime.f64equal..f runtime.c64equal..f runtime.c128equal..f runtime.strequal..f runtime.interequal..f runtime.nilinterequal..f
endif
LDFLAGS = -n -T $(LDSCRIPT) -m elf_i386 -z noexecstack -z relro -z now --gc-sections --eh-frame-hdr --entry=$(ENTRYPOINT)

APP_SOURCES = $(shell $(SELECT_GO_FILES) --package-dir $(CURDIR) --goos $(GOOS_TARGET) --goarch $(GOARCH_TARGET) --tags "$(BUILD_TAGS)")
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
RESOLVED_PACKAGE_DIRS := $(shell $(MK_DIR)/resolve-packages.py --root $(ROOT_ABS) --stdlib $(STDLIB_DIR_ABS) --first-party "$(FIRST_PARTY_DIRS)" --third-party "$(THIRD_PARTY_DIRS)" --app-dir $(CURDIR) --packages "$(PACKAGE_DIRS)" --goos $(GOOS_TARGET) --goarch $(GOARCH_TARGET) --tags "$(BUILD_TAGS)")
override PACKAGE_DIRS := $(RESOLVED_PACKAGE_DIRS)
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
PACKAGE_DIRS_FILE_$(1) := $$(PACKAGE_SOURCE_DIR_$(1))/package_dirs.txt
PACKAGE_SOURCE_SUBDIRS_$(1) := $$(strip $$(shell if [ -f "$$(PACKAGE_DIRS_FILE_$(1))" ]; then sed -e 's/#.*//' -e '/./!d' "$$(PACKAGE_DIRS_FILE_$(1))"; fi))
PACKAGE_SOURCE_DIRS_$(1) := $$(PACKAGE_SOURCE_DIR_$(1)) $$(foreach rel,$$(PACKAGE_SOURCE_SUBDIRS_$(1)),$$(PACKAGE_SOURCE_DIR_$(1))/$$(rel))
PACKAGE_SOURCES_$(1) := $$(shell $(SELECT_GO_FILES) --package-dir $$(PACKAGE_SOURCE_DIR_$(1)) --goos $(GOOS_TARGET) --goarch $(GOARCH_TARGET) --tags "$(BUILD_TAGS)")
PACKAGE_SOURCE_FILES_$(1) := $$(patsubst $$(PACKAGE_SOURCE_DIR_$(1))/%,%,$$(PACKAGE_SOURCES_$(1)))
PACKAGE_ARTIFACT_PREFIX_$(1) := $(PACKAGE_ARTIFACT_ROOT)/$(1)
PACKAGE_ARTIFACT_PREFIX_ABS_$(1) := $(PACKAGE_ARTIFACT_ROOT_ABS)/$(1)
PACKAGE_GO_OBJ_$(1) := $$(PACKAGE_ARTIFACT_PREFIX_$(1)).gccgo.go.o
PACKAGE_NATIVE_OBJ_DIR_$(1) := $$(PACKAGE_ARTIFACT_PREFIX_$(1)).native
PACKAGE_C_SOURCES_$(1) := $$(strip $$(foreach dir,$$(PACKAGE_SOURCE_DIRS_$(1)),$$(wildcard $$(dir)/*.c)))
PACKAGE_ASM_SOURCES_$(1) := $$(strip $$(foreach dir,$$(PACKAGE_SOURCE_DIRS_$(1)),$$(wildcard $$(dir)/*.S)))
PACKAGE_C_SOURCE_FILES_$(1) := $$(patsubst $$(PACKAGE_SOURCE_DIR_$(1))/%,%,$$(PACKAGE_C_SOURCES_$(1)))
PACKAGE_ASM_SOURCE_FILES_$(1) := $$(patsubst $$(PACKAGE_SOURCE_DIR_$(1))/%,%,$$(PACKAGE_ASM_SOURCES_$(1)))
PACKAGE_C_OBJS_$(1) := $$(patsubst %.c,$$(PACKAGE_NATIVE_OBJ_DIR_$(1))/%.o,$$(PACKAGE_C_SOURCE_FILES_$(1)))
PACKAGE_ASM_OBJS_$(1) := $$(patsubst %.S,$$(PACKAGE_NATIVE_OBJ_DIR_$(1))/%.o,$$(PACKAGE_ASM_SOURCE_FILES_$(1)))
PACKAGE_NATIVE_OBJS_$(1) := $$(PACKAGE_C_OBJS_$(1)) $$(PACKAGE_ASM_OBJS_$(1))
PACKAGE_OBJ_$(1) := $$(PACKAGE_ARTIFACT_PREFIX_$(1)).gccgo.o
PACKAGE_GOX_$(1) := $$(PACKAGE_ARTIFACT_PREFIX_$(1)).gox
PACKAGE_GLOBALIZE_SYMBOLS_$(1) :=
ifneq ($(RUNTIME_FLAVOR),libgo)
PACKAGE_GLOBALIZE_SYMBOLS_$(1) :=
else ifeq ($(1),runtime)
PACKAGE_GLOBALIZE_SYMBOLS_$(1) := $(LIBGO_RUNTIME_GLOBALIZE_SYMBOLS)
endif
ifneq ($(FAST_PKG),0)
PACKAGE_ORDER_DEPS_$(1) := $$(if $$(PREVIOUS_PACKAGE_GOXS),| $$(PREVIOUS_PACKAGE_GOXS),)
else
PACKAGE_ORDER_DEPS_$(1) := $$(PREVIOUS_PACKAGE_GOXS)
endif

PACKAGE_OBJS += $$(PACKAGE_OBJ_$(1))
PACKAGE_GOXS += $$(PACKAGE_GOX_$(1))

$$(PACKAGE_GO_OBJ_$(1)): $$(PACKAGE_SOURCES_$(1)) $$(PACKAGE_ORDER_DEPS_$(1))
	mkdir -p $$(dir $$@)
	cd $$(PACKAGE_SOURCE_DIR_$(1)) && $(GO) $(GO_COMPILER_FLAGS) -fgo-pkgpath=$(1) -o $$(PACKAGE_ARTIFACT_PREFIX_ABS_$(1)).gccgo.go.o $$(PACKAGE_SOURCE_FILES_$(1))
	$$(if $$(strip $$(PACKAGE_GLOBALIZE_SYMBOLS_$(1))),for sym in $$(PACKAGE_GLOBALIZE_SYMBOLS_$(1)); do $(OBJCOPY) --globalize-symbol=$$$$sym $$@; done)

$$(PACKAGE_NATIVE_OBJ_DIR_$(1))/%.o: $$(PACKAGE_SOURCE_DIR_$(1))/%.c
	mkdir -p $$(dir $$@)
	$(GCC) $(GCC_COMPILER_FLAGS) -I$$(PACKAGE_SOURCE_DIR_$(1)) -I$(ROOT_ABS) -I$(PACKAGE_ARTIFACT_ROOT_ABS) $(PACKAGE_NATIVE_INCLUDE_FLAGS) $$< -o $$@

$$(PACKAGE_NATIVE_OBJ_DIR_$(1))/%.o: $$(PACKAGE_SOURCE_DIR_$(1))/%.S
	mkdir -p $$(dir $$@)
	$(GCC) $(GCC_COMPILER_FLAGS) -I$$(PACKAGE_SOURCE_DIR_$(1)) -I$(ROOT_ABS) -I$(PACKAGE_ARTIFACT_ROOT_ABS) $(PACKAGE_NATIVE_INCLUDE_FLAGS) $$< -o $$@

$$(PACKAGE_OBJ_$(1)): $$(PACKAGE_GO_OBJ_$(1)) $$(PACKAGE_NATIVE_OBJS_$(1))
	mkdir -p $$(dir $$@)
	$(LD) -r -m elf_i386 -o $$@ $$^

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

ABI_SYSCALLS_OBJ = $(ABI_ARTIFACT_ROOT)/syscalls_i386.o
ABI_SYSCALLS_ALIAS_OBJ =
ABI_RUNTIME_OBJ = $(ABI_ARTIFACT_ROOT)/runtime_gccgo.o
ABI_UNWIND_OBJ = $(ABI_ARTIFACT_ROOT)/go-unwind.o
ABI_CONTEXT_OBJ = $(ABI_ARTIFACT_ROOT)/runtime_context_386.o
ifeq ($(RUNTIME_FLAVOR),libgo)
LIBGO_RUNTIME_DIR := $(ROOT_ABS)/native/libgo/staging/runtime
LIBGO_RUNTIME_INC := $(LIBGO_RUNTIME_DIR)/runtime.inc
LIBGO_RUNTIME_INC_RAW := $(ABI_ARTIFACT_ROOT)/runtime.inc.raw
LIBGO_RUNTIME_INC_OBJ := $(ABI_ARTIFACT_ROOT)/runtime.inc.go.o
LIBGO_RUNTIME_INC_TOOL := $(ROOT_ABS)/tooling/generate-libgo-runtime-inc.sh
ABI_RUNTIME_DEPS += $(LIBGO_RUNTIME_INC)
ABI_UNWIND_OBJ :=
ABI_SYSCALLS_ALIAS_OBJ := $(ABI_ARTIFACT_ROOT)/syscalls_i386_libgo_alias.o
LIBGO_RUNTIME_EXTRA_SRCS := $(LIBGO_RUNTIME_DIR)/aeshash.c $(LIBGO_RUNTIME_DIR)/go-construct-map.c $(LIBGO_RUNTIME_DIR)/go-memclr.c $(LIBGO_RUNTIME_DIR)/go-memequal.c $(LIBGO_RUNTIME_DIR)/go-memmove.c $(LIBGO_RUNTIME_DIR)/go-nanotime.c $(LIBGO_RUNTIME_DIR)/go-now.c $(LIBGO_RUNTIME_DIR)/go-unsafe-pointer.c $(LIBGO_RUNTIME_DIR)/go-unwind.c $(LIBGO_RUNTIME_DIR)/print.c $(LIBGO_RUNTIME_DIR)/runtime_c.c $(LIBGO_RUNTIME_DIR)/yield.c
ABI_EXTRA_RUNTIME_OBJS := $(patsubst $(LIBGO_RUNTIME_DIR)/%.c,$(ABI_ARTIFACT_ROOT)/libgo-runtime/%.o,$(LIBGO_RUNTIME_EXTRA_SRCS))
ABI_EXTRA_RUNTIME_OBJS += $(ABI_ARTIFACT_ROOT)/libc_compat.o
endif
ABI_OBJS = $(ABI_SYSCALLS_OBJ) $(ABI_SYSCALLS_ALIAS_OBJ) $(ABI_RUNTIME_OBJ) $(ABI_UNWIND_OBJ) $(ABI_CONTEXT_OBJ) $(ABI_EXTRA_RUNTIME_OBJS)
STARTUP_ARTIFACTS = $(STARTUP_SOURCE) $(STARTUP_OBJ)
OBJS = $(ABI_OBJS) $(PACKAGE_OBJS) $(APP_OBJ)
OBJS += $(STARTUP_OBJ)
PACKAGE_ARTIFACTS = $(PACKAGE_OBJS) $(PACKAGE_GOXS)
INTERMEDIATE_ARTIFACTS = $(ABI_OBJS) $(APP_OBJ) $(LDSCRIPT) $(STARTUP_ARTIFACTS)

.DEFAULT_GOAL := all
.PHONY: all clean clean-cache distclean link

all: $(PROGRAM).kex

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(APP_OBJ) $(PROGRAM).kex

clean-cache:
	rm -rf $(BUILD_CACHE_BASE)

distclean: clean clean-cache

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

$(ABI_RUNTIME_OBJ): $(ABI_RUNTIME_SOURCE) $(ABI_RUNTIME_DEPS)
	mkdir -p $(dir $@)
	$(GCC) $(GCC_COMPILER_FLAGS) $(ABI_RUNTIME_CPPFLAGS) $(ABI_RUNTIME_INCLUDE_FLAGS) -fexceptions $< -o $@

ifeq ($(RUNTIME_FLAVOR),libgo)
$(LIBGO_RUNTIME_INC): $(PACKAGE_SOURCES_runtime) $(PACKAGE_ORDER_DEPS_runtime) $(LIBGO_RUNTIME_INC_TOOL)
	mkdir -p $(ABI_ARTIFACT_ROOT)
	cd $(PACKAGE_SOURCE_DIR_runtime) && $(GO) $(GO_COMPILER_FLAGS) -fgo-pkgpath=runtime -fgo-compiling-runtime -fgo-c-header=$(abspath $(LIBGO_RUNTIME_INC_RAW)) -o $(abspath $(LIBGO_RUNTIME_INC_OBJ)) $(PACKAGE_SOURCE_FILES_runtime)
	$(LIBGO_RUNTIME_INC_TOOL) $(abspath $(LIBGO_RUNTIME_INC_RAW)) $(abspath $@)

$(ABI_ARTIFACT_ROOT)/libgo-runtime/%.o: $(ROOT_ABS)/native/libgo/staging/runtime/%.c $(LIBGO_RUNTIME_INC)
	mkdir -p $(dir $@)
	$(GCC) $(GCC_COMPILER_FLAGS) $(ABI_RUNTIME_CPPFLAGS) $(ABI_RUNTIME_INCLUDE_FLAGS) -fexceptions $< -o $@

$(ABI_ARTIFACT_ROOT)/libc_compat.o: $(ABI_DIR)/libc_compat.c
	mkdir -p $(dir $@)
	$(GCC) $(GCC_COMPILER_FLAGS) $(ABI_RUNTIME_CPPFLAGS) -fexceptions $< -o $@
endif

$(ABI_SYSCALLS_OBJ): $(ABI_DIR)/syscalls_i386.asm
	mkdir -p $(dir $@)
	$(NASM) $< -o $@

$(ABI_SYSCALLS_ALIAS_OBJ): $(ABI_DIR)/syscalls_i386_libgo_alias.asm
	mkdir -p $(dir $@)
	$(NASM) $< -o $@

$(ABI_UNWIND_OBJ): $(ABI_DIR)/go-unwind.c
	mkdir -p $(dir $@)
	$(GCC) $(GCC_COMPILER_FLAGS) -fexceptions $< -o $@

$(ABI_CONTEXT_OBJ): $(ABI_DIR)/runtime_context_386.S
	mkdir -p $(dir $@)
	$(GCC) $(GCC_COMPILER_FLAGS) $< -o $@
