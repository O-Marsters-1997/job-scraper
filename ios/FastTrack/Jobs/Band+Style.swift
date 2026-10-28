import SwiftUI

extension Band {
    var title: String { rawValue.capitalized }

    var colour: Color {
        switch self {
        case .great: Color(hex: 0x059669)
        case .good: Color(hex: 0x10b981)
        case .fair: Color(hex: 0xd97706)
        case .poor: Color(hex: 0xdc2626)
        }
    }
}
