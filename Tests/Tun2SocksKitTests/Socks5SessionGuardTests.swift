import Darwin
import VpnFoundation.HevSocks5Tunnel
import XCTest
@testable import Tun2SocksKit

/// Runs the real hev on one end of a socketpair instead of a utun device.
final class Socks5SessionGuardTests: XCTestCase {
    private static let guardedConfig = """
    tunnel:
      mtu: 1500
    socks5:
      port: 1080
      address: 127.0.0.1
      udp: 'udp'
    misc:
      log-level: error
      udp-max-sessions: 64
      tcp-max-sessions: 80
    """

    private var fds: [Int32] = [-1, -1]
    private var provider: Socks5TunnelProvider!

    override func setUp() {
        super.setUp()
        XCTAssertEqual(socketpair(AF_UNIX, SOCK_DGRAM, 0, &fds), 0)
        let fd = fds[0]
        provider = Socks5TunnelProvider(resolveTunnelFileDescriptor: { fd })
    }

    override func tearDown() {
        _ = provider.setMemoryProbe(nil)
        fds.forEach { close($0) }
        provider = nil
        super.tearDown()
    }

    func testRelieveWithoutRunningTunnelFails() {
        XCTAssertFalse(provider.relieve(.udp, idleMs: 0, maxCount: 0))
    }

    func testRunningTunnelReportsCapsFromConfigAndRelieves() async throws {
        try await provider.start(with: .string(content: Self.guardedConfig)) { _ in }
        try await Task.sleep(nanoseconds: 300_000_000)

        let stats = try XCTUnwrap(provider.sessionStats())
        XCTAssertEqual(stats.udpCap, 64)
        XCTAssertEqual(stats.tcpCap, 80)
        XCTAssertEqual(stats.admissionUdp, .open)
        XCTAssertEqual(stats.admissionTcp, .open)
        XCTAssertNil(stats.lastProbeKb)
        XCTAssertTrue(provider.relieve([.udp, .tcp], idleMs: 0, maxCount: 0))

        await provider.stop()
        XCTAssertFalse(provider.relieve(.udp, idleMs: 0, maxCount: 0))
        XCTAssertNil(provider.sessionStats())
    }

    func testMemoryProbeRejectsSwappedThresholds() {
        let swapped = Socks5MemoryProbeConfig(
            probe: { _ in 0 },
            holdBelowKb: 8 * 1024,
            closeBelowKb: 12 * 1024,
            hysteresisKb: 0
        )
        XCTAssertFalse(provider.setMemoryProbe(swapped))

        let valid = Socks5MemoryProbeConfig(
            probe: { _ in 0 },
            holdBelowKb: 12 * 1024,
            closeBelowKb: 8 * 1024,
            hysteresisKb: 2 * 1024
        )
        XCTAssertTrue(provider.setMemoryProbe(valid))
        XCTAssertTrue(provider.setMemoryProbe(nil))
    }

    func testStatsMapEveryField() {
        var raw = HevSocks5TunnelSessionStats()
        raw.udp_live = 1
        raw.tcp_live = 2
        raw.dying = 3
        raw.udp_queued_frames = 4
        raw.admission_udp = 1
        raw.admission_tcp = 2
        raw.udp_cap = 5
        raw.tcp_cap = 6
        raw.last_probe_kb = 7
        raw.udp_created = 8
        raw.tcp_created = 9
        raw.udp_evicted = 10
        raw.tcp_evicted = 11
        raw.udp_refused = 12
        raw.tcp_refused = 13
        raw.udp_relieved = 14
        raw.tcp_relieved = 15
        raw.udp_refused_adm = 16
        raw.tcp_refused_adm = 17
        raw.udp_queue_dropped = 18
        raw.probe_hold_ms = 19
        raw.probe_closed_ms = 20

        let stats = Socks5SessionStats(raw)

        XCTAssertEqual(
            [
                stats.udpLive, stats.tcpLive, stats.dying, stats.udpQueuedFrames,
                stats.udpCap, stats.tcpCap, stats.lastProbeKb ?? -1,
                stats.udpCreated, stats.tcpCreated, stats.udpEvicted, stats.tcpEvicted,
                stats.udpRefused, stats.tcpRefused, stats.udpRelieved, stats.tcpRelieved,
                stats.udpRefusedByAdmission, stats.tcpRefusedByAdmission, stats.udpQueueDropped,
                stats.probeHoldMs, stats.probeClosedMs,
            ],
            [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
        )
        XCTAssertEqual(stats.admissionUdp, .hold)
        XCTAssertEqual(stats.admissionTcp, .closed)
    }

    func testUnknownProbeResultIsNil() {
        var raw = HevSocks5TunnelSessionStats()
        raw.last_probe_kb = -1
        XCTAssertNil(Socks5SessionStats(raw).lastProbeKb)
    }
}
