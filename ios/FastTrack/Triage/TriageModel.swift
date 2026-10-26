import Observation
import SwiftUI

@Observable
final class TriageModel {
    static let cacheSize = 50

    private(set) var queue: [Job] = []
    private(set) var loadState: LoadState<Void> = .idle
    private(set) var undoStack: [TriageDecision] = []
    private(set) var banner: TriageDecision?
    private(set) var lastRemoved: TriageAction?
    private(set) var failureMessage = ""
    private(set) var queued: [QueuedAction]
    private(set) var isOffline = false
    private(set) var syncNote: String?
    private(set) var sessionExpired = false
    var hasFailure = false

    private let client: APIClient
    private let store: OfflineStore
    private var receipts: [UUID: UndoReceipt] = [:]

    init(client: APIClient, store: OfflineStore = .memory()) {
        self.client = client
        self.store = store
        queued = (try? store.loadQueue()) ?? []
    }

    var current: Job? { queue.first }
    var upcoming: [Job] { Array(queue.dropFirst().prefix(3)) }

    func load() async {
        if queue.isEmpty { loadState = .loading }
        await sync()
        do {
            try await reloadQueue()
            isOffline = false
            loadState = .loaded(())
        } catch is CancellationError {
            return
        } catch is URLError {
            isOffline = true
            if queue.isEmpty { queue = (try? store.loadCards()) ?? [] }
            loadState = .loaded(())
        } catch {
            loadState = .failed(error)
        }
    }

    @discardableResult
    func decide(_ action: TriageAction) -> Task<Void, Never> {
        guard let job = queue.first else { return Task {} }
        if !action.keepsCard { queue.removeFirst() }
        persistCards()
        lastRemoved = action
        banner = nil
        let id = UUID()
        let kind = kind(of: action, on: job)
        return Task {
            var receipt: UndoReceipt?
            do {
                guard !isOffline else { throw URLError(.notConnectedToInternet) }
                receipt = try await client.perform(kind, jobID: job.id)
            } catch where cannotReach(error) {
                enqueue(QueuedAction(id: id, jobID: job.id, kind: kind, queuedAt: .now), after: error)
            } catch {
                if !action.keepsCard { queue.insert(job, at: 0) }
                persistCards()
                fail(error)
                return
            }
            if case .excludeCompany = action { queue.removeAll { $0.companyID == job.companyID } }
            persistCards()
            let decision = TriageDecision(id: id, job: job, action: action, receipt: receipt)
            undoStack.append(decision)
            if action != .skip { banner = decision }
        }
    }

    @discardableResult
    func undo() -> Task<Void, Never> {
        guard let decision = undoStack.popLast() else { return Task {} }
        let keepsCard = decision.action.keepsCard
        if !keepsCard { queue.insert(decision.job, at: 0) }
        persistCards()
        banner = nil
        if let index = queued.firstIndex(where: { $0.id == decision.id }) {
            queued.remove(at: index)
            persistQueue()
            return Task {}
        }
        return Task {
            do {
                guard !isOffline else { throw URLError(.notConnectedToInternet) }
                try await client.revert(decision.receipt, jobID: decision.job.id)
                if case .exclusion = decision.receipt { try? await reloadQueue() }
            } catch where cannotReach(error) && decision.receipt != nil {
                receipts[decision.id] = decision.receipt
                enqueue(QueuedAction(id: UUID(), jobID: decision.job.id, kind: .undo(of: decision.id), queuedAt: .now), after: error)
            } catch {
                if !keepsCard { queue.removeAll { $0.id == decision.job.id } }
                persistCards()
                undoStack.append(decision)
                fail(error)
            }
        }
    }

    func currentNote() async throws -> String {
        guard let job = current else { return "" }
        return try await client.note(job.id).body
    }

    func dismissBanner(_ id: UUID) {
        if banner?.id == id { banner = nil }
    }

    func clearSyncNote() {
        syncNote = nil
    }

    private func kind(of action: TriageAction, on job: Job) -> QueuedAction.Kind {
        switch action {
        case .shortlist: .shortlist(draftCV: false)
        case .dismiss(let reasons): .dismiss(reasons: reasons)
        case .skip: .skip
        case .excludeCompany: .excludeCompany(companyID: job.companyID)
        case .note(let body): .note(body: body)
        }
    }

    private func cannotReach(_ error: any Error) -> Bool {
        error is URLError || error as? APIError == .unauthorized
    }

    private func enqueue(_ action: QueuedAction, after error: any Error) {
        queued.append(action)
        persistQueue()
        if error is URLError { isOffline = true } else { sessionExpired = true }
    }

    private func sync() async {
        guard !queued.isEmpty else { return }
        var local = receipts
        let outcome = await replay(queued, client: client, receipts: &local)
        receipts = local
        queued = outcome.remaining
        persistQueue()
        undoStack = undoStack.map { decision in
            var decision = decision
            decision.receipt = decision.receipt ?? receipts[decision.id]
            return decision
        }
        if outcome.result != ReplayResult(synced: 0, skipped: 0) { syncNote = outcome.result.note }
        if outcome.stoppedBy is URLError { isOffline = true }
        if outcome.stoppedBy as? APIError == .unauthorized { sessionExpired = true }
    }

    private func reloadQueue() async throws {
        let decided = Set(undoStack.filter { !$0.action.keepsCard }.map(\.job.id))
        queue = try await client.triageQueue().filter { !decided.contains($0.id) }
        persistCards()
    }

    private func persistCards() {
        try? store.saveCards(Array(queue.prefix(Self.cacheSize)))
    }

    private func persistQueue() {
        try? store.saveQueue(queued)
    }

    private func fail(_ error: any Error) {
        failureMessage = error.localizedDescription
        hasFailure = true
    }
}
