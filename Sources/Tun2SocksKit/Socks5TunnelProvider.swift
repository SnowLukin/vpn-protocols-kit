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
    func start(with config: Socks5Config) throws
    func stop()
    func statistics() -> SocksTunStats
}

public actor Socks5TunnelProvider: Socks5TunnelProviding {

    public static let shared = Socks5TunnelProvider()
    private init() {}

    private var workerTask: Task<Int, Never>?

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

    public func start(with config: Socks5Config) throws(Socks5TunnelError) {
        guard let fd = resolveTunnelFileDescriptor() else {
            throw .failedToResolveTunnelFileDescriptor
        }

        workerTask = Task.detached(priority: .userInitiated) { [weak self] in
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
            
            return Int(exitCode)
        }
        
        // TODO: Handle exit code
    }

    public func stop() {
        hev_socks5_tunnel_quit()
        workerTask?.cancel()
        workerTask = nil
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

    private func resolveTunnelFileDescriptor() -> Int32? {
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
