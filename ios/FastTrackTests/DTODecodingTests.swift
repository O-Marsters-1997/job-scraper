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

    @Test func companyExclusionDecodes() throws {
        let exclusion = try JSONDecoder.api.decode(CompanyExclusion.self, from: fixture("company_exclusion"))

        #expect(exclusion.name == "Monzo")
        #expect(exclusion.added == true)
    }

    @Test func jobNoteDecodes() throws {
        let note = try JSONDecoder.api.decode(JobNote.self, from: fixture("job_note"))

        #expect(note.jobID == "job-1")
        #expect(note.body == "call back Friday")
    }

    @Test func dueListDecodes() throws {
        let list = try JSONDecoder.api.decode(DueList.self, from: fixture("due_list"))

        let due = try #require(list.jobs.first)
        #expect(due.id == "job-1")
        #expect(due.band == .great)
        #expect(due.score == 92)
        #expect(due.applicationID == "app-1")
        #expect(due.chaseBy?.dayString == "2026-10-01")
        #expect(due.tasks.map(\.typeName) == ["Apply"])
        #expect(due.draft?.state == .ready)
        let bare = try #require(list.jobs.last)
        #expect(bare.band == nil)
        #expect(bare.score == nil)
        #expect(bare.chaseBy == nil)
        #expect(bare.draft == nil)
    }

    @Test func recruiterContactDecodes() throws {
        let contact = try JSONDecoder.api.decode(RecruiterContact.self, from: fixture("recruiter_contact"))

        #expect(contact.id == "contact-1")
        #expect(contact.channel == .email)
        #expect(contact.outcome == .noReply)
        #expect(contact.contactedOn.dayString == "2026-10-01")
    }
}

private final class BundleToken {}
