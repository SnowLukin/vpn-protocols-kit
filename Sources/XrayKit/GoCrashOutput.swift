import Foundation
import VpnFoundation.VpnFoundationCore

/// Fatal report of the Go runtime shared by Xray and AmneziaWG.
public enum GoCrashOutput {
    /// Truncates the file at `path` and makes the Go runtime copy its next
    /// fatal report there. Move a previous report away before calling.
    public static func redirect(toPath path: String) throws(XrayError) {
        guard let error = LibXraySetCrashOutput(path) else { return }
        defer { free(error) }
        throw XrayError.callFailed(message: String(cString: error))
    }
}
