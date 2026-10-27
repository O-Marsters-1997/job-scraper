import Foundation
import Testing
@testable import FastTrack

@MainActor
struct OfflineQueueTests {
    private final class Calls {
        var log: [String] = []
        var offline = false
        var shortlistError: APIError?
    }

    private let calls = Calls()

    private func client() -> APIClient {
        var client = APIClient.mock(jobs: [.sample.with(id: "a"), .sample.with(id: "b")])
        let calls = calls
        client.shortlist = { id in
            let jobID = String(id)
            if calls.offline { throw URLError(.notConnectedToInternet) }
            if let error = calls.shortlistError { throw error }
            calls.log.append("shortlist \(jobID)")
            return Application(id: "app-\(jobID)", jobID: jobID)
        }
        client.deleteApplication = { calls.log.append("deleteApplication \($0)") }
        client.setSeen = { ids, seen in
            if calls.offline { throw URLError(.notConnectedToInternet) }
            calls.log.append("setSeen \(ids) \(seen)")
        }
        client.triageQueue = { stack in
            if calls.offline { throw URLError(.notConnectedToInternet) }
            let items = stack == .direct ? [Job.sample.with(id: "a"), .sample.with(id: "b")] : []
            return JobPage(items: items, total: items.count)
        }
        return client
    }

    private func action(_ kind: QueuedAction.Kind, job: String, at seconds: TimeInterval) -> QueuedAction {
        QueuedAction(id: UUID(), jobID: job, kind: kind, queuedAt: Date(timeIntervalSince1970: seconds))
    }

    @Test func replayRunsOldestFirst() async {
        let queue = [action(.skip, job: "b", at: 20), action(.shortlist(draftCV: false), job: "a", at: 10)]
        var receipts: [UUID: UndoReceipt] = [:]

        let outcome = await replay(queue, client: client(), receipts: &receipts)

        #expect(calls.log == ["shortlist a", "setSeen [\"b\"] true"])
        #expect(outcome.result == ReplayResult(synced: 2, skipped: 0))
        #expect(outcome.remaining.isEmpty)
    }

    @Test func alreadyExistsCountsAsSynced() async {
        calls.shortlistError = .status(409)
        var receipts: [UUID: UndoReceipt] = [:]

        let outcome = await replay([action(.shortlist(draftCV: false), job: "a", at: 1)], client: client(), receipts: &receipts)

        #expect(outcome.result == ReplayResult(synced: 1, skipped: 0))
    }

    @Test func missingJobCountsAsSkipped() async {
        calls.shortlistError = .status(404)
        var receipts: [UUID: UndoReceipt] = [:]

        let outcome = await replay([action(.shortlist(draftCV: false), job: "a", at: 1)], client: client(), receipts: &receipts)

        #expect(outcome.result == ReplayResult(synced: 0, skipped: 1))
        #expect(outcome.remaining.isEmpty)
    }

    @Test func serverFailureKeepsTheActionQueued() async {
        calls.shortlistError = .status(500)
        var receipts: [UUID: UndoReceipt] = [:]

        let outcome = await replay([action(.shortlist(draftCV: false), job: "a", at: 1)], client: client(), receipts: &receipts)

        #expect(outcome.result.isEmpty)
        #expect(outcome.remaining.map(\.jobID) == ["a"])
    }

    @Test func lostConnectionKeepsTheRestQueued() async {
        calls.offline = true
        var receipts: [UUID: UndoReceipt] = [:]
        let queue = [action(.skip, job: "a", at: 1), action(.skip, job: "b", at: 2)]

        let outcome = await replay(queue, client: client(), receipts: &receipts)

        #expect(outcome.result == ReplayResult(synced: 0, skipped: 0))
        #expect(outcome.remaining.map(\.jobID) == ["a", "b"])
    }

    @Test func queuedUndoReversesTheSyncedAction() async {
        let shortlist = action(.shortlist(draftCV: false), job: "a", at: 1)
        let undo = action(.undo(of: shortlist.id), job: "a", at: 2)
        var receipts: [UUID: UndoReceipt] = [:]

        let outcome = await replay([shortlist, undo], client: client(), receipts: &receipts)

        #expect(calls.log == ["shortlist a", "deleteApplication app-a"])
        #expect(outcome.result == ReplayResult(synced: 2, skipped: 0))
    }

    @Test func offlineDecisionQueuesAndSurvivesRelaunch() async {
        let store = OfflineStore.memory(cards: [.sample.with(id: "a"), .sample.with(id: "b")])
        calls.offline = true
        let model = TriageModel(client: client(), store: store)
        await model.load()

        await model.decide(.shortlist(draftCV: false)).value

        #expect(model.isOffline)
        #expect(model.current?.id == "b")
        #expect(model.queued.map(\.jobID) == ["a"])
        #expect(TriageModel(client: client(), store: store).queued.map(\.jobID) == ["a"])
    }

    @Test func undoingAQueuedActionCancelsItLocally() async throws {
        let store = OfflineStore.memory(cards: [.sample.with(id: "a"), .sample.with(id: "b")])
        calls.offline = true
        let model = TriageModel(client: client(), store: store)
        await model.load()
        await model.decide(.shortlist(draftCV: false)).value

        await model.undo().value

        #expect(model.queued.isEmpty)
        #expect(model.current?.id == "a")
        #expect(try store.loadQueue().isEmpty)
    }

    @Test func reconnectingReplaysAndReportsTheNote() async {
        let store = OfflineStore.memory(cards: [.sample.with(id: "a"), .sample.with(id: "b")])
        calls.offline = true
        let model = TriageModel(client: client(), store: store)
        await model.load()
        await model.decide(.shortlist(draftCV: false)).value
        calls.offline = false

        await model.load()

        #expect(calls.log == ["shortlist a"])
        #expect(model.syncNote == "1 synced, 0 skipped")
        #expect(model.queued.isEmpty)
        #expect(!model.isOffline)
    }

    @Test func fileStoreRoundTripsCardsAndQueue() throws {
        let directory = FileManager.default.temporaryDirectory.appending(path: UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: directory) }
        let store = OfflineStore.files(in: directory)
        let dismiss = action(.dismiss(reasons: [.tech]), job: "a", at: 5)

        try store.saveQueue([dismiss])
        try store.saveCards([.sample], .direct)
        try store.saveCards([.sample.with(id: "agency-job")], .agency)

        let queue = try store.loadQueue()
        #expect(queue.map(\.id) == [dismiss.id])
        #expect(queue.first?.queuedAt == dismiss.queuedAt)
        #expect(try store.loadCards(.agency).map(\.id) == ["agency-job"])
        let cards = try store.loadCards(.direct)
        #expect(cards.map(\.id) == ["sample"])
        #expect(cards.first?.band == .great)
        #expect(cards.first?.breakdown == Job.sample.breakdown)
    }
}

private extension Job {
    func with(id: String) -> Job {
        Job(id: id, title: title, companyID: companyID, companySlug: companySlug, location: location,
            workArrangement: workArrangement, salaryRaw: salaryRaw, url: url, postedAt: postedAt,
            firstDiscoveredAt: firstDiscoveredAt, band: band, score: score, breakdown: breakdown,
            description: description, companyKind: companyKind)
    }
}
