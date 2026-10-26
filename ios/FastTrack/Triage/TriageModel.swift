import Observation
import SwiftUI

@Observable
final class TriageModel {
    private(set) var queue: [Job] = []
    private(set) var loadState: LoadState<Void> = .idle
    private(set) var undoStack: [TriageDecision] = []
    private(set) var banner: TriageDecision?
    private(set) var lastRemoved: TriageAction?
    private(set) var failureMessage = ""
    var hasFailure = false

    private let client: APIClient

    init(client: APIClient) {
        self.client = client
    }

    var current: Job? { queue.first }
    var upcoming: [Job] { Array(queue.dropFirst().prefix(3)) }

    func load() async {
        if queue.isEmpty { loadState = .loading }
        do {
            try await reloadQueue()
            loadState = .loaded(())
        } catch is CancellationError {
            return
        } catch {
            loadState = .failed(error)
        }
    }

    @discardableResult
    func decide(_ action: TriageAction) -> Task<Void, Never> {
        guard let job = queue.first else { return Task {} }
        if !action.keepsCard { queue.removeFirst() }
        lastRemoved = action
        banner = nil
        return Task {
            do {
                let receipt = try await perform(action, on: job)
                if case .excludeCompany = action { queue.removeAll { $0.companyID == job.companyID } }
                let decision = TriageDecision(id: UUID(), job: job, action: action, receipt: receipt)
                undoStack.append(decision)
                if action != .skip { banner = decision }
            } catch {
                if !action.keepsCard { queue.insert(job, at: 0) }
                fail(error)
            }
        }
    }

    @discardableResult
    func undo() -> Task<Void, Never> {
        guard let decision = undoStack.popLast() else { return Task {} }
        let keepsCard = decision.action.keepsCard
        if !keepsCard { queue.insert(decision.job, at: 0) }
        banner = nil
        return Task {
            do {
                try await revert(decision)
            } catch {
                if !keepsCard { queue.removeAll { $0.id == decision.job.id } }
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

    private func perform(_ action: TriageAction, on job: Job) async throws -> UndoReceipt? {
        switch action {
        case .shortlist(let draftCV):
            let id = try await client.shortlist(job.id).id
            if draftCV { await startDraft(for: job) }
            return .application(id: id)
        case .dismiss(let reasons):
            try await client.dismiss(job.id, reasons)
            try? await client.recordEvent(EventInput(type: .jobDismissed, subjectID: job.id, reason: "", undone: false))
            return .grade
        case .skip:
            try await client.setSeen([job.id], true)
            return .seen
        case .excludeCompany:
            let exclusion = try await client.excludeCompany(job.companyID)
            return .exclusion(companyID: job.companyID, removeName: exclusion.added)
        case .note(let body):
            let previous = try await client.note(job.id).body
            try await client.putNote(job.id, body)
            return .note(previous: previous)
        }
    }

    private func revert(_ decision: TriageDecision) async throws {
        switch decision.receipt {
        case .application(let id):
            try await client.deleteApplication(id)
        case .grade:
            try await client.clearGrade(decision.job.id)
            try? await client.recordEvent(EventInput(type: .jobDismissed, subjectID: decision.job.id, reason: "", undone: true))
        case .seen:
            try await client.setSeen([decision.job.id], false)
        case .exclusion(let companyID, let removeName):
            try await client.unexcludeCompany(companyID, removeName)
            try? await reloadQueue()
        case .note(let previous):
            try await client.putNote(decision.job.id, previous)
        case nil:
            break
        }
    }

    private func reloadQueue() async throws {
        let decided = Set(undoStack.filter { !$0.action.keepsCard }.map(\.job.id))
        queue = try await client.triageQueue().filter { !decided.contains($0.id) }
    }

    private func fail(_ error: any Error) {
        fail(error.localizedDescription)
    }

    private func fail(_ message: String) {
        failureMessage = message
        hasFailure = true
    }
}
