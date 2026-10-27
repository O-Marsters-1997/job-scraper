import Foundation

struct JobAge: Equatable {
    private let seconds: Int

    init(since date: Date, now: Date = .now) {
        seconds = max(Int(now.timeIntervalSince(date)), 0)
    }

    private var unit: (count: Int, short: String, long: String)? {
        switch seconds {
        case ..<60: nil
        case ..<3_600: (seconds / 60, "m", "minute")
        case ..<86_400: (seconds / 3_600, "h", "hour")
        case ..<604_800: (seconds / 86_400, "d", "day")
        default: (seconds / 604_800, "w", "week")
        }
    }

    var label: String { unit.map { "\($0.count)\($0.short)" } ?? "now" }

    var spoken: String {
        guard let unit else { return "just posted" }
        return "\(unit.count) \(unit.long)\(unit.count == 1 ? "" : "s") old"
    }
}
