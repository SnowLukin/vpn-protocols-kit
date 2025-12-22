# VAR SETUP

MACOS_PLATFORM_NAME = macosx
MACOS_DEPLOYMENT_TARGET = 12.0
MACOS_ARCHS = x86_64 arm64
IOS_PLATFORM_NAME = iphoneos
IOS_DEPLOYMENT_TARGET = 15.0
IOS_ARCHS = arm64
IOS_SIM_PLATFORM_NAME = iphonesimulator
IOS_SIM_ARCHS = arm64 x86_64
CONFIGURATION_TEMP_DIR ?= $(CURDIR)/.tmp
CONFIGURATION_BUILD_DIR ?= $(CURDIR)/.build
MACOS_CONFIGURATION_BUILD_DIR ?= $(CONFIGURATION_TEMP_DIR)/out-macos
IOS_CONFIGURATION_BUILD_DIR ?= $(CONFIGURATION_TEMP_DIR)/out-ios
IOS_SIM_CONFIGURATION_BUILD_DIR ?= $(CONFIGURATION_TEMP_DIR)/out-ios-sim
XCFRAMEWORK_BASENAME = VpnFoundation
XCFRAMEWORK_NAME = $(CONFIGURATION_BUILD_DIR)/$(XCFRAMEWORK_BASENAME).xcframework
USER := $(shell whoami)

define get_sdkroot
$(shell xcrun --sdk $(1) --show-sdk-path)
endef

MACOS_SDKROOT = $(call get_sdkroot,$(MACOS_PLATFORM_NAME))
IOS_SDKROOT = $(call get_sdkroot,$(IOS_PLATFORM_NAME))
IOS_SIM_SDKROOT = $(call get_sdkroot,$(IOS_SIM_PLATFORM_NAME))

export PATH := $(PATH):/usr/local/bin:/opt/homebrew/bin
export CC ?= clang
LIPO ?= lipo
MACOS_DESTDIR ?= $(MACOS_CONFIGURATION_BUILD_DIR)
IOS_DESTDIR ?= $(IOS_CONFIGURATION_BUILD_DIR)
IOS_SIM_DESTDIR ?= $(IOS_SIM_CONFIGURATION_BUILD_DIR)
BUILDDIR ?= $(CONFIGURATION_TEMP_DIR)/vpnfoundation-go-bridge
HEV_SOCKS5_DIR ?= $(CURDIR)/submodules/hev-socks5-tunnel
HEV_SOCKS5_BUILDDIR ?= $(CONFIGURATION_TEMP_DIR)/hev-socks5-tunnel

MACOS_CFLAGS_PREFIX := -mmacosx-version-min=$(MACOS_DEPLOYMENT_TARGET) -isysroot $(MACOS_SDKROOT) -arch
IOS_CFLAGS_PREFIX := -miphoneos-version-min=$(IOS_DEPLOYMENT_TARGET) -isysroot $(IOS_SDKROOT) -arch
IOS_SIM_CFLAGS_PREFIX := -mios-simulator-version-min=$(IOS_DEPLOYMENT_TARGET) -isysroot $(IOS_SIM_SDKROOT) -arch
GOARCH_arm64 := arm64
GOARCH_x86_64 := amd64
GOOS_macosx := darwin
GOOS_iphoneos := ios
GOOS_iphonesimulator := ios

# -- CONVENIENCE

macos-version-header: $(MACOS_DESTDIR)/wireguard-go-version.h
ios-version-header: $(IOS_DESTDIR)/wireguard-go-version.h $(IOS_SIM_DESTDIR)/wireguard-go-version.h
macos-build: $(MACOS_DESTDIR)/libvpnfoundation-go.a macos-version-header
ios-build: $(IOS_DESTDIR)/libvpnfoundation-go.a $(IOS_SIM_DESTDIR)/libvpnfoundation-go.a ios-version-header
hev-socks5-macos-build: $(MACOS_DESTDIR)/libhev-socks5-tunnel.a
hev-socks5-ios-build: $(IOS_DESTDIR)/libhev-socks5-tunnel.a $(IOS_SIM_DESTDIR)/libhev-socks5-tunnel.a
build: macos-build ios-build hev-socks5-macos-build hev-socks5-ios-build
build-xcframework: $(XCFRAMEWORK_NAME)

REAL_GOROOT := $(shell go env GOROOT 2>/dev/null)
export GOROOT := $(BUILDDIR)/goroot
$(GOROOT)/.prepared:
	[ -n "$(REAL_GOROOT)" ]
	@echo "BUILDDIR: $(BUILDDIR)"
	@echo "GOROOT: $(GOROOT)"
	@echo "CURDIR: $(CURDIR)"
	@echo "PWD: $(PWD)"
	mkdir -p "$(GOROOT)"
	mkdir -p "$(CONFIGURATION_TEMP_DIR)"
	mkdir -p "$(BUILDDIR)/tmp"
	rsync -Dlprth --delete --exclude=pkg/obj/go-build "$(REAL_GOROOT)/" "$(GOROOT)/"
	chmod -R u+w "$(BUILDDIR)"
	cd "$(GOROOT)" && TMPDIR="$(BUILDDIR)/tmp" patch -p1 -f -N -r- < "$(CURDIR)/goruntime-boottime-over-monotonic.diff"
	touch "$@"


# -- COMPILING

# ARCH-$(1) CFLAGS_PREFIX-$(2)  PLATFORM_NAME-$(3)
define libvpnfoundation-go-a
$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a: export CGO_ENABLED := 1
$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a: export CGO_CFLAGS := $(2) $(1)
$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a: export CGO_LDFLAGS := $(2) $(1)
$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a: export GOOS := $(GOOS_$(3))
$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a: export GOARCH := $(GOARCH_$(1))
$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a: $(GOROOT)/.prepared go.mod
	@echo "🟨🟨🟨🟨🟨 building $(3) $(1) 🟨🟨🟨🟨🟨"
	$(info 'target 🎯: $(3) $(1)')
	go build -ldflags=-w -trimpath -v -o "$(BUILDDIR)/libvpnfoundation-go-$(3)-$(1).a" -buildmode c-archive
	rm -f "$(BUILDDIR)/libvpnfoundation-go-$(1).h"
endef

$(foreach ARCH,$(MACOS_ARCHS),$(eval $(call libvpnfoundation-go-a,$(ARCH),$(MACOS_CFLAGS_PREFIX),$(MACOS_PLATFORM_NAME))))
$(foreach ARCH,$(IOS_ARCHS),$(eval $(call libvpnfoundation-go-a,$(ARCH),$(IOS_CFLAGS_PREFIX),$(IOS_PLATFORM_NAME))))
$(foreach ARCH,$(IOS_SIM_ARCHS),$(eval $(call libvpnfoundation-go-a,$(ARCH),$(IOS_SIM_CFLAGS_PREFIX),$(IOS_SIM_PLATFORM_NAME))))


# -- HEV-SOCKS5-TUNNEL BUILDING

# ARCH-$(1) CFLAGS-$(2) PLATFORM_NAME-$(3)
define libhev-socks5-a
$(HEV_SOCKS5_BUILDDIR)/libhev-socks5-tunnel-$(3)-$(1).a:
	@echo "🔵🔵🔵🔵🔵 building hev-socks5-tunnel $(3) $(1) 🔵🔵🔵🔵🔵"
	@mkdir -p "$(HEV_SOCKS5_BUILDDIR)"
	@echo "Building third-party dependencies..."
	cd "$(HEV_SOCKS5_DIR)" && \
		$(MAKE) PP="xcrun --sdk $(3) --toolchain $(3) clang" \
			CC="xcrun --sdk $(3) --toolchain $(3) clang" \
			AR="xcrun ar" \
			CFLAGS="-arch $(1) $(2)" \
			LFLAGS="-arch $(1) $(2) -Wl,-Bsymbolic-functions" tp-static
	@echo "Building hev-socks5-tunnel..."
	cd "$(HEV_SOCKS5_DIR)" && \
		$(MAKE) PP="xcrun --sdk $(3) --toolchain $(3) clang" \
			CC="xcrun --sdk $(3) --toolchain $(3) clang" \
			AR="xcrun ar" \
			CFLAGS="-arch $(1) $(2)" \
			LFLAGS="-arch $(1) $(2) -Wl,-Bsymbolic-functions" static
	@mkdir -p "$$(dir $$@)"
	libtool -static -o "$$@" \
		"$(HEV_SOCKS5_DIR)/bin/libhev-socks5-tunnel.a" \
		"$(HEV_SOCKS5_DIR)/third-part/lwip/bin/liblwip.a" \
		"$(HEV_SOCKS5_DIR)/third-part/yaml/bin/libyaml.a" \
		"$(HEV_SOCKS5_DIR)/third-part/hev-task-system/bin/libhev-task-system.a"
	cd "$(HEV_SOCKS5_DIR)" && $(MAKE) clean
endef

$(eval $(call libhev-socks5-a,arm64,-mmacosx-version-min=$(MACOS_DEPLOYMENT_TARGET),$(MACOS_PLATFORM_NAME)))
$(eval $(call libhev-socks5-a,x86_64,-mmacosx-version-min=$(MACOS_DEPLOYMENT_TARGET),$(MACOS_PLATFORM_NAME)))
$(eval $(call libhev-socks5-a,arm64,-miphoneos-version-min=$(IOS_DEPLOYMENT_TARGET),$(IOS_PLATFORM_NAME)))
$(eval $(call libhev-socks5-a,arm64,-mios-simulator-version-min=$(IOS_DEPLOYMENT_TARGET),$(IOS_SIM_PLATFORM_NAME)))
$(eval $(call libhev-socks5-a,x86_64,-mios-simulator-version-min=$(IOS_DEPLOYMENT_TARGET),$(IOS_SIM_PLATFORM_NAME)))


# -- HEADERS STUFF

define version-header-rule
$(1)/wireguard-go-version.h: go.mod $$(GOROOT)/.prepared
	@mkdir -p "$$(dir $$@)"
	sed -E -n 's/.*golang\.zx2c4\.com\/wireguard +v[0-9.]+-([0-9]+-[0-9a-f]{8})[0-9a-f]{4}.*/#define WIREGUARD_GO_VERSION "\1"/p' "$$<" > "$$@"
endef

$(eval $(call version-header-rule,$(MACOS_DESTDIR)))
$(eval $(call version-header-rule,$(IOS_DESTDIR)))
$(eval $(call version-header-rule,$(IOS_SIM_DESTDIR)))

# -- LINKING

# DESTDIR: $(1) ARCHS: $(2) PLATFORM_NAME: $(3)
define linkinator
$(1)/libvpnfoundation-go.a: $(foreach ARCH,$(2),$(BUILDDIR)/libvpnfoundation-go-$(3)-$(ARCH).a)
	@mkdir -vp "$(1)"
	$(LIPO) -create -output "$$@" $$^
endef

$(eval $(call linkinator,$(MACOS_DESTDIR),$(MACOS_ARCHS),$(MACOS_PLATFORM_NAME)))
$(eval $(call linkinator,$(IOS_DESTDIR),$(IOS_ARCHS),$(IOS_PLATFORM_NAME)))
$(eval $(call linkinator,$(IOS_SIM_DESTDIR),$(IOS_SIM_ARCHS),$(IOS_SIM_PLATFORM_NAME)))

# -- HEV-SOCKS5-TUNNEL LINKING

# DESTDIR: $(1) ARCHS: $(2) PLATFORM_NAME: $(3)
define hev-linkinator
$(1)/libhev-socks5-tunnel.a: $(foreach ARCH,$(2),$(HEV_SOCKS5_BUILDDIR)/libhev-socks5-tunnel-$(3)-$(ARCH).a)
	@mkdir -vp "$(1)"
	$(LIPO) -create -output "$$@" $$^
endef

$(eval $(call hev-linkinator,$(MACOS_DESTDIR),$(MACOS_ARCHS),$(MACOS_PLATFORM_NAME)))
$(eval $(call hev-linkinator,$(IOS_DESTDIR),$(IOS_ARCHS),$(IOS_PLATFORM_NAME)))
$(eval $(call hev-linkinator,$(IOS_SIM_DESTDIR),$(IOS_SIM_ARCHS),$(IOS_SIM_PLATFORM_NAME)))

# -- MERGE BOTH LIBRARIES

# DESTDIR: $(1)
define merge-libraries
$(1)/libvpnfoundation-combined.a: $(1)/libvpnfoundation-go.a $(1)/libhev-socks5-tunnel.a
	@echo "🔗 Merging libraries for $(1)"
	libtool -static -o "$$@" "$(1)/libvpnfoundation-go.a" "$(1)/libhev-socks5-tunnel.a"
endef

$(eval $(call merge-libraries,$(MACOS_DESTDIR)))
$(eval $(call merge-libraries,$(IOS_DESTDIR)))
$(eval $(call merge-libraries,$(IOS_SIM_DESTDIR)))


# -- XCFRAMEWORK COMPILING

$(XCFRAMEWORK_NAME): $(MACOS_DESTDIR)/libvpnfoundation-combined.a $(IOS_DESTDIR)/libvpnfoundation-combined.a $(IOS_SIM_DESTDIR)/libvpnfoundation-combined.a ios-version-header macos-version-header
	@mkdir -p "$(CONFIGURATION_BUILD_DIR)"
	rm -rf "$(XCFRAMEWORK_NAME)"

	# Create temporary directory for headers
	@mkdir -p "$(BUILDDIR)/headers-temp"

	# Copy VpnFoundation headers
	cp Headers/vpnfoundation.h "$(BUILDDIR)/headers-temp/vpnfoundation.h"

	# Add WireGuard version
	cat "$(IOS_DESTDIR)/wireguard-go-version.h" >> "$(BUILDDIR)/headers-temp/vpnfoundation.h"

	# Copy HevSocks5Tunnel header
	cp "$(HEV_SOCKS5_DIR)/src/hev-main.h" "$(BUILDDIR)/headers-temp/hev-main.h"

	# Create unified modulemap with submodules
	@echo "Creating unified module.modulemap..."
	@echo "module VpnFoundation {" > "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "    explicit module VpnFoundationCore {" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "        header \"vpnfoundation.h\"" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "        link \"vpnfoundation-combined\"" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "        export *" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "    }" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "    explicit module HevSocks5Tunnel {" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "        header \"hev-main.h\"" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "        link \"vpnfoundation-combined\"" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "        export *" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "    }" >> "$(BUILDDIR)/headers-temp/module.modulemap"
	@echo "}" >> "$(BUILDDIR)/headers-temp/module.modulemap"

	xcrun xcodebuild -create-xcframework \
		-library $(IOS_DESTDIR)/libvpnfoundation-combined.a \
		-headers "$(BUILDDIR)/headers-temp" \
		-library $(IOS_SIM_DESTDIR)/libvpnfoundation-combined.a \
		-headers "$(BUILDDIR)/headers-temp" \
		-library $(MACOS_DESTDIR)/libvpnfoundation-combined.a \
		-headers "$(BUILDDIR)/headers-temp" \
		-output "$(XCFRAMEWORK_BASENAME).xcframework"

	mv "$(XCFRAMEWORK_BASENAME).xcframework" "$(CONFIGURATION_BUILD_DIR)/"

	@echo "📦 Creating ZIP archive..."
	cd "$(CONFIGURATION_BUILD_DIR)" && zip -r "$(XCFRAMEWORK_BASENAME).xcframework.zip" "$(XCFRAMEWORK_BASENAME).xcframework"

	@echo "🔐 Calculating checksum..."
	@cd "$(CONFIGURATION_BUILD_DIR)" && shasum -a 256 "$(XCFRAMEWORK_BASENAME).xcframework.zip" | cut -d' ' -f1 | xargs echo "Checksum:"

	@echo "🧹 Cleaning up temporary build files..."
	$(MAKE) clean-temp

	@echo "✅ XCFramework built and archived successfully!"


# -- CLEANING

# BUILDDIR: $(1) DESTDIR: $(2)
define clean-output
rm -rf "$(1)" "$(2)"
endef

clean-temp:
	$(call clean-output,$(BUILDDIR),$(MACOS_DESTDIR))
	$(call clean-output,$(BUILDDIR),$(IOS_DESTDIR))
	$(call clean-output,$(BUILDDIR),$(IOS_SIM_DESTDIR))
	rm -rf "$(CONFIGURATION_TEMP_DIR)"
	rm -rf "$(BUILDDIR)/headers-temp"
	rm -rf "$(HEV_SOCKS5_BUILDDIR)"
	rm -f Headers/vpnfoundation.h.tmp
	cd "$(HEV_SOCKS5_DIR)" && $(MAKE) clean || true

clean:
	$(call clean-output,$(BUILDDIR),$(MACOS_DESTDIR))
	$(call clean-output,$(BUILDDIR),$(IOS_DESTDIR))
	$(call clean-output,$(BUILDDIR),$(IOS_SIM_DESTDIR))
	rm -rf "$(CONFIGURATION_TEMP_DIR)"
	rm -rf "$(BUILDDIR)/headers-temp"
	rm -rf "$(HEV_SOCKS5_BUILDDIR)"
	rm -f Headers/vpnfoundation.h.tmp
	rm -rf "$(CONFIGURATION_BUILD_DIR)"
	rm -f "$(XCFRAMEWORK_BASENAME).xcframework.zip"
	cd "$(HEV_SOCKS5_DIR)" && $(MAKE) clean || true

install-macos: macos-build
install-ios: ios-build
install: build

rebuild: clean build-xcframework

.PHONY: clean clean-temp macos-build ios-build version-header install-macos install-ios debug build install rebuild