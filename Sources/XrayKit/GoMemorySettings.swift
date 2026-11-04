import Foundation
import VpnFoundation.VpnFoundationCore

/// Collector settings of the Go runtime shared by Xray and AmneziaWG.
public enum GoMemorySettings {
    /// Overrides GOGC and the soft memory limit that libXray forces on iOS at
    /// every xray start, so call it after the start. GOGC below 10 is raised
    /// to 10, `memoryLimitMiB` 0 removes the limit.
    public static func apply(gcPercent: Int, memoryLimitMiB: Int) {
        LibXraySetGoMemorySettings(Int32(clamping: gcPercent), Int32(clamping: memoryLimitMiB))
    }

    /// How often libXray returns memory to the system on iOS, every second by
    /// default; 0 stops that loop. Call it before the xray start so the start
    /// itself runs with this interval.
    public static func setFreeOSMemoryInterval(seconds: Int) {
        LibXraySetFreeOSMemoryInterval(Int32(clamping: seconds))
    }

    /// Runs a full collection and returns as much memory to the system as
    /// possible. Blocks until the collection ends, so keep it off the main
    /// thread.
    public static func freeOSMemory() {
        LibXrayFreeOSMemory()
    }

    /// Limits how many threads run Go code at once; 0 returns to the runtime
    /// default.
    public static func setMaxProcs(_ count: Int) {
        LibXraySetGoMaxProcs(Int32(clamping: count))
    }
}
