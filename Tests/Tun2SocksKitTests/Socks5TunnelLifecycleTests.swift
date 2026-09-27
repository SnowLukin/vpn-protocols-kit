import Darwin
import XCTest
@testable import Tun2SocksKit

/// Runs the real hev on one end of a socketpair instead of a utun device.
final class Socks5TunnelLifecycleTests: XCTestCase {
    private static let validConfig = """
    tunnel:
      mtu: 1500
    socks5:
      port: 1080
      address: 127.0.0.1
      udp: 'udp'
    misc:
      log-level: error
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
        fds.forEach { close($0) }
        provider = nil
        super.tearDown()
    }

    func testStopWithoutStartDoesNotBlock() async {
        await assertReturns { await Socks5TunnelProvider.shared.stop() }
    }

    func testInvalidConfigReportsExitCodeAndStopReturns() async {
        let exited = expectation(description: "unexpected exit")
        let exitCode = LockedValue<Int32?>(nil)

        try? await provider.start(with: .string(content: "garbage: [")) { code in
            exitCode.set(code)
            exited.fulfill()
        }
        await fulfillment(of: [exited], timeout: 2)

        XCTAssertEqual(exitCode.get(), -1)
        await assertReturns { await self.provider.stop() }
    }

    func testStartAfterUnexpectedExitRunsNewHev() async throws {
        let exited = expectation(description: "unexpected exit")
        try await provider.start(with: .string(content: "garbage: [")) { _ in exited.fulfill() }
        await fulfillment(of: [exited], timeout: 2)

        try await startValid()
        await assertReturns { await self.provider.stop() }
    }

    func testStopEndsRunningHevWithoutUnexpectedExit() async throws {
        let unexpectedExit = try await startValid()

        await assertReturns { await self.provider.stop() }
        XCTAssertFalse(unexpectedExit.get())
    }

    func testRepeatedAndConcurrentStopReturn() async throws {
        try await startValid()

        await assertReturns {
            async let first: Void = self.provider.stop()
            async let second: Void = self.provider.stop()
            _ = await (first, second)
        }
        await assertReturns { await self.provider.stop() }
    }

    func testStartWhileRunningFails() async throws {
        try await startValid()

        do {
            try await provider.start(with: .string(content: Self.validConfig)) { _ in }
            XCTFail("Expected the second start to fail")
        } catch {
            guard case .failedToStartSocks5Tunnel = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }
        await assertReturns { await self.provider.stop() }
    }

    func testStartAfterStopWithoutWorkerRunsHev() async throws {
        await assertReturns { await self.provider.stop() }

        try await startValid()
        await assertReturns { await self.provider.stop() }
    }

    /// Starts hev and gives it time to finish initialization.
    @discardableResult
    private func startValid() async throws -> LockedValue<Bool> {
        let unexpectedExit = LockedValue(false)
        try await provider.start(with: .string(content: Self.validConfig)) { _ in unexpectedExit.set(true) }
        try await Task.sleep(nanoseconds: 300_000_000)
        XCTAssertFalse(unexpectedExit.get())
        return unexpectedExit
    }

    private func assertReturns(
        within timeout: TimeInterval = 3,
        _ operation: @escaping @Sendable () async -> Void
    ) async {
        let returned = expectation(description: "operation returned")
        Task.detached {
            await operation()
            returned.fulfill()
        }
        await fulfillment(of: [returned], timeout: timeout)
    }
}

private final class LockedValue<Value>: @unchecked Sendable {
    private let lock = NSLock()
    private var value: Value

    init(_ value: Value) {
        self.value = value
    }

    func get() -> Value {
        lock.lock()
        defer { lock.unlock() }
        return value
    }

    func set(_ newValue: Value) {
        lock.lock()
        defer { lock.unlock() }
        value = newValue
    }
}
