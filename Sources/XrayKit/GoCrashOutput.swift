import Foundation
import VpnFoundation.VpnFoundationCore

/// Fatal report of the Go runtime shared by Xray and AmneziaWG.
public enum GoCrashOutput {
    /// Makes the Go runtime copy its next fatal report to the file at `path`,
    /// truncating whatever the file held. The target is one per process:
    /// a later call replaces it for both Xray and AmneziaWG.
    public static func redirect(toPath path: String) throws(XrayError) {
        guard let error = LibXraySetCrashOutput(path) else { return }
        defer { free(error) }
        throw XrayError.callFailed(message: String(cString: error))
    }
}
