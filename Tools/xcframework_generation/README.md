# VpnFoundation XCFramework

This directory contains build scripts for creating a unified XCFramework that combines:
- **VpnFoundation** (WireGuard + Xray protocols)
- **HevSocks5Tunnel** (SOCKS5 tunnel implementation)

## Building

### Prerequisites

1. Xcode and Command Line Tools installed
2. Go installed (for WireGuard and Xray compilation)
3. Git submodules initialized:

```bash
git submodule update --init --recursive
cd submodules/hev-socks5-tunnel
git submodule update --init --recursive
```

### Build Commands

```bash
# Clean previous builds
make clean

# Build the complete XCFramework
make build-xcframework

# Build individual components
make macos-build           # VpnFoundation for macOS
make ios-build             # VpnFoundation for iOS
make hev-socks5-macos-build  # HevSocks5Tunnel for macOS
make hev-socks5-ios-build    # HevSocks5Tunnel for iOS
```

### Output

The build produces:
- `.build/VpnFoundation.xcframework` - The combined framework
- `.build/VpnFoundation.xcframework.zip` - Zipped archive for distribution
- Checksum printed at the end of build

## Using in Swift

The unified XCFramework exports two explicit submodules:

### VpnFoundationCore (WireGuard + Xray)

```swift
import VpnFoundation.VpnFoundationCore

// WireGuard API
wgSetLogger(context, loggerCallback)
let handle = wgTurnOn(config, tunFd)
wgTurnOff(handle)

// Xray API
LibXrayRunXray(datDir, configPath, maxMemory)
LibXrayStopXray()
```

### HevSocks5Tunnel

```swift
import VpnFoundation.HevSocks5Tunnel

// Start SOCKS5 tunnel
let result = hev_socks5_tunnel_main_from_str(configData, configLen, tunFd)

// Get statistics
var txPackets: size_t = 0
var txBytes: size_t = 0
var rxPackets: size_t = 0
var rxBytes: size_t = 0
hev_socks5_tunnel_stats(&txPackets, &txBytes, &rxPackets, &rxBytes)

// Stop tunnel
hev_socks5_tunnel_quit()
```

## Module Map Structure

The unified `module.modulemap` looks like:

```
module VpnFoundation {
    explicit module VpnFoundationCore {
        header "vpnfoundation.h"
        link "vpnfoundation-combined"
        export *
    }

    explicit module HevSocks5Tunnel {
        header "hev-main.h"
        link "vpnfoundation-combined"
        export *
    }
}
```

Both submodules link to the same combined static library (`libvpnfoundation-combined.a`), which resolves the "Multiple commands produce module.modulemap" conflict.

## Why This Approach?

Previously, both VpnFoundation and Tun2SocksKit (which uses HevSocks5Tunnel) had separate XCFrameworks with their own `module.modulemap` files. This caused conflicts during Xcode builds:

```
Multiple commands produce '.../include/module.modulemap'
```

By combining both libraries into a single XCFramework with explicit submodules, we:
1. ✅ Eliminate the modulemap naming conflict
2. ✅ Keep both APIs separate and clean
3. ✅ Reduce the number of dependencies
4. ✅ Maintain backward compatibility with the same function signatures

## Architecture

The combined library includes:

### From VpnFoundation (Go):
- WireGuard Go implementation
- Xray Core implementation

### From HevSocks5Tunnel (C):
- SOCKS5 tunnel core
- lwIP TCP/IP stack
- libyaml configuration parser
- hev-task-system coroutine library

## Troubleshooting

### Build fails with "No rule to make target clean"
This is expected and safe to ignore. The error occurs when cleaning non-existent third-party builds.

### Build fails with "lwip/tcp.h file not found"
Initialize hev-socks5-tunnel submodules:
```bash
cd submodules/hev-socks5-tunnel
git submodule update --init --recursive
```

### "Operation not permitted" on .tmp directory
This can happen in sandboxed environments. The build will retry or use alternative paths.

## Updating Package.swift

When updating the framework version in your Swift Package:

```swift
.binaryTarget(
    name: "VpnFoundation",
    url: "https://github.com/YOUR_REPO/releases/download/VERSION/VpnFoundation.xcframework.zip",
    checksum: "USE_CHECKSUM_FROM_BUILD_OUTPUT"
)
```

The checksum is printed at the end of successful builds.

