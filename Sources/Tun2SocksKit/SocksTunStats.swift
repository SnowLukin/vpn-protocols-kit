import Foundation

public struct SocksTunStats: Sendable {
    public struct Stat: Sendable {
        public let packets: Int
        public let bytes: Int
        
        public init(packets: Int, bytes: Int) {
            self.packets = packets
            self.bytes = bytes
        }
    }

    public let up: Stat
    public let down: Stat
    
    public init(up: Stat, down: Stat) {
        self.up = up
        self.down = down
    }
}
