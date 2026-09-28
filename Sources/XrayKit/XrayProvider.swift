// Created by Denis Mandych on 04.11.2025

import Foundation
import VpnFoundation.VpnFoundationCore

public protocol XrayProviding: Actor {
    nonisolated var isRunning: Bool { get }
    nonisolated var version: String { get }

    func start(config: Data, datDir: String) throws
    func start(config: String, datDir: String) throws
    func stop() throws
    func configureLogHistory(directory: String?, connectionId: String?, maxSegmentBytes: Int, maxSourceBytes: Int, maxSegments: Int, maxPendingBytes: Int) throws
    nonisolated func logHistoryState() -> String?
}

public extension XrayProviding {
    func start(config: Data) throws {
        try start(config: config, datDir: NSTemporaryDirectory())
    }

    func start(config: String) throws {
        try start(config: config, datDir: NSTemporaryDirectory())
    }
}

public enum XrayError: LocalizedError {
    case callFailed(message: String)
    case invalidResponse

    public var errorDescription: String? {
        switch self {
        case .callFailed(let msg): return msg
        case .invalidResponse: return "Invalid response from LibXray"
        }
    }
}

public actor XrayProvider: XrayProviding {
    public init() {}

    public nonisolated var isRunning: Bool {
        LibXrayGetXrayState() != 0
    }

    public nonisolated var version: String {
        String(cString: LibXrayXrayVersion())
    }

    public func start(config: Data, datDir: String) throws {
        guard let jsonString = String(data: config, encoding: .utf8) else {
            throw XrayError.callFailed(message: "Config is not UTF-8 JSON")
        }

        try start(config: jsonString, datDir: datDir)
    }

    public func start(config: String, datDir: String) throws {
        let result = LibXrayRunXrayFromJSON(datDir, config)
        let resultString = result.map { String(cString: $0) }
        try unwrapBase64Response(resultString)
    }

    public func stop() throws {
        let result = LibXrayStopXray()
        let resultString = result.map { String(cString: $0) }
        try unwrapBase64Response(resultString)
    }

    public func configureLogHistory(
        directory: String?, connectionId: String?, maxSegmentBytes: Int,
        maxSourceBytes: Int, maxSegments: Int, maxPendingBytes: Int
    ) throws {
        guard maxSegmentBytes >= 512, maxSourceBytes >= maxSegmentBytes,
              maxSegments >= 1, maxSegments <= maxSourceBytes / maxSegmentBytes,
              maxPendingBytes > 0 else {
            throw XrayError.callFailed(message: "invalid_log_history_policy")
        }
        let payload: String
        if let directory, !directory.isEmpty {
            let object: [String: Any] = [
                "directory": directory, "connectionId": connectionId ?? "",
                "policy": ["maxSegmentBytes": maxSegmentBytes, "maxSourceBytes": maxSourceBytes,
                           "maxSegments": maxSegments, "maxPendingBytes": maxPendingBytes]
            ]
            payload = String(decoding: try JSONSerialization.data(withJSONObject: object), as: UTF8.self)
        } else {
            payload = ""
        }
        guard LibXrayConfigureLogHistory(payload) == 0 else {
            throw XrayError.callFailed(message: "log_history_configuration_failed")
        }
    }

    public nonisolated func logHistoryState() -> String? {
        guard let response = LibXrayGetLogHistoryState() else { return nil }
        defer { free(response) }
        return String(cString: response)
    }

    private func unwrapBase64Response(_ str: String?) throws(XrayError) {
        guard let str = str,
              let data = Data(base64Encoded: str) else {
            throw XrayError.invalidResponse
        }
        guard let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw XrayError.invalidResponse
        }
        if let errMsg = obj["error"] as? String, !errMsg.isEmpty {
            throw XrayError.callFailed(message: errMsg)
        }
    }
}
