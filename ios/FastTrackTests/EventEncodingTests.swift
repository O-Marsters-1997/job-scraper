import Foundation
import Testing
@testable import FastTrack

struct EventEncodingTests {
    private func json(_ value: some Encodable) throws -> [String: Any] {
        let data = try JSONEncoder().encode(value)
        return try #require(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    @Test func impressionBatchUsesServerKeys() throws {
        let batch = ImpressionBatchInput(surface: "triage", sort: "relevant", items: [.init(jobID: "job-1", position: 2)])
        let object = try json(batch)
        #expect(object["surface"] as? String == "triage")
        #expect(object["sort"] as? String == "relevant")
        let items = try #require(object["items"] as? [[String: Any]])
        #expect(items.count == 1)
        #expect(items[0]["job_id"] as? String == "job-1")
        #expect(items[0]["position"] as? Int == 2)
    }

    @Test func openedEventUsesServerKeys() throws {
        let object = try json(EventInput(type: .jobOpened, subjectID: "job-1", reason: "", undone: false))
        #expect(object["type"] as? String == "job_opened")
        #expect(object["subject_id"] as? String == "job-1")
    }
}
