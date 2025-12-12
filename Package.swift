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
        .library(name: "WireGuardKit", targets: ["WireGuardKit"])
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
        // Release usage
        .binaryTarget(
           name: "VpnFoundation",
           url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/1.1.3/VpnFoundation.xcframework.zip",
           checksum: "d1956cd622e0398410e6733312219e891dc2cc55ad629aabd2c94b9cce356d3a"
        )

        // Local usage
    //    .binaryTarget(
    //        name: "VpnFoundation",
    //        path: "Tools/xcframework_generation/.build/VpnFoundation.xcframework"
    //    ),
    ]
)
