import Darwin
import Foundation
import NetworkExtension

enum PacketFlowAdapterError: LocalizedError {
    case closed
    case emptyPacket
    case unsupportedPacketVersion(UInt8)

    var errorDescription: String? {
        switch self {
        case .closed:
            return "Packet flow is closed."
        case .emptyPacket:
            return "Packet flow returned an empty packet."
        case .unsupportedPacketVersion(let version):
            return "Unsupported IP packet version: \(version)."
        }
    }
}

final class PacketFlowAdapter: NSObject {
    private let packetFlow: NEPacketTunnelFlow
    private let lock = NSLock()
    private let semaphore = DispatchSemaphore(value: 0)
    private var packets: [Data] = []
    private var closed = false

    init(packetFlow: NEPacketTunnelFlow) {
        self.packetFlow = packetFlow
        super.init()
        readMorePackets()
    }

    @objc
    func readPacket() throws -> Data {
        semaphore.wait()
        lock.lock()
        defer { lock.unlock() }
        if closed {
            throw PacketFlowAdapterError.closed
        }
        if packets.isEmpty {
            throw PacketFlowAdapterError.emptyPacket
        }
        return packets.removeFirst()
    }

    @objc
    func writePacket(_ packet: Data) throws {
        let protocolFamily = try packetProtocolFamily(packet)
        packetFlow.writePackets([packet], withProtocols: [NSNumber(value: protocolFamily)])
    }

    @objc
    func close() {
        lock.lock()
        closed = true
        packets.removeAll()
        lock.unlock()
        semaphore.signal()
    }

    private func readMorePackets() {
        packetFlow.readPackets { [weak self] packets, _ in
            guard let self = self else { return }
            var shouldContinue = true
            self.lock.lock()
            if self.closed {
                shouldContinue = false
            } else {
                self.packets.append(contentsOf: packets)
            }
            self.lock.unlock()

            for _ in 0..<packets.count {
                self.semaphore.signal()
            }
            if shouldContinue {
                self.readMorePackets()
            }
        }
    }

    private func packetProtocolFamily(_ packet: Data) throws -> Int32 {
        guard let firstByte = packet.first else {
            throw PacketFlowAdapterError.emptyPacket
        }
        let version = firstByte >> 4
        switch version {
        case 4:
            return AF_INET
        case 6:
            return AF_INET6
        default:
            throw PacketFlowAdapterError.unsupportedPacketVersion(version)
        }
    }
}