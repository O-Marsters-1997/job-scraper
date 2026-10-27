import Observation
import SwiftUI

@Observable
final class TriageModel {
    static let cacheSize = 50

    var stack: TriageStack = .direct
    private(set) var queues: [TriageStack: [Job]] = [:]
    private(set) var unseen: [TriageStack: Int] = [:]
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

    var current: Job? { queues[stack]?.first }
    var upcoming: [Job] { Array((queues[stack] ?? []).dropFirst().prefix(3)) }

    func load() async {
        if queues.values.allSatisfy(\.isEmpty) { loadState = .loading }
        await sync()
        do {
            try await reloadQueue()
            isOffline = false
            loadState = .loaded(())
        } catch is CancellationError {
            return
        } catch is URLError {
            isOffline = true
            for stack in TriageStack.allCases where queues[stack, default: []].isEmpty {
                let cards = (try? store.loadCards(stack)) ?? []
                queues[stack] = cards
                unseen[stack] = cards.count
            }
            loadState = .loaded(())
        } catch {
            loadState = .failed(error)
        }
    }

    @discardableResult
    func decide(_ action: TriageAction) -> Task<Void, Never> {
        guard let job = current else { return Task {} }
        apply(action, to: job)
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
                if case .shortlist(draftCV: true) = action { await startDraft(for: job) }
            } catch where cannotReach(error) {
                enqueue(QueuedAction(id: id, jobID: job.id, kind: kind, queuedAt: .now), after: error)
            } catch {
                rollback(action, of: job)
                persistCards()
                fail(error)
                return
            }
            if case .excludeCompany = action { removeCompany(job.companyID) }
            persistCards()
            let decision = TriageDecision(id: id, job: job, action: action, receipt: receipt)
            undoStack.append(decision)
            if action != .skip { banner = decision }
        }
    }

    @discardableResult
    func undo() -> Task<Void, Never> {
        guard let decision = undoStack.popLast() else { return Task {} }
        rollback(decision.action, of: decision.job)
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
                apply(decision.action, to: decision.job)
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

    @discardableResult
    func draftCV(_ decision: TriageDecision) -> Task<Void, Never> {
        banner = nil
        return Task { await startDraft(for: decision.job) }
    }

    private func startDraft(for job: Job) async {
        do {
            guard let cv = try await client.cvTemplates().first(where: \.isDefault) else {
                fail("There's no default CV, so the job was shortlisted without a draft.")
                return
            }
            _ = try await client.createDraft(job.id, cv)
        } catch {
            fail("The job was shortlisted, but the CV draft didn't start: \(error.localizedDescription)")
        }
    }

    private func kind(of action: TriageAction, on job: Job) -> QueuedAction.Kind {
        switch action {
        case .shortlist(let draftCV): .shortlist(draftCV: draftCV)
        case .dismiss(let reasons): .dismiss(reasons: reasons)
        case .skip: .skip
        case .excludeCompany: .excludeCompany(companyID: job.companyID)
        case .reclassify(let kind): .reclassify(companyID: job.companyID, to: kind)
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
        if !outcome.result.isEmpty { syncNote = outcome.result.note }
        if outcome.stoppedBy is URLError { isOffline = true }
        if outcome.stoppedBy as? APIError == .unauthorized { sessionExpired = true }
    }

    private func reloadQueue() async throws {
        let decided = Set(undoStack.filter { !$0.action.keepsCard }.map(\.job.id))
        var pages: [TriageStack: JobPage] = [:]
        for stack in TriageStack.allCases { pages[stack] = try await client.triageQueue(stack) }
        for (stack, page) in pages {
            let items = page.items.filter { !decided.contains($0.id) }
            queues[stack] = items
            let dropped = page.items.count - items.count
            unseen[stack] = max((page.total ?? page.items.count) - dropped, items.count)
        }
        persistCards()
    }

    private func apply(_ action: TriageAction, to job: Job) {
        if case .reclassify(let kind) = action {
            move(company: job.companyID, to: kind)
        } else if !action.keepsCard {
            remove(job)
        }
    }

    private func rollback(_ action: TriageAction, of job: Job) {
        if case .reclassify = action {
            move(company: job.companyID, to: job.companyKind)
            stack = job.stack
        } else if !action.keepsCard {
            restore(job)
        }
    }

    private func move(company companyID: String, to kind: CompanyKind) {
        var moved: [Job] = []
        for stack in TriageStack.allCases {
            let jobs = queues[stack, default: []]
            let mine = jobs.filter { $0.companyID == companyID }
            queues[stack] = jobs.filter { $0.companyID != companyID }
            unseen[stack] = max((unseen[stack] ?? 0) - mine.count, 0)
            moved += mine
        }
        let target: TriageStack = kind.isAgency ? .agency : .direct
        queues[target, default: []].insert(contentsOf: moved.map { var job = $0; job.companyKind = kind; return job }, at: 0)
        unseen[target, default: 0] += moved.count
    }

    private func remove(_ job: Job) {
        let stack = job.stack
        guard let index = queues[stack]?.firstIndex(where: { $0.id == job.id }) else { return }
        queues[stack]?.remove(at: index)
        unseen[stack] = max((unseen[stack] ?? 0) - 1, 0)
    }

    private func restore(_ job: Job) {
        let stack = job.stack
        queues[stack, default: []].insert(job, at: 0)
        unseen[stack, default: 0] += 1
    }

    private func removeCompany(_ companyID: String) {
        for stack in TriageStack.allCases {
            let before = queues[stack, default: []].count
            queues[stack]?.removeAll { $0.companyID == companyID }
            unseen[stack] = max((unseen[stack] ?? 0) - (before - (queues[stack]?.count ?? 0)), 0)
        }
    }

    private func persistCards() {
        for stack in TriageStack.allCases {
            try? store.saveCards(Array(queues[stack, default: []].prefix(Self.cacheSize)), stack)
        }
    }

    private func persistQueue() {
        try? store.saveQueue(queued)
    }

    private func fail(_ error: any Error) {
        fail(error.localizedDescription)
    }

    private func fail(_ message: String) {
        failureMessage = message
        hasFailure = true
    }
}
