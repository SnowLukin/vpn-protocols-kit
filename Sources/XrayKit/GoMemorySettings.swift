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
}
