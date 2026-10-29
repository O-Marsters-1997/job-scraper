import Foundation
import Testing
@testable import FastTrack

struct NextUpTests {
    private let calendar = Calendar.current
    private let today = Calendar.current.startOfDay(for: Date(timeIntervalSince1970: 1_790_000_000))

    private func day(_ offset: Int, hour: Int = 0) -> Date {
        calendar.date(byAdding: .hour, value: hour, to: calendar.date(byAdding: .day, value: offset, to: today)!)!
    }

    private func item(_ id: String, chase: Int? = nil, ready: Bool = false, score: Int? = 80) -> NextUpSnapshot.Item {
        .init(jobID: id, title: "Role \(id)", companyName: "Co", score: score, chaseBy: chase.map { day($0) }, readyToApply: ready)
    }

    private func pick(_ items: [NextUpSnapshot.Item], at: Date? = nil) -> NextUp {
        NextUp(NextUpSnapshot(writtenAt: today, items: items), at: at ?? day(0, hour: 1), calendar: calendar)
    }

    @Test func mostOverdueChaseWins() {
        let items = [item("a", chase: -1), item("b", chase: -3), item("c", ready: true, score: 99)]
        #expect(pick(items) == .chase(items[1], overdueDays: 3))
    }

    @Test func chaseDueTodayBeatsReadyToApply() {
        let items = [item("a", ready: true, score: 99), item("b", chase: 0)]
        #expect(pick(items) == .chase(items[1], overdueDays: 0))
    }

    @Test func highestScoringReadyToApplyWhenNoChaseDue() {
        let items = [item("a", ready: true, score: 70), item("b", ready: true, score: 90), item("c", chase: 2)]
        #expect(pick(items) == .apply(items[1]))
    }

    @Test func earliestFutureChaseWhenNothingElse() {
        let items = [item("a", chase: 5), item("b", chase: 2)]
        #expect(pick(items) == .comingUp(items[1]))
    }

    @Test func allClearWhenEmpty() {
        #expect(pick([]) == .allClear)
    }

    @Test func chaseBecomesDueWhenTimeCrossesMidnight() {
        let items = [item("a", chase: 1)]
        #expect(pick(items, at: day(0, hour: 23)) == .comingUp(items[0]))
        #expect(pick(items, at: day(1, hour: 1)) == .chase(items[0], overdueDays: 0))
    }

    @Test func isStaleOnlyPastTwentyFourHours() {
        let snapshot = NextUpSnapshot(writtenAt: today, items: [])
        #expect(snapshot.isStale(at: day(0, hour: 24)) == false)
        #expect(snapshot.isStale(at: day(0, hour: 25)))
    }
}
