import Foundation

#if canImport(DetourTunnelBackend)
import DetourTunnelBackend

extension PacketFlowAdapter: IospacketPacketFlow {}

final class GomobileTun2SocksBackend: Tun2SocksBackend {
    private let adapter: PacketFlowAdapter
    private let backend: IospacketBackend

    init(adapter: PacketFlowAdapter, config: DetourTunnelConfig) throws {
        self.adapter = adapter
        self.backend = try IospacketNewBackend(
            adapter,
            config.remoteAddress,
            config.password,
            config.mtu,
            config.localSocksPort,
            config.poolSize
        )
    }

    func start() throws {
        try backend.start()
    }

    func stop() {
        backend.stop()
        adapter.close()
    }
}

enum GomobileBackendInstaller {
    static func install() {
        Tun2SocksRunner.backendFactory = { adapter, config in
            try GomobileTun2SocksBackend(adapter: adapter, config: config)
        }
    }
}
#else
enum GomobileBackendInstaller {
    static func install() {}
}
#endif