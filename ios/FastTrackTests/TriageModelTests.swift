import Foundation
import Testing
@testable import FastTrack

struct TriageModelTests {
    private final class Calls {
        var log: [String] = []
        var failShortlist = false
        var noDefaultCV = false
    }

    private let calls = Calls()

    private func model() async -> TriageModel {
        var client = APIClient.mock(jobs: [Job.sample.with(id: "a"), Job.sample.with(id: "b")])
        let calls = calls
        client.shortlist = { id in
            if calls.failShortlist { throw APIError.status(500) }
            calls.log.append("shortlist \(id)")
            return Application(id: "app-\(id)", jobID: id)
        }
        client.deleteApplication = { calls.log.append("deleteApplication \($0)") }
        client.dismiss = { id, reasons in calls.log.append("dismiss \(id) \(reasons.map(\.rawValue))") }
        client.clearGrade = { calls.log.append("clearGrade \($0)") }
        client.setSeen = { ids, seen in calls.log.append("setSeen \(ids) \(seen)") }
        client.recordEvent = { calls.log.append("event \($0.type.rawValue) undone=\($0.undone)") }
        client.excludeCompany = { calls.log.append("exclude \($0)"); return CompanyExclusion(name: "Monzo", added: true) }
        client.unexcludeCompany = { id, remove in calls.log.append("unexclude \(id) \(remove)") }
        client.note = { JobNote(jobID: $0, body: "old") }
        client.putNote = { id, body in calls.log.append("putNote \(id) \(body)") }
        client.cvTemplates = {
            [CVTemplate(docID: "d", tabID: "t", title: "Backend", isDefault: !calls.noDefaultCV)]
        }
        client.createDraft = { id, cv in
            calls.log.append("createDraft \(id) \(cv.title)")
            return DraftRef(id: "draft-\(id)")
        }
        let model = TriageModel(client: client)
        await model.load()
        return model
    }

    @Test func excludeDropsTheCompanysJobsAndUndoRestoresThem() async {
        let model = await model()

        await model.decide(.excludeCompany).value
        #expect(model.current == nil)
        await model.undo().value

        #expect(calls.log == ["exclude company-monzo", "unexclude company-monzo true"])
        #expect(model.queues[.direct]?.map(\.id) == ["a", "b"])
    }

    @Test func reclassifyMovesTheCompanysCardsAndUndoRestoresTheExactKind() async {
        var client = APIClient.mock(jobs: [Job.sample.with(id: "a", kind: .likelyDirect), Job.sample.with(id: "b", kind: .likelyDirect)])
        let calls = calls
        client.setCompanyKind = { id, kind in calls.log.append("set \(id) \(kind.rawValue)") }
        client.undoCompanyKind = { id, previous in calls.log.append("undo \(id) \(previous.rawValue)") }
        client.job = { _ in Job.sample.with(id: "a", kind: .likelyDirect) }
        let model = TriageModel(client: client)
        await model.load()

        await model.decide(.reclassify(to: .agency)).value

        #expect(model.queues[.direct]?.isEmpty == true)
        #expect(model.queues[.agency]?.map(\.id) == ["a", "b"])
        #expect(model.queues[.agency]?.allSatisfy { $0.companyKind == .agency } == true)
        #expect(model.unseen[.direct] == 0)
        #expect(model.unseen[.agency] == 2)

        model.stack = .agency
        await model.undo().value

        #expect(calls.log == ["set company-monzo agency", "undo company-monzo likely_direct"])
        #expect(model.stack == .direct)
        #expect(model.queues[.direct]?.map(\.companyKind) == [.likelyDirect, .likelyDirect])
        #expect(model.queues[.agency]?.isEmpty == true)
        #expect(model.unseen[.direct] == 2)
    }

    @Test func noteKeepsTheCardAndUndoPutsThePreviousBodyBack() async {
        let model = await model()

        await model.decide(.note("new")).value
        #expect(model.current?.id == "a")
        await model.undo().value

        #expect(calls.log == ["putNote a new", "putNote a old"])
        #expect(model.queues[.direct]?.map(\.id) == ["a", "b"])
    }

    @Test func shortlistThenUndoDeletesTheApplication() async throws {
        let model = await model()

        await model.decide(.shortlist(draftCV: false)).value
        #expect(model.current?.id == "b")
        await model.undo().value

        #expect(calls.log == ["shortlist a", "deleteApplication app-a"])
        #expect(model.current?.id == "a")
    }

    @Test func shortlistAndDraftStartsADraftFromTheDefaultCV() async {
        let model = await model()

        await model.decide(.shortlist(draftCV: true)).value

        #expect(calls.log == ["shortlist a", "createDraft a Backend"])
        #expect(!model.hasFailure)
    }

    @Test func plainShortlistNeverDrafts() async {
        let model = await model()

        await model.decide(.shortlist(draftCV: false)).value

        #expect(calls.log == ["shortlist a"])
    }

    @Test func shortlistAndDraftWithoutADefaultCVShortlistsAndSaysSo() async {
        let model = await model()
        calls.noDefaultCV = true

        await model.decide(.shortlist(draftCV: true)).value

        #expect(calls.log == ["shortlist a"])
        #expect(model.undoStack.count == 1)
        #expect(model.hasFailure)
        #expect(model.failureMessage.contains("default CV"))
    }

    @Test func draftCVFromTheBannerStartsADraft() async throws {
        let model = await model()
        await model.decide(.shortlist(draftCV: false)).value
        let decision = try #require(model.banner)

        await model.draftCV(decision).value

        #expect(calls.log == ["shortlist a", "createDraft a Backend"])
        #expect(model.banner == nil)
    }

    @Test func dismissPostsEventAndUndoClearsGrade() async {
        let model = await model()

        await model.decide(.dismiss(reasons: [.tech, .companySize])).value
        await model.undo().value

        #expect(calls.log == [
            "dismiss a [\"tech\", \"company_size\"]", "event job_dismissed undone=false",
            "clearGrade a", "event job_dismissed undone=true",
        ])
    }

    @Test func skipMarksSeenAndUndoUnmarksIt() async {
        let model = await model()

        await model.decide(.skip).value
        #expect(model.banner == nil)
        await model.undo().value

        #expect(calls.log == ["setSeen [\"a\"] true", "setSeen [\"a\"] false"])
    }

    @Test func undoWalksBackNewestFirst() async {
        let model = await model()

        await model.decide(.shortlist(draftCV: false)).value
        await model.decide(.skip).value
        await model.undo().value

        #expect(calls.log.last == "setSeen [\"b\"] false")
        #expect(model.current?.id == "b")
        #expect(model.undoStack.count == 1)
    }

    @Test func failedDecisionPutsTheCardBack() async {
        let model = await model()
        calls.failShortlist = true

        await model.decide(.shortlist(draftCV: false)).value

        #expect(model.current?.id == "a")
        #expect(model.undoStack.isEmpty)
        #expect(model.hasFailure)
    }

    private func detourModel(store: OfflineStore = .memory()) async -> TriageModel {
        var client = APIClient.mock()
        client.triageQueue = { stack in
            let items = switch stack {
            case .direct: [Job.sample.with(id: "d1")]
            case .agency: [Job.sample.with(id: "g1", kind: .agency), .sample.with(id: "g2", kind: .likelyAgency)]
            }
            return JobPage(items: items, total: stack == .agency ? 7 : 1)
        }
        client.dismiss = { _, _ in }
        client.recordEvent = { _ in }
        let model = TriageModel(client: client, store: store)
        await model.load()
        return model
    }

    @Test func opensOnDirectWithTheAgencyStacksUnseenCount() async {
        let model = await detourModel()

        #expect(model.stack == .direct)
        #expect(model.current?.id == "d1")
        #expect(model.unseen[.agency] == 7)
        #expect(model.unseen[.direct] == 1)
    }

    @Test func switchingStackServesThatStacksCards() async {
        let model = await detourModel()

        model.stack = .agency

        #expect(model.current?.id == "g1")
        #expect(model.upcoming.map(\.id) == ["g2"])
    }

    @Test func decidingOnAgencyDecrementsItsCountAndUndoRestoresIt() async {
        let model = await detourModel()
        model.stack = .agency

        await model.decide(.dismiss(reasons: [])).value
        #expect(model.current?.id == "g2")
        #expect(model.unseen[.agency] == 6)
        await model.undo().value

        #expect(model.current?.id == "g1")
        #expect(model.unseen[.agency] == 7)
        #expect(model.queues[.direct]?.map(\.id) == ["d1"])
    }

    @Test func offlineEachStackServesItsOwnCache() async throws {
        let store = OfflineStore.memory(cards: [.sample.with(id: "d1")])
        try store.saveCards([.sample.with(id: "g1", kind: .agency)], .agency)
        var client = APIClient.mock()
        client.triageQueue = { _ in throw URLError(.notConnectedToInternet) }
        let model = TriageModel(client: client, store: store)

        await model.load()

        #expect(model.current?.id == "d1")
        model.stack = .agency
        #expect(model.current?.id == "g1")
        #expect(model.unseen[.agency] == 1)
    }
}

private extension Job {
    func with(id: String, kind: CompanyKind? = nil) -> Job {
        Job(id: id, title: title, companyID: companyID, companySlug: companySlug, location: location,
            workArrangement: workArrangement, salaryRaw: salaryRaw, url: url, postedAt: postedAt,
            firstDiscoveredAt: firstDiscoveredAt, band: band, score: score, breakdown: breakdown,
            description: description, companyKind: kind ?? companyKind)
    }
}
