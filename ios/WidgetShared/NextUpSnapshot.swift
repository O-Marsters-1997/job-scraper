import Foundation

struct NextUpSnapshot: Codable, Equatable {
    struct Item: Codable, Equatable {
        let jobID: String
        let title: String
        let companyName: String
        let score: Int?
        let chaseBy: Date?
        let readyToApply: Bool
    }

    let writtenAt: Date
    let items: [Item]

    static let appGroup = "group.com.ollymarsters.fasttrack"
    static let fileName = "next-up.json"
    static let widgetKind = "NextUp"
}

extension NextUpSnapshot {
    func isStale(at: Date) -> Bool {
        at.timeIntervalSince(writtenAt) > 24 * 60 * 60
    }

    static var sharedFileURL: URL? {
        FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: appGroup)?.appending(path: fileName)
    }

    static func read(from url: URL) -> NextUpSnapshot? {
        try? JSONDecoder().decode(NextUpSnapshot.self, from: Data(contentsOf: url))
    }

    func write(to url: URL) throws {
        try JSONEncoder().encode(self).write(to: url, options: .atomic)
    }

    static func refreshDates(from at: Date, calendar: Calendar, days: Int = 7) -> [Date] {
        let midnights = (1...days).compactMap { calendar.date(byAdding: .day, value: $0, to: calendar.startOfDay(for: at)) }
        return [at] + midnights
    }
}
