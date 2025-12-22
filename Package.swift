// swift-tools-version: 5.5
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "VpnProtocolsKit",
    platforms: [
        .macOS(.v12),
        .iOS(.v15),
    ],
    products: [
        .library(name: "XrayKit", targets: ["XrayKit"]),
        .library(name: "WireGuardKit", targets: ["WireGuardKit"]),
        .library(name: "Tun2SocksKit", targets: ["Tun2SocksKit"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "XrayKit",
            dependencies: ["VpnFoundation"]
        ),
        .target(
            name: "WireGuardKit",
            dependencies: ["VpnFoundation", "WireGuardKitC"]
        ),
        .target(
            name: "WireGuardKitC",
            dependencies: [],
            publicHeadersPath: "."
        ),
        .target(
            name: "Tun2SocksKit",
            dependencies: ["VpnFoundation", "WireGuardKitC"]
        ),
        // Remote usage
       .binaryTarget(
          name: "VpnFoundation",
          url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/2.0.0/VpnFoundation.xcframework.zip",
          checksum: "538584d2f9cb1818ce62674ee74287a106d5933c1b55f3477c6144b0a12ff419"
       )

        // Local usage
        // .binaryTarget(
        //     name: "VpnFoundation",
        //     path: "Tools/xcframework_generation/.build/VpnFoundation.xcframework"
        // ),
    ]
)
