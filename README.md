# [WireGuard](https://www.wireguard.com/) and Xray for iOS and macOS

## What problem we're solving

Currently there is no way of integrating both WireGuard and Xray protocols together in iOS/macOS app using different libraries. The problem is – different Go runtimes conflicting with each other causing reference issues and Bad access fatal errors.

Solution: Use of the same go runtime for both protocols.

## VpnProtocolsKit integration

1. Open your Xcode project and add the Swift package with the following URL:
   
   ```
   https://github.com/SnowLukin/vpn-protocols-kit
   ```
   
2. VpnFoundation will automatically fetched from GitHub releases.

## VpnProtocolsKit integration for SPM

1. In the `dependencies` array of your `Package.swift` file add the URL and version requirement of the package you want to integrate. For example:

   ```swift
   dependencies: [
      .package(
         url: "https://github.com/SnowLukin/vpn-protocols-kit.git", 
         exact: "1.0.0" // use exact version to avoid conflicts with amenzia releases
      ),
   ],
   ```
2. In the `targets` section, add the imported package as a dependency of the target that needs it:

   ```swift
   targets: [
      .target(
         name: "YourTarget",
         dependencies: ["XrayKit", "WireGuardKit"]
      ),
   ]
   ```

3. In your module use `import XrayKit` or `import WireGuardKit` depending on your needs

## VpnFoundation Update Process

Follow these steps to update VpnFoundation:

1. Go to directory `Tools/xcframework_generation/` and start script:

   ```bash
    cd Tools/xcframework_generation
    ./build.sh
   ```
   The script will remove all the previous .builds and .tmp folders and start building a new xcframework.
   Xcframework will be created in .build folder alongside with zip.

2. Get archive's checksum:

   checksum can be seen in the end of `build.sh` script

   E.g:
   ```
   b546dc09726f18ea8a59f3c8c3df94825694ff3d5163c477a9f381328f8059c5
   ```

3. Download ZIP-file from `.build/` dir. 

   The you can:
   - Integrate it locally in Package.swift
   - Load it to GitHub releases

   Examples:

   **Local integration в Package.swift:**
   ```swift
   .binaryTarget(
      name: "VpnFoundation",
      path: "Tools/xcframework_generation/.build/WireGuardFoundation.xcframework"
   )
   ```

   **Remote integration from GitHub release:**
   ```swift
   .binaryTarget(
      name: "VpnFoundation",
      url: "https://github.com/SnowLukin/vpn-protocols-kit/releases/download/1.0.0/VpnFoundation.xcframework.zip",
      checksum: "b546dc09726f18ea8a59f3c8c3df94825694ff3d5163c477a9f381328f8059c5"
   )
   ```


## MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
