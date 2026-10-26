#if DEBUG
import Observation
import SwiftUI

@Observable
final class TriagePrototypeStore {
    enum Action: String {
        case shortlist = "Shortlisted"
        case shortlistDraft = "Shortlisted + CV draft"
        case dismiss = "Dismissed"
        case dismissReason = "Dismissed with reason"
        case exclude = "Excluded company"
        case skip = "Skipped"
        case note = "Note added"
        case draft = "CV draft started"
        case chaseDone = "Chase done"
        case snooze = "Chase snoozed"
        case taskDone = "Task done"
        case markApplied = "Marked applied"
        case contact = "Contact logged"

        var removesCard: Bool { [.shortlist, .shortlistDraft, .dismiss, .dismissReason, .exclude, .skip].contains(self) }
        var isPositive: Bool { self == .shortlist || self == .shortlistDraft }
        var isUndoable: Bool { removesCard || [.chaseDone, .snooze, .taskDone, .markApplied].contains(self) }
    }

    struct Entry: Identifiable, Equatable {
        let id = UUID()
        let action: Action
        let job: PrototypeJob
        let index: Int
        let removed: Bool
        var queued = false
    }

    private(set) var queue: [PrototypeJob] = PrototypeJob.fixtures
    private(set) var log: [Entry] = []
    var banner: Entry?
    var syncNote: String?

    var signedIn = !ProcessInfo.processInfo.arguments.contains("-prototype-login")
    var offline = ProcessInfo.processInfo.arguments.contains("-prototype-offline") {
        didSet { if oldValue && !offline { replay() } }
    }

    var chases = DueFixtures.chases()
    var tasks = DueFixtures.tasks()
    var drafts = DueFixtures.drafts()
    var notes: [PrototypeJob.ID: String] = [:]
    var contacts: [PrototypeJob.ID: [PrototypeContact]] = [:]

    private var undoSnapshots: [UUID: (chases: [PrototypeChase], tasks: [PrototypeTask], drafts: [PrototypeDraft])] = [:]

    var current: PrototypeJob? { queue.first }
    var canUndo: Bool { log.contains { $0.action.isUndoable && ($0.removed || $0.action != .skip) } }
    var lastRemoved: Entry? { log.last(where: \.removed) }
    var queuedCount: Int { log.filter(\.queued).count }
    var dueBadge: Int {
        chases.filter { $0.chaseBy <= .now }.count + drafts.filter { $0.state == .ready }.count
    }

    func act(_ action: TriagePrototypeStore.Action, on job: PrototypeJob) {
        let snapshot = (chases, tasks, drafts)
        let index = queue.firstIndex(of: job) ?? 0
        if action == .exclude {
            queue.removeAll { $0.company == job.company }
        } else if action.removesCard {
            queue.removeAll { $0 == job }
        }
        if action.isPositive {
            tasks.insert(PrototypeTask(type: "Apply", job: job), at: 0)
        }
        if action == .shortlistDraft || action == .draft {
            startDraft(for: job)
        }
        var entry = Entry(action: action, job: job, index: index, removed: action.removesCard)
        entry.queued = offline
        undoSnapshots[entry.id] = snapshot
        log.append(entry)
        banner = action == .skip ? nil : entry
    }

    func markSeen(_ job: PrototypeJob) {
        guard !log.contains(where: { $0.job == job }) else { return }
        log.append(Entry(action: .skip, job: job, index: 0, removed: false))
    }

    func undo() {
        guard let i = log.lastIndex(where: { $0.action.isUndoable && ($0.removed || $0.action != .skip) }) else { return }
        let entry = log.remove(at: i)
        if entry.removed {
            let restored = entry.action == .exclude
                ? PrototypeJob.fixtures.filter { $0.company == entry.job.company }
                : [entry.job]
            for job in restored.reversed() where !queue.contains(job) {
                queue.insert(job, at: min(entry.index, queue.count))
            }
        }
        if let snapshot = undoSnapshots.removeValue(forKey: entry.id) {
            (chases, tasks, drafts) = snapshot
        }
        banner = nil
    }

    func completeChase(_ chase: PrototypeChase) {
        record(.chaseDone, chase.job) { chases.removeAll { $0.id == chase.id } }
    }

    func snooze(_ chase: PrototypeChase, days: Int) {
        record(.snooze, chase.job) {
            guard let i = chases.firstIndex(of: chase) else { return }
            chases[i].chaseBy = Calendar.current.date(byAdding: .day, value: days, to: .now) ?? .now
        }
    }

    func completeTask(_ task: PrototypeTask) {
        record(.taskDone, task.job) { tasks.removeAll { $0.id == task.id } }
    }

    func markApplied(_ draft: PrototypeDraft) {
        record(.markApplied, draft.job) {
            drafts.removeAll { $0.id == draft.id }
            tasks.removeAll { $0.job == draft.job && $0.type == "Apply" }
        }
    }

    func retry(_ draft: PrototypeDraft) {
        drafts.removeAll { $0.id == draft.id }
        startDraft(for: draft.job)
    }

    func logContact(_ contact: PrototypeContact, for job: PrototypeJob) {
        contacts[job.id, default: []].insert(contact, at: 0)
        record(.contact, job) {}
    }

    func saveNote(_ text: String, for job: PrototypeJob) {
        notes[job.id] = text
        act(.note, on: job)
    }

    func reset() {
        queue = PrototypeJob.fixtures
        log = []
        banner = nil
        chases = DueFixtures.chases()
        tasks = DueFixtures.tasks()
        drafts = DueFixtures.drafts()
        notes = [:]
        contacts = [:]
        undoSnapshots = [:]
    }

    private func record(_ action: Action, _ job: PrototypeJob, change: () -> Void) {
        let snapshot = (chases, tasks, drafts)
        change()
        let entry = Entry(action: action, job: job, index: 0, removed: false)
        undoSnapshots[entry.id] = snapshot
        log.append(entry)
        banner = entry
    }

    private func startDraft(for job: PrototypeJob) {
        let draft = PrototypeDraft(job: job, template: "Default", state: .pending)
        drafts.insert(draft, at: 0)
        Task {
            try? await Task.sleep(for: .seconds(5))
            if let i = drafts.firstIndex(where: { $0.id == draft.id }) { drafts[i].state = .ready }
        }
    }

    private func replay() {
        let count = queuedCount
        guard count > 0 else { return }
        for i in log.indices { log[i].queued = false }
        syncNote = "\(count) synced, 0 skipped"
    }

    var stateLine: String {
        let counts = Dictionary(grouping: log, by: \.action).mapValues(\.count)
        let parts = counts.sorted { $0.key.rawValue < $1.key.rawValue }.map { "\($0.key.rawValue) \($0.value)" }
        return (["\(queue.count) left"] + parts).joined(separator: " · ")
    }
}
#endif
