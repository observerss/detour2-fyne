import Foundation

public enum DetourTunnelConfigError: LocalizedError {
    case missingRemoteAddress
    case missingPassword

    public var errorDescription: String? {
        switch self {
        case .missingRemoteAddress:
            return "Detour2 tunnel config is missing remoteAddress."
        case .missingPassword:
            return "Detour2 tunnel config is missing password."
        }
    }
}

public struct DetourTunnelConfig: Codable, Equatable {
    private static let defaultDNSServers = ["1.1.1.1", "8.8.8.8"]
    private static let defaultMTU = 1500
    private static let defaultLocalSocksPort = 7776
    private static let defaultPoolSize = 64

    public let remoteAddress: String
    public let password: String
    public let dnsServers: [String]
    public let mtu: Int
    public let localSocksPort: Int
    public let poolSize: Int

    public init(
        remoteAddress: String,
        password: String,
        dnsServers: [String] = ["1.1.1.1", "8.8.8.8"],
        mtu: Int = 1500,
        localSocksPort: Int = 7776,
        poolSize: Int = 64
    ) {
        self.remoteAddress = remoteAddress.trimmingCharacters(in: .whitespacesAndNewlines)
        self.password = password
        self.dnsServers = Self.normalizedDNSServers(dnsServers)
        self.mtu = Self.clamped(mtu, defaultValue: Self.defaultMTU, range: 576...9000)
        self.localSocksPort = Self.clamped(localSocksPort, defaultValue: Self.defaultLocalSocksPort, range: 1...65535)
        self.poolSize = max(poolSize, 1)
    }

    public init(providerConfiguration: [String: Any]) throws {
        guard let remoteAddress = providerConfiguration["remoteAddress"] as? String,
              !remoteAddress.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            throw DetourTunnelConfigError.missingRemoteAddress
        }
        guard let password = providerConfiguration["password"] as? String,
              !password.isEmpty else {
            throw DetourTunnelConfigError.missingPassword
        }

        self.init(
            remoteAddress: remoteAddress,
            password: password,
            dnsServers: Self.stringArray(providerConfiguration["dnsServers"]),
            mtu: Self.intValue(providerConfiguration["mtu"], defaultValue: Self.defaultMTU),
            localSocksPort: Self.intValue(providerConfiguration["localSocksPort"], defaultValue: Self.defaultLocalSocksPort),
            poolSize: Self.intValue(providerConfiguration["poolSize"], defaultValue: Self.defaultPoolSize)
        )
    }

    public var providerConfiguration: [String: NSObject] {
        [
            "remoteAddress": remoteAddress as NSString,
            "password": password as NSString,
            "dnsServers": dnsServers as NSArray,
            "mtu": NSNumber(value: mtu),
            "localSocksPort": NSNumber(value: localSocksPort),
            "poolSize": NSNumber(value: poolSize),
        ]
    }

    private static func normalizedDNSServers(_ servers: [String]) -> [String] {
        let normalized = servers
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        return normalized.isEmpty ? defaultDNSServers : normalized
    }

    private static func stringArray(_ value: Any?) -> [String] {
        if let strings = value as? [String] {
            return strings
        }
        if let strings = value as? [NSString] {
            return strings.map { $0 as String }
        }
        if let array = value as? NSArray {
            return array.compactMap { $0 as? String }
        }
        return defaultDNSServers
    }

    private static func intValue(_ value: Any?, defaultValue: Int) -> Int {
        if let number = value as? NSNumber {
            return number.intValue
        }
        if let number = value as? Int {
            return number
        }
        if let string = value as? String, let number = Int(string.trimmingCharacters(in: .whitespacesAndNewlines)) {
            return number
        }
        return defaultValue
    }

    private static func clamped(_ value: Int, defaultValue: Int, range: ClosedRange<Int>) -> Int {
        if !range.contains(value) {
            return defaultValue
        }
        return value
    }
}