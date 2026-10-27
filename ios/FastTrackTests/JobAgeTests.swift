import Foundation
import Testing
@testable import FastTrack

struct JobAgeTests {
    private let now = Date(timeIntervalSince1970: 1_000_000_000)

    @Test(arguments: [
        (0, "now", "just posted"),
        (59, "now", "just posted"),
        (60, "1m", "1 minute old"),
        (1_800, "30m", "30 minutes old"),
        (3_599, "59m", "59 minutes old"),
        (3_600, "1h", "1 hour old"),
        (18_000, "5h", "5 hours old"),
        (86_399, "23h", "23 hours old"),
        (86_400, "1d", "1 day old"),
        (259_200, "3d", "3 days old"),
        (604_799, "6d", "6 days old"),
        (604_800, "1w", "1 week old"),
        (1_209_600, "2w", "2 weeks old"),
    ])
    func readsAtEachBoundary(seconds: Int, label: String, spoken: String) {
        let age = JobAge(since: now.addingTimeInterval(-Double(seconds)), now: now)
        #expect(age.label == label)
        #expect(age.spoken == spoken)
    }

    @Test func futureDatesReadAsNow() {
        #expect(JobAge(since: now.addingTimeInterval(600), now: now).label == "now")
    }
}
