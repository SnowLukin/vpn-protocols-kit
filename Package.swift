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
        .binaryTarget(
           name: "VpnFoundation",
           url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/1.0.0/VpnFoundation.xcframework.zip",
           checksum: "46e447508acbacbaf96f1a10c42d6ab3d0ad246f3737141b581254514ed2baf1"
        )

        // Local usage
//        .binaryTarget(
//            name: "VpnFoundation",
//            path: "Tools/xcframework_generation/.build/VpnFoundation.xcframework"
//        ),
    ]
)
