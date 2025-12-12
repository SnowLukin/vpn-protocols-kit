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
           url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/1.1.0/VpnFoundation.xcframework.zip",
           checksum: "4f0621a1addadb02e73a189c88e8c4681811bbff5cb5961ce9f81c68d72c3ab7"
        )

        // Local usage
    //    .binaryTarget(
    //        name: "VpnFoundation",
    //        path: "Tools/xcframework_generation/.build/VpnFoundation.xcframework"
    //    ),
    ]
)
