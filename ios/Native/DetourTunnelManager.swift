import Foundation
import NetworkExtension

public enum DetourTunnelManagerError: LocalizedError {
    case tunnelProtocolUnavailable

    public var errorDescription: String? {
        switch self {
        case .tunnelProtocolUnavailable:
            return "NETunnelProviderProtocol is unavailable after saving preferences."
        }
    }
}

public final class DetourTunnelManager {
    public let providerBundleIdentifier: String
    public let localizedDescription: String

    public init(
        providerBundleIdentifier: String = "github.com.observerss.detour2fyne.PacketTunnel",
        localizedDescription: String = "Detour2"
    ) {
        self.providerBundleIdentifier = providerBundleIdentifier
        self.localizedDescription = localizedDescription
    }

    public func start(config: DetourTunnelConfig, completion: @escaping (Error?) -> Void) {
        save(config: config) { result in
            switch result {
            case .failure(let error):
                completion(error)
            case .success(let manager):
                do {
                    try manager.connection.startVPNTunnel()
                    completion(nil)
                } catch {
                    completion(error)
                }
            }
        }
    }

    public func stop(completion: @escaping (Error?) -> Void) {
        loadManager { result in
            switch result {
            case .failure(let error):
                completion(error)
            case .success(let manager):
                manager.connection.stopVPNTunnel()
                completion(nil)
            }
        }
    }

    public func save(config: DetourTunnelConfig, completion: @escaping (Result<NETunnelProviderManager, Error>) -> Void) {
        loadManager { [providerBundleIdentifier, localizedDescription] result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let manager):
                let tunnelProtocol = (manager.protocolConfiguration as? NETunnelProviderProtocol) ?? NETunnelProviderProtocol()
                tunnelProtocol.providerBundleIdentifier = providerBundleIdentifier
                tunnelProtocol.providerConfiguration = config.providerConfiguration
                tunnelProtocol.serverAddress = localizedDescription

                manager.localizedDescription = localizedDescription
                manager.protocolConfiguration = tunnelProtocol
                manager.isEnabled = true
                manager.saveToPreferences { error in
                    if let error = error {
                        completion(.failure(error))
                        return
                    }
                    manager.loadFromPreferences { error in
                        if let error = error {
                            completion(.failure(error))
                        } else if manager.protocolConfiguration is NETunnelProviderProtocol {
                            completion(.success(manager))
                        } else {
                            completion(.failure(DetourTunnelManagerError.tunnelProtocolUnavailable))
                        }
                    }
                }
            }
        }
    }

    private func loadManager(completion: @escaping (Result<NETunnelProviderManager, Error>) -> Void) {
        NETunnelProviderManager.loadAllFromPreferences { [localizedDescription] managers, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            let manager = managers?.first { $0.localizedDescription == localizedDescription } ?? NETunnelProviderManager()
            completion(.success(manager))
        }
    }
}