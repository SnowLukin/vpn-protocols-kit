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
        .testTarget(
            name: "Tun2SocksKitTests",
            dependencies: ["Tun2SocksKit"]
        ),
        // Remote usage
       .binaryTarget(
          name: "VpnFoundation",
          url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/2.3.0-beta.4/VpnFoundation.xcframework.zip",
          checksum: "639834fb8d3dfe5c9b34ca5f3bb11f782a99ba7add302f5d69784847486f009e"
       )

        // Local usage
        // .binaryTarget(
        //     name: "VpnFoundation",
        //     path: "Tools/xcframework_generation/.build/VpnFoundation.xcframework"
        // ),
    ]
)
