import Foundation
import Testing
@testable import FastTrack

struct NextUpWidgetTests {
    private let item = NextUpSnapshot.Item(jobID: "j1", title: "Role", companyName: "Co", score: 80, chaseBy: nil, readyToApply: true)

    @Test func focusFollowsTheDeepLinkContract() {
        #expect(NextUp.chase(item, overdueDays: 2).focus == .chase(jobID: "j1"))
        #expect(NextUp.apply(item).focus == .apply(jobID: "j1"))
        #expect(NextUp.comingUp(item).focus == .chase(jobID: "j1"))
        #expect(NextUp.allClear.focus == nil)
    }

    @Test func refreshDatesAreNowThenEachUpcomingMidnight() {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "UTC")!
        let now = Date(timeIntervalSince1970: 1_790_000_000 + 3600)
        let dates = NextUpSnapshot.refreshDates(from: now, calendar: calendar, days: 3)
        #expect(dates.count == 4)
        #expect(dates[0] == now)
        let midnight = calendar.startOfDay(for: now).addingTimeInterval(24 * 3600)
        #expect(dates[1] == midnight)
        #expect(dates[3] == midnight.addingTimeInterval(2 * 24 * 3600))
    }

    @Test func snapshotRoundTripsThroughTheFile() throws {
        let url = FileManager.default.temporaryDirectory.appending(path: "\(UUID().uuidString).json")
        defer { try? FileManager.default.removeItem(at: url) }
        let snapshot = NextUpSnapshot(writtenAt: Date(timeIntervalSince1970: 1_790_000_000), items: [item])
        try snapshot.write(to: url)
        #expect(NextUpSnapshot.read(from: url) == snapshot)
        #expect(NextUpSnapshot.read(from: url.appending(path: "missing")) == nil)
    }
}
