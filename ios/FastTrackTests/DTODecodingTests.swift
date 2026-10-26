import Foundation
import Testing
@testable import FastTrack

struct DTODecodingTests {
    private struct Page: Decodable {
        let items: [Job]
    }

    private func fixture(_ name: String) throws -> Data {
        let url = try #require(Bundle(for: BundleToken.self).url(forResource: name, withExtension: "json"))
        return try Data(contentsOf: url)
    }

    @Test func jobPageDecodes() throws {
        let page = try JSONDecoder.api.decode(Page.self, from: fixture("job_page"))

        let job = try #require(page.items.first)
        #expect(job.id == "job-1")
        #expect(job.title == "Senior Go Engineer")
        #expect(job.companySlug == "monzo")
        #expect(job.location == "London")
        #expect(job.url == "https://example.com/jobs/1")
        #expect(job.band == .great)
        #expect(job.score == 92)
        #expect(job.breakdown.map(\.effect) == [.meets, .misses])
        #expect(job.breakdown.map(\.label) == ["Go", "Fintech"])
    }
}

private final class BundleToken {}
