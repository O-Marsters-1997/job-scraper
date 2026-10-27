import Observation
import SwiftUI

struct DueUndo: Identifiable {
    let id = UUID()
    let title: String
    let jobID: String
    let jobTitle: String
    let restoreLocally: (DueJob) -> DueJob
    let revert: () async throws -> Void
}

@Observable
final class DueModel {
    private(set) var jobs: [DueJob] = []
    private(set) var loadState: LoadState<Void> = .idle
    private(set) var undoStack: [DueUndo] = []
    private(set) var banner: DueUndo?
    private(set) var failureMessage = ""
    var hasFailure = false

    private let client: APIClient
    private let now: () -> Date

    init(client: APIClient, now: @escaping () -> Date = { .now }) {
        self.client = client
        self.now = now
    }

    var dueJobs: [DueJob] { jobs.filter { hasChaseDue($0) || !$0.tasks.isEmpty || $0.draft != nil } }
    var shortlisted: [DueJob] { jobs.filter { $0.showsApplyRow }.sorted { ($0.score ?? .min, $1.title) > ($1.score ?? .min, $0.title) } }

    func dueJobs(matching query: String) -> [DueJob] { dueJobs.filter { $0.matches(query) } }
    func shortlisted(matching query: String) -> [DueJob] { shortlisted.filter { $0.matches(query) } }
    var comingUp: [DueJob] { jobs.filter { $0.chaseBy != nil && !hasChaseDue($0) }.sorted { ($0.chaseBy ?? .distantFuture) < ($1.chaseBy ?? .distantFuture) } }
    var chasesDue: Int { jobs.filter(hasChaseDue).count }
    var readyDrafts: Int { jobs.filter { $0.draft?.state == .ready }.count }
    var hasPendingDraft: Bool { jobs.contains { $0.draft?.state == .pending } }
    var badge: Int { loadState.value == nil ? 0 : chasesDue + readyDrafts }

    func hasChaseDue(_ job: DueJob) -> Bool {
        guard let chaseBy = job.chaseBy else { return false }
        let calendar = Calendar.current
        guard let tomorrow = calendar.date(byAdding: .day, value: 1, to: calendar.startOfDay(for: now())) else { return false }
        return chaseBy < tomorrow
    }

    func overdueDays(_ job: DueJob) -> Int {
        guard let chaseBy = job.chaseBy else { return 0 }
        let calendar = Calendar.current
        let days = calendar.dateComponents([.day], from: calendar.startOfDay(for: chaseBy), to: calendar.startOfDay(for: now())).day ?? 0
        return max(days, 0)
    }

    func load() async {
        if jobs.isEmpty { loadState = .loading }
        do {
            jobs = try await client.due()
            loadState = .loaded(())
        } catch is CancellationError {
            return
        } catch {
            jobs = []
            loadState = .failed(error)
        }
    }

    func refreshWhilePending(every interval: Duration = .seconds(2)) async {
        while hasPendingDraft {
            try? await Task.sleep(for: interval)
            if Task.isCancelled { return }
            await load()
        }
    }

    @discardableResult
    func completeChase(_ job: DueJob, title: String = "Chase done") -> Task<Void, Never> {
        guard let previous = job.chaseBy else { return Task {} }
        let client = client
        return run(title, job, job.chasing(by: nil), restoreLocally: { $0.chasing(by: previous) }) {
            try await client.clearChase(job.applicationID)
        } revert: {
            try await client.setChase(job.applicationID, previous)
        }
    }

    @discardableResult
    func snooze(_ job: DueJob, days: Int) -> Task<Void, Never> {
        guard let previous = job.chaseBy,
              let target = Calendar.current.date(byAdding: .day, value: days, to: Calendar.current.startOfDay(for: now()))
        else { return Task {} }
        let client = client
        return run("Snoozed \(days) days", job, job.chasing(by: target), restoreLocally: { $0.chasing(by: previous) }) {
            try await client.setChase(job.applicationID, target)
        } revert: {
            try await client.setChase(job.applicationID, previous)
        }
    }

    @discardableResult
    func markApplied(_ job: DueJob) -> Task<Void, Never> {
        guard let taskID = job.applyTaskID else { return Task {} }
        if let task = job.tasks.first(where: { $0.id == taskID }) { return complete(task, in: job, title: "Marked applied") }
        let client = client
        return run("Marked applied", job, job.withApplyTask(nil), restoreLocally: { $0.withApplyTask(taskID) }) {
            try await client.setTaskDone(taskID, true)
        } revert: {
            try await client.setTaskDone(taskID, false)
        }
    }

    func delete(_ job: DueJob) async {
        let before = jobs
        jobs.removeAll { $0.id == job.id }
        do {
            try await client.deleteApplication(job.applicationID)
        } catch {
            jobs = before
            fail(error)
        }
    }

    func retryDraft(_ job: DueJob) async {
        do {
            guard let cv = try await client.cvTemplates().first(where: \.isDefault) else {
                fail("There's no default CV to draft from.")
                return
            }
            _ = try await client.createDraft(job.id, cv)
            await load()
        } catch {
            fail(error)
        }
    }

    func pdf(_ draft: DueJob.Draft) async -> URL? {
        do {
            return try await client.draftPDF(draft.id)
        } catch {
            fail(error)
            return nil
        }
    }

    @discardableResult
    func complete(_ task: DueJob.Task, in job: DueJob, title: String = "Task done") -> Task<Void, Never> {
        let client = client
        let index = job.tasks.firstIndex(of: task) ?? job.tasks.endIndex
        let restore: (DueJob) -> DueJob = { current in
            var tasks = current.tasks
            tasks.insert(task, at: min(index, tasks.endIndex))
            return current.withTasks(tasks)
        }
        return run(title, job, job.withTasks(job.tasks.filter { $0.id != task.id }), restoreLocally: restore) {
            try await client.setTaskDone(task.id, true)
        } revert: {
            try await client.setTaskDone(task.id, false)
        }
    }

    func logContact(_ job: DueJob, channel: ContactChannel, outcome: ContactOutcome) async -> Bool {
        do {
            _ = try await client.logContact(job.id, channel, outcome)
            if let current = jobs.first(where: { $0.id == job.id }), hasChaseDue(current) {
                await completeChase(current, title: "Contact logged").value
            }
            return true
        } catch {
            fail(error)
            return false
        }
    }

    @discardableResult
    func undo() -> Task<Void, Never> {
        guard let entry = undoStack.popLast() else { return Task {} }
        let current = jobs.first { $0.id == entry.jobID }
        if let current { replace(entry.restoreLocally(current)) }
        banner = nil
        return Task {
            do {
                try await entry.revert()
            } catch {
                if let current { replace(current) }
                undoStack.append(entry)
                fail(error)
            }
        }
    }

    func dismissBanner(_ id: UUID) {
        if banner?.id == id { banner = nil }
    }

    private func run(
        _ title: String, _ before: DueJob, _ after: DueJob, restoreLocally: @escaping (DueJob) -> DueJob,
        forward: @escaping () async throws -> Void, revert: @escaping () async throws -> Void
    ) -> Task<Void, Never> {
        replace(after)
        banner = nil
        return Task {
            do {
                try await forward()
                let entry = DueUndo(title: title, jobID: before.id, jobTitle: before.title, restoreLocally: restoreLocally, revert: revert)
                undoStack.append(entry)
                banner = entry
            } catch {
                replace(before)
                fail(error)
            }
        }
    }

    private func replace(_ job: DueJob) {
        guard let index = jobs.firstIndex(where: { $0.id == job.id }) else { return }
        jobs[index] = job
    }

    private func fail(_ error: any Error) {
        fail(error.localizedDescription)
    }

    private func fail(_ message: String) {
        failureMessage = message
        hasFailure = true
    }
}
