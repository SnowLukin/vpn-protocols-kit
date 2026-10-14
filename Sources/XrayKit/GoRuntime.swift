import Foundation
import VpnFoundation.VpnFoundationCore

/// Garbage collector settings for a managed xray start.
public struct GoGCPolicy: Sendable, Equatable {
    public var gcPercent: Int
    /// `nil` - no soft memory limit.
    public var memoryLimitMiB: Int?
    /// 0 - no periodic FreeOSMemory.
    public var freeOSMemoryIntervalMs: Int
    /// `nil` - keep GOMAXPROCS.
    public var maxProcs: Int?

    public init(gcPercent: Int, memoryLimitMiB: Int?, freeOSMemoryIntervalMs: Int, maxProcs: Int?) {
        self.gcPercent = gcPercent
        self.memoryLimitMiB = memoryLimitMiB
        self.freeOSMemoryIntervalMs = freeOSMemoryIntervalMs
        self.maxProcs = maxProcs
    }
}

/// Process-wide calls into the Go runtime shared by Xray and AmneziaWG.
public enum GoRuntime {
    /// Forces a collection and returns free pages to the OS.
    public static func freeOSMemory() {
        LibXrayFreeOSMemory()
    }

    /// Go memory and GC counters as JSON.
    public static func memStats() -> String? {
        guard let stats = LibXrayGoMemStats() else { return nil }
        defer { free(stats) }
        return String(cString: stats)
    }
}
