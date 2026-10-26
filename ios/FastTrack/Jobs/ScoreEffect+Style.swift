import SwiftUI

extension ScoreEffect {
    var isGate: Bool { self == .gated || self == .blocked }

    var symbol: String {
        switch self {
        case .gated, .blocked: "xmark.octagon.fill"
        case .meets: "checkmark.circle"
        case .misses: "xmark.circle"
        case .favourite: "star.circle"
        case .level: "equal.circle"
        case .unknown: "questionmark.circle"
        case .neutral, .retired: "minus.circle"
        }
    }

    var colour: Color {
        switch self {
        case .gated, .blocked, .misses: Color(hex: 0xdc2626)
        case .meets, .favourite: Color(hex: 0x059669)
        case .level, .unknown, .neutral, .retired: .secondary
        }
    }

    var groupTitle: String {
        switch self {
        case .gated, .blocked: "Gates"
        case .favourite: "Boosts"
        case .meets: "Met"
        case .misses: "Missed"
        case .level: "Level"
        case .unknown: "Unknown"
        case .neutral, .retired: "Other"
        }
    }
}
