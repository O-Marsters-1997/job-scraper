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
            let decided = Set(undoStack.map(\.job.id))
            queue = try await client.triageQueue().filter { !decided.contains($0.id) }
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
        queue.removeFirst()
        lastRemoved = action
        banner = nil
        return Task {
            do {
                let receipt = try await perform(action, on: job)
                let decision = TriageDecision(id: UUID(), job: job, action: action, receipt: receipt)
                undoStack.append(decision)
                if action != .skip { banner = decision }
            } catch {
                queue.insert(job, at: 0)
                fail(error)
            }
        }
    }

    @discardableResult
    func undo() -> Task<Void, Never> {
        guard let decision = undoStack.popLast() else { return Task {} }
        queue.insert(decision.job, at: 0)
        banner = nil
        return Task {
            do {
                try await revert(decision)
            } catch {
                queue.removeAll { $0.id == decision.job.id }
                undoStack.append(decision)
                fail(error)
            }
        }
    }

    func dismissBanner(_ id: UUID) {
        if banner?.id == id { banner = nil }
    }

    private func perform(_ action: TriageAction, on job: Job) async throws -> UndoReceipt? {
        switch action {
        case .shortlist:
            return .application(id: try await client.shortlist(job.id).id)
        case .dismiss(let reasons):
            try await client.dismiss(job.id, reasons)
            try? await client.recordEvent(EventInput(type: .jobDismissed, subjectID: job.id, reason: "", undone: false))
            return .grade
        case .skip:
            try await client.setSeen([job.id], true)
            return .seen
        case .excludeCompany, .note:
            return nil
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
        case .exclusion, .note, nil:
            break
        }
    }

    private func fail(_ error: any Error) {
        failureMessage = error.localizedDescription
        hasFailure = true
    }
}
