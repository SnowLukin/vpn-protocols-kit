// Created by Denis Mandych on 04.11.2025

import Foundation
import VpnFoundation

public protocol XrayProviding: Actor {
    nonisolated var isRunning: Bool { get }
    nonisolated var version: String { get }

    func start(config: Data, datDir: String) throws
    func start(config: String, datDir: String) throws
    func stop() throws
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
        guard let request = LibXrayRunFromJSONRequest(datDir, config) else {
            throw XrayError.callFailed(message: "Failed to create request")
        }
        let requestString = String(cString: request)
        try unwrapBase64Response(requestString)

        guard let result = LibXrayRunFromJSON(requestString) else {
            throw XrayError.callFailed(message: "Failed to run Xray")
        }
        let resultString = String(cString: result)
        try unwrapBase64Response(resultString)
    }

    public func stop() throws {
        let result = LibXrayStopXray()
        let resultString = result.map { String(cString: $0) }
        try unwrapBase64Response(resultString)
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
