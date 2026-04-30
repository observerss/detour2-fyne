import Foundation
import NetworkExtension

final class PacketTunnelProvider: NEPacketTunnelProvider {
    private var runner: Tun2SocksRunner?

    override func startTunnel(options: [String: NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        do {
            let providerConfiguration = (protocolConfiguration as? NETunnelProviderProtocol)?.providerConfiguration ?? [:]
            let config = try DetourTunnelConfig(providerConfiguration: providerConfiguration)
            let settings = makeNetworkSettings(config: config)

            setTunnelNetworkSettings(settings) { [weak self] error in
                guard let self = self else {
                    completionHandler(nil)
                    return
                }
                if let error = error {
                    completionHandler(error)
                    return
                }
                GomobileBackendInstaller.install()
                let runner = Tun2SocksRunner(packetFlow: self.packetFlow, config: config)
                self.runner = runner
                do {
                    try runner.start()
                    completionHandler(nil)
                } catch {
                    completionHandler(error)
                }
            }
        } catch {
            completionHandler(error)
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        runner?.stop()
        runner = nil
        completionHandler()
    }

    private func makeNetworkSettings(config: DetourTunnelConfig) -> NEPacketTunnelNetworkSettings {
        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "10.21.0.1")
        settings.mtu = NSNumber(value: config.mtu)

        let ipv4 = NEIPv4Settings(addresses: ["10.21.0.2"], subnetMasks: ["255.255.255.255"])
        ipv4.includedRoutes = [NEIPv4Route.default()]
        settings.ipv4Settings = ipv4

        let ipv6 = NEIPv6Settings(addresses: ["fd00:10:21::2"], networkPrefixLengths: [NSNumber(value: 128)])
        ipv6.includedRoutes = [NEIPv6Route.default()]
        settings.ipv6Settings = ipv6

        let dns = NEDNSSettings(servers: config.dnsServers)
        dns.matchDomains = [""]
        settings.dnsSettings = dns

        return settings
    }
}