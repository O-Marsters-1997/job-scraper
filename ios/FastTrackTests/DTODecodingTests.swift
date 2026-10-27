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
        #expect(job.companyKind == .agency)
        #expect(job.breakdown.map(\.effect) == [.meets, .misses])
        #expect(job.breakdown.map(\.label) == ["Go", "Fintech"])
    }

    @Test(arguments: ["\"CompanyKind\": \"\"", "\"CompanyKind\": \"recruiter_ish\"", "\"Other\": 1"])
    func unknownOrAbsentCompanyKindDecodesAsUnclassified(replacement: String) throws {
        let original = try #require(String(data: fixture("job_page"), encoding: .utf8))
        try #require(original.contains("\"CompanyKind\": \"agency\""))
        let json = original.replacingOccurrences(of: "\"CompanyKind\": \"agency\"", with: replacement)
        let job = try #require(JSONDecoder.api.decode(Page.self, from: Data(json.utf8)).items.first)
        #expect(job.companyKind == .unclassified)
    }

    @Test(arguments: [(CompanyKind.agency, "Agency"), (.likelyAgency, "Maybe recruiter")])
    func agencyKindsGetABadge(kind: CompanyKind, badge: String) {
        #expect(kind.badge == badge)
        #expect(kind.isAgency == true)
    }

    @Test(arguments: [CompanyKind.unclassified, .direct, .likelyDirect])
    func otherKindsHaveNoBadge(kind: CompanyKind) {
        #expect(kind.badge == nil)
        #expect(kind.isAgency == false)
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
        #expect(due.companyKind == .agency)
        #expect(due.score == 92)
        #expect(due.applicationID == "app-1")
        #expect(due.chaseBy?.dayString == "2026-10-01")
        #expect(due.tasks.map(\.typeName) == ["Apply", "Finish application"])
        #expect(due.tasks.map(\.systemKey) == ["apply", "finish"])
        #expect(due.isAtDesk)
        #expect(due.draft?.state == .ready)
        let bare = try #require(list.jobs.last)
        #expect(bare.companyKind == .unclassified)
        #expect(bare.band == nil)
        #expect(bare.score == nil)
        #expect(bare.chaseBy == nil)
        #expect(bare.draft == nil)
        #expect(!bare.isAtDesk)
    }

    @Test func taskTypesDecode() throws {
        let types = try JSONDecoder.api.decode([TaskType].self, from: fixture("task_types"))

        #expect(types.map(\.name) == ["Finish application", "Reach out"])
        #expect(types.first?.systemKey == "finish")
        #expect(types.last?.systemKey == "")
    }

    @Test func recruiterContactDecodes() throws {
        let contact = try JSONDecoder.api.decode(RecruiterContact.self, from: fixture("recruiter_contact"))

        #expect(contact.id == "contact-1")
        #expect(contact.channel == .email)
        #expect(contact.outcome == .noReply)
        #expect(contact.contactedOn.dayString == "2026-10-01")
    }

    @Test func cvTemplatesDecode() throws {
        let cvs = try JSONDecoder.api.decode([CVTemplate].self, from: fixture("cv_templates"))

        let cv = try #require(cvs.first)
        #expect(cv.docID == "doc-1")
        #expect(cv.tabID == "tab-1")
        #expect(cv.title == "Backend")
        #expect(cv.isDefault)
    }

    @Test func draftInputEncodesTheTailoringShape() throws {
        let data = try JSONEncoder().encode(DraftInput(jobID: "j", docID: "d", tabID: "t"))
        let json = try #require(JSONSerialization.jsonObject(with: data) as? [String: String])

        #expect(json == ["jobId": "j", "docId": "d", "tabId": "t", "mode": "generate"])
    }
}

private final class BundleToken {}
