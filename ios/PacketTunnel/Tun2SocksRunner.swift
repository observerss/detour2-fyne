import Foundation
import NetworkExtension

enum Tun2SocksRunnerError: LocalizedError {
    case backendNotLinked

    var errorDescription: String? {
        switch self {
        case .backendNotLinked:
            return "Packet Tunnel target is present, but no tun2socks backend is linked yet."
        }
    }
}

protocol Tun2SocksBackend {
    func start() throws
    func stop()
}

typealias Tun2SocksBackendFactory = (PacketFlowAdapter, DetourTunnelConfig) throws -> Tun2SocksBackend

final class MissingTun2SocksBackend: Tun2SocksBackend {
    func start() throws {
        throw Tun2SocksRunnerError.backendNotLinked
    }

    func stop() {}
}

final class Tun2SocksRunner {
    static var backendFactory: Tun2SocksBackendFactory = { _, _ in MissingTun2SocksBackend() }

    private let config: DetourTunnelConfig
    private let adapter: PacketFlowAdapter
    private var backend: Tun2SocksBackend?

    init(packetFlow: NEPacketTunnelFlow, config: DetourTunnelConfig) {
        self.config = config
        self.adapter = PacketFlowAdapter(packetFlow: packetFlow)
    }

    func start() throws {
        if backend != nil {
            return
        }
        let backend = try Self.backendFactory(adapter, config)
        do {
            try backend.start()
            self.backend = backend
        } catch {
            backend.stop()
            throw error
        }
    }

    func stop() {
        backend?.stop()
        backend = nil
        adapter.close()
    }
}