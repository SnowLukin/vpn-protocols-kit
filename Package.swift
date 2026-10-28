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
          url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/2.4.0-beta.5/VpnFoundation.xcframework.zip",
          checksum: "770bc10ea79c1a19c29c2c78bec404bebf57b732402597dbaed6de0f54873088"
       )

        // Local usage
        // .binaryTarget(
        //     name: "VpnFoundation",
        //     path: "Tools/xcframework_generation/.build/VpnFoundation.xcframework"
        // ),
    ]
)
