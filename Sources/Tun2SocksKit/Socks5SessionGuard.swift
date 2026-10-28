import Foundation
import VpnFoundation.HevSocks5Tunnel
#if os(iOS)
import os
#endif

/// Session kinds for `relieve` and the memory probe.
public struct Socks5SessionTypes: OptionSet, Sendable {
    public let rawValue: Int32

    public init(rawValue: Int32) {
        self.rawValue = rawValue
    }

    public static let udp = Socks5SessionTypes(rawValue: Int32(HEV_SOCKS5_TUNNEL_SESSION_UDP))
    public static let tcp = Socks5SessionTypes(rawValue: Int32(HEV_SOCKS5_TUNNEL_SESSION_TCP))
}

/// Whether hev lets new sessions in.
public enum Socks5Admission: Int32, Sendable {
    /// New sessions are limited only by the caps.
    case open = 0
    /// No growth: a new session only takes the place of a silent one of the same kind.
    case hold = 1
    /// No new sessions, live ones keep working.
    case closed = 2
}

/// Bytes the process can still allocate before its limit, negative if unknown.
/// hev calls it on its own thread at most once per
/// `Socks5MemoryProbeConfig.intervalMs`, so it must be fast and must not block.
public typealias Socks5MemoryProbe = @convention(c) (UnsafeMutableRawPointer?) -> Int64

/// Lets hev switch admission by itself from the memory the process has left.
/// Thresholds are in KiB of memory left, not of memory used. Unchecked
/// Sendable: the probe is a C function pointer and cannot capture state.
public struct Socks5MemoryProbeConfig: @unchecked Sendable {
    public var probe: Socks5MemoryProbe
    /// The last probe result is reused for this long.
    public var intervalMs: Int
    /// Less than this left: `.hold`.
    public var holdBelowKb: Int
    /// Less than this left: `.closed`. Must not be above `holdBelowKb`.
    public var closeBelowKb: Int
    /// A state is left only this far above its threshold.
    public var hysteresisKb: Int
    /// A session younger than this has not taken its memory yet,
    public var youngMs: Int
    /// so this much is reserved for each one that grew the live count.
    public var youngReserveKb: Int
    /// Session kinds the probe may put in `.hold`.
    public var holdTypes: Socks5SessionTypes
    /// Session kinds the probe may put in `.closed`; the others get `.hold`
    /// if `holdTypes` has them.
    public var closeTypes: Socks5SessionTypes

    public init(
        probe: Socks5MemoryProbe,
        intervalMs: Int = 50,
        holdBelowKb: Int,
        closeBelowKb: Int,
        hysteresisKb: Int,
        youngMs: Int = 1_000,
        youngReserveKb: Int = 0,
        holdTypes: Socks5SessionTypes = [.udp, .tcp],
        closeTypes: Socks5SessionTypes = [.udp, .tcp]
    ) {
        self.probe = probe
        self.intervalMs = intervalMs
        self.holdBelowKb = holdBelowKb
        self.closeBelowKb = closeBelowKb
        self.hysteresisKb = hysteresisKb
        self.youngMs = youngMs
        self.youngReserveKb = youngReserveKb
        self.holdTypes = holdTypes
        self.closeTypes = closeTypes
    }
}

#if os(iOS)
public extension Socks5MemoryProbeConfig {
    /// Memory left before iOS kills this process. `os_proc_available_memory`
    /// returns 0 both past the limit and where it does not apply, so 0 is
    /// reported as unknown and hev keeps its previous state.
    static let processAvailableMemory: Socks5MemoryProbe = { _ in
        let available = os_proc_available_memory()
        return available > 0 ? Int64(available) : -1
    }
}
#endif

/// Session counters of hev. Counters restart with every tunnel start.
public struct Socks5SessionStats: Sendable, Equatable {
    public let udpLive: Int
    public let tcpLive: Int
    /// Closed sessions that have not released their memory yet.
    public let dying: Int
    /// Datagrams waiting in all UDP session queues.
    public let udpQueuedFrames: Int
    public let admissionUdp: Socks5Admission?
    public let admissionTcp: Socks5Admission?
    /// Caps in effect, 0 for none.
    public let udpCap: Int
    public let tcpCap: Int
    /// Last memory probe result, nil before the first probe of this start.
    public let lastProbeKb: Int?
    public let udpCreated: Int
    public let tcpCreated: Int
    /// Silent sessions closed to make room for a new one.
    public let udpEvicted: Int
    public let tcpEvicted: Int
    /// New sessions refused by a cap or by admission.
    public let udpRefused: Int
    public let tcpRefused: Int
    /// Part of the refused ones turned away for lack of memory, not by a cap.
    public let udpRefusedByAdmission: Int
    public let tcpRefusedByAdmission: Int
    /// Sessions closed by `relieve`.
    public let udpRelieved: Int
    public let tcpRelieved: Int
    /// Datagrams dropped because a UDP queue was full.
    public let udpQueueDropped: Int
    /// Time the memory probe has kept `.hold` and `.closed`.
    public let probeHoldMs: Int
    public let probeClosedMs: Int

    init(_ s: HevSocks5TunnelSessionStats) {
        udpLive = Int(s.udp_live)
        tcpLive = Int(s.tcp_live)
        dying = Int(s.dying)
        udpQueuedFrames = Int(s.udp_queued_frames)
        admissionUdp = Socks5Admission(rawValue: s.admission_udp)
        admissionTcp = Socks5Admission(rawValue: s.admission_tcp)
        udpCap = Int(s.udp_cap)
        tcpCap = Int(s.tcp_cap)
        lastProbeKb = s.last_probe_kb < 0 ? nil : Int(s.last_probe_kb)
        udpCreated = Int(clamping: s.udp_created)
        tcpCreated = Int(clamping: s.tcp_created)
        udpEvicted = Int(clamping: s.udp_evicted)
        tcpEvicted = Int(clamping: s.tcp_evicted)
        udpRefused = Int(clamping: s.udp_refused)
        tcpRefused = Int(clamping: s.tcp_refused)
        udpRefusedByAdmission = Int(clamping: s.udp_refused_adm)
        tcpRefusedByAdmission = Int(clamping: s.tcp_refused_adm)
        udpRelieved = Int(clamping: s.udp_relieved)
        tcpRelieved = Int(clamping: s.tcp_relieved)
        udpQueueDropped = Int(clamping: s.udp_queue_dropped)
        probeHoldMs = Int(clamping: s.probe_hold_ms)
        probeClosedMs = Int(clamping: s.probe_closed_ms)
    }
}

/// Caps, eviction and queue limits come from the `misc` block of the hev
/// config on every start. These calls are safe from any thread, whether or
/// not the tunnel is running, so they skip the actor and never wait on `stop()`.
public extension Socks5TunnelProvider {
    /// Closes matching sessions to give memory back, the tunnel keeps running.
    /// `idleMs` 0 closes regardless of activity, `maxCount` 0 means no limit.
    /// Returns false when the tunnel is not running.
    nonisolated func relieve(_ types: Socks5SessionTypes, idleMs: Int = 0, maxCount: Int = 0) -> Bool {
        hev_socks5_tunnel_relieve(types.rawValue, Int32(clamping: idleMs), Int32(clamping: maxCount)) == 0
    }

    /// nil turns the probe off and lets it act on both kinds again. The probe
    /// outlives tunnel restarts. Returns false for thresholds hev rejects.
    nonisolated func setMemoryProbe(_ config: Socks5MemoryProbeConfig?) -> Bool {
        guard let config else {
            hev_socks5_tunnel_set_memory_probe(nil, nil, 0, 0, 0, 0, 0, 0)
            let all = Socks5SessionTypes([.udp, .tcp]).rawValue
            return hev_socks5_tunnel_set_memory_probe_types(all, all) == 0
        }
        let accepted = hev_socks5_tunnel_set_memory_probe(
            config.probe,
            nil,
            Int32(clamping: config.intervalMs),
            Int32(clamping: config.holdBelowKb),
            Int32(clamping: config.closeBelowKb),
            Int32(clamping: config.hysteresisKb),
            Int32(clamping: config.youngMs),
            Int32(clamping: config.youngReserveKb)
        ) == 0
        guard accepted else { return false }
        return hev_socks5_tunnel_set_memory_probe_types(config.holdTypes.rawValue, config.closeTypes.rawValue) == 0
    }

    nonisolated func sessionStats() -> Socks5SessionStats? {
        var stats = HevSocks5TunnelSessionStats()
        stats.size = UInt32(MemoryLayout<HevSocks5TunnelSessionStats>.stride)
        guard hev_socks5_tunnel_get_session_stats(&stats) == 0 else {
            return nil
        }
        return Socks5SessionStats(stats)
    }
}
