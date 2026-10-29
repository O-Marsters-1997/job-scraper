import Foundation

enum NextUp: Equatable {
    case chase(NextUpSnapshot.Item, overdueDays: Int)
    case apply(NextUpSnapshot.Item)
    case comingUp(NextUpSnapshot.Item)
    case allClear

    init(_ snapshot: NextUpSnapshot, at: Date, calendar: Calendar) {
        let chases = snapshot.items.compactMap { item in item.chaseBy.map { (item, $0) } }.sorted { $0.1 < $1.1 }
        if let (item, chaseBy) = chases.first(where: { Self.isChaseDue($0.1, at: at, calendar: calendar) }) {
            let days = calendar.dateComponents([.day], from: calendar.startOfDay(for: chaseBy), to: calendar.startOfDay(for: at)).day ?? 0
            self = .chase(item, overdueDays: max(days, 0))
        } else if let item = snapshot.items.filter(\.readyToApply).max(by: { ($0.score ?? .min) < ($1.score ?? .min) }) {
            self = .apply(item)
        } else if let (item, _) = chases.first {
            self = .comingUp(item)
        } else {
            self = .allClear
        }
    }

    static func isChaseDue(_ chaseBy: Date, at: Date, calendar: Calendar) -> Bool {
        guard let tomorrow = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: at)) else { return false }
        return chaseBy < tomorrow
    }
}
