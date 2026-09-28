import Foundation
import VpnFoundation.HevSocks5Tunnel
import WireGuardKitC

public enum Socks5Config {
    case file(path: URL)
    case string(content: String)
}

public enum Socks5TunnelError: LocalizedError {
    case failedToResolveTunnelFileDescriptor
    case failedToStartSocks5Tunnel
    case invalidLogHistoryPolicy
    case failedToConfigureLogHistory
}

public protocol Socks5TunnelProviding: Actor {
    func configureLogHistory(
        directory: String?,
        connectionId: String?,
        maxSegmentBytes: Int,
        maxSourceBytes: Int,
        maxSegments: Int
    ) throws
    nonisolated func logHistoryState() -> String?
    /// `onUnexpectedExit` receives the hev exit code when hev stops without a
    /// `stop()` request, e.g. a negative code when its initialization fails.
    func start(
        with config: Socks5Config,
        onUnexpectedExit: @escaping @Sendable (Int32) -> Void
    ) throws
    func stop() async
    func statistics() -> SocksTunStats
}

public actor Socks5TunnelProvider: Socks5TunnelProviding {

    public static let shared = Socks5TunnelProvider(
        resolveTunnelFileDescriptor: { Socks5TunnelProvider.findTunnelFileDescriptor() }
    )

    private let resolveTunnelFileDescriptor: @Sendable () -> Int32?
    private var worker: Worker?

    init(resolveTunnelFileDescriptor: @escaping @Sendable () -> Int32?) {
        self.resolveTunnelFileDescriptor = resolveTunnelFileDescriptor
    }

    public func configureLogHistory(
        directory: String?,
        connectionId: String?,
        maxSegmentBytes: Int,
        maxSourceBytes: Int,
        maxSegments: Int
    ) throws(Socks5TunnelError) {
        guard maxSegmentBytes >= 512,
              maxSourceBytes >= maxSegmentBytes,
              maxSegments >= 1,
              maxSegments <= maxSourceBytes / maxSegmentBytes,
              maxSegments <= Int(UInt32.max)
        else {
            throw .invalidLogHistoryPolicy
        }

        var policy = HevSocks5LogHistoryPolicy()
        policy.max_segment_bytes = numericCast(maxSegmentBytes)
        policy.max_source_bytes = numericCast(maxSourceBytes)
        policy.max_segments = numericCast(maxSegments)

        let normalizedDirectory = directory?.isEmpty == true ? nil : directory
        let normalizedConnectionId = connectionId?.isEmpty == true ? nil : connectionId
        let result = Self.withNullableCString(normalizedDirectory) { directory in
            Self.withNullableCString(normalizedConnectionId) { connectionId in
                hev_socks5_tunnel_log_history_configure(directory, connectionId, &policy)
            }
        }
        guard result == 0 else {
            throw .failedToConfigureLogHistory
        }
    }

    public nonisolated func logHistoryState() -> String? {
        var buffer = [CChar](repeating: 0, count: 4096)
        let result = buffer.withUnsafeMutableBufferPointer { buffer in
            hev_socks5_tunnel_log_history_state(buffer.baseAddress, buffer.count)
        }
        guard result == 0 else {
            return nil
        }
        return buffer.withUnsafeBufferPointer { buffer in
            guard let baseAddress = buffer.baseAddress else {
                return nil
            }
            return String(validatingUTF8: baseAddress)
        }
    }

    public func start(
        with config: Socks5Config,
        onUnexpectedExit: @escaping @Sendable (Int32) -> Void
    ) throws(Socks5TunnelError) {
        if let worker {
            // hev state is process-wide: a second hev must not start while
            // the previous one is still running or being stopped.
            guard worker.exit.hasExited else {
                throw .failedToStartSocks5Tunnel
            }
            self.worker = nil
        }

        guard let fd = resolveTunnelFileDescriptor() else {
            throw .failedToResolveTunnelFileDescriptor
        }

        let exit = WorkerExit()
        let task = Task.detached(priority: .userInitiated) {
            let exitCode: Int32
            switch config {
            case .file(let url):
                exitCode = url.path.withCString { ptr in
                    hev_socks5_tunnel_main(ptr, fd)
                }
            case .string(let content):
                let utf8Bytes = [UInt8](content.utf8)
                exitCode = utf8Bytes.withUnsafeBufferPointer { buffer in
                    hev_socks5_tunnel_main_from_str(buffer.baseAddress, UInt32(buffer.count), fd)
                }
            }
            if exit.markExited() {
                onUnexpectedExit(exitCode)
            }
        }
        worker = Worker(task: task, exit: exit)
    }

    public func stop() async {
        guard let worker else { return }

        // hev_socks5_tunnel_quit() blocks until a running hev can receive it.
        // A hev that has not started or has already exited never can, so only
        // the first stop of a live worker sends it.
        if worker.exit.requestQuit() {
            hev_socks5_tunnel_quit()
        }
        await worker.task.value

        // Another start may have replaced the worker while this stop waited.
        if self.worker?.exit === worker.exit {
            self.worker = nil
        }
    }

    public func statistics() -> SocksTunStats {
        var tPackets: Int = 0
        var tBytes: Int = 0
        var rPackets: Int = 0
        var rBytes: Int = 0
        hev_socks5_tunnel_stats(&tPackets, &tBytes, &rPackets, &rBytes)
        return SocksTunStats(
            up: SocksTunStats.Stat(packets: tPackets, bytes: tBytes),
            down: SocksTunStats.Stat(packets: rPackets, bytes: rBytes)
        )
    }

    private nonisolated static func withNullableCString<Result>(
        _ value: String?,
        body: (UnsafePointer<CChar>?) -> Result
    ) -> Result {
        guard let value else {
            return body(nil)
        }
        return value.withCString(body)
    }

    private static func findTunnelFileDescriptor() -> Int32? {
        var ctlInfo = ctl_info()
        withUnsafeMutablePointer(to: &ctlInfo.ctl_name) {
            $0.withMemoryRebound(to: CChar.self, capacity: MemoryLayout.size(ofValue: $0.pointee)) {
                _ = strcpy($0, "com.apple.net.utun_control")
            }
        }
        for fd: Int32 in 0...1024 {
            var addr = sockaddr_ctl()
            var ret: Int32 = -1
            var len = socklen_t(MemoryLayout.size(ofValue: addr))
            withUnsafeMutablePointer(to: &addr) {
                $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                    ret = getpeername(fd, $0, &len)
                }
            }
            if ret != 0 || addr.sc_family != AF_SYSTEM {
                continue
            }
            if ctlInfo.ctl_id == 0 {
                ret = ioctl(fd, CTLIOCGINFO, &ctlInfo)
                if ret != 0 {
                    continue
                }
            }
            if addr.sc_id == ctlInfo.ctl_id {
                return fd
            }
        }
        return nil
    }
}

private struct Worker {
    let task: Task<Void, Never>
    let exit: WorkerExit
}

/// Written from the hev thread when hev returns, so a stop() never waits for
/// an actor hop to learn that hev is gone.
private final class WorkerExit: @unchecked Sendable {
    private let lock = NSLock()
    private var exited = false
    private var quitRequested = false

    var hasExited: Bool {
        lock.lock()
        defer { lock.unlock() }
        return exited
    }

    /// Returns `true` only for the first request while hev is still running.
    func requestQuit() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        guard !exited, !quitRequested else { return false }
        quitRequested = true
        return true
    }

    /// Returns `true` when hev exited without a quit request.
    func markExited() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        exited = true
        return !quitRequested
    }
}
