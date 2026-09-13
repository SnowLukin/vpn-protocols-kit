import XCTest
@testable import Tun2SocksKit

final class Socks5TunnelProviderTests: XCTestCase {

    func testRejectsPolicyBelowMinimumSegmentSize() async {
        do {
            try await Socks5TunnelProvider.shared.configureLogHistory(
                directory: "/tmp/logs",
                connectionId: nil,
                maxSegmentBytes: 511,
                maxSourceBytes: 2_048,
                maxSegments: 4
            )
            XCTFail("Expected invalid log history policy")
        } catch let error as Socks5TunnelError {
            guard case .invalidLogHistoryPolicy = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }
}
