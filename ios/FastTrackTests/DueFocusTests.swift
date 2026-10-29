import Foundation
import Testing
@testable import FastTrack

struct DueFocusTests {
    @Test(arguments: [
        (DueFocus.chase(jobID: "j1"), "fasttrack://due/j1"),
        (DueFocus.apply(jobID: "j2"), "fasttrack://apply/j2"),
    ])
    func urlRoundTrips(focus: DueFocus, expected: String) {
        #expect(focus.url.absoluteString == expected)
        #expect(DueFocus(url: focus.url) == focus)
    }

    @Test(arguments: ["fasttrack://jobs/j1", "fasttrack://due", "fasttrack://due/a/b", "https://due/j1"])
    func rejectsOtherURLs(string: String) {
        #expect(DueFocus(url: URL(string: string)!) == nil)
    }
}
