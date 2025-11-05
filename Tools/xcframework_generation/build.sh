#!/bin/bash

# VPNProtocolsKit XCFramework Build Script
# This script builds the XCFramework for VPNProtocolsKit
# Run this script from Tools/xcframework_generation/ directory

set -e  # Exit on any error

echo "🚀 Starting VPNProtocolsKit XCFramework build..."
echo "📂 Working directory: $(pwd)"

# Check if we're in the right directory
if [ ! -f "Makefile" ] || [ ! -f "go.mod" ]; then
    echo "❌ Error: Please run this script from Tools/xcframework_generation/ directory"
    echo "   cd Tools/xcframework_generation/"
    exit 1
fi

echo "🔧 Cleaning up previous build..."
make clean

echo "🔧 Building XCFramework..."

# Run the build
make build-xcframework

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "✅ Build completed successfully!"
    echo "📦 XCFramework created at: .build/VPNProtocolsFoundation.xcframework"
    echo "📦 ZIP archive created at: .build/VPNProtocolsFoundation.xcframework.zip"
    
    # Show checksum if available
    if [ -f ".build/VPNProtocolsFoundation.xcframework.zip" ]; then
        CHECKSUM=$(shasum -a 256 ".build/VPNProtocolsFoundation.xcframework.zip" | cut -d' ' -f1)
        echo "🔐 Checksum (SHA256): $CHECKSUM"
    fi
else
    echo "❌ Build failed!"
    exit 1
fi