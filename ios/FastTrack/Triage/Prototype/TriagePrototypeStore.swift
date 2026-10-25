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

        var removesCard: Bool { self != .note && self != .draft }
        var isPositive: Bool { self == .shortlist || self == .shortlistDraft }
    }

    struct Entry: Identifiable, Equatable {
        let id = UUID()
        let action: Action
        let job: PrototypeJob
        let index: Int
        let removed: Bool
    }

    private(set) var queue: [PrototypeJob] = PrototypeJob.fixtures
    private(set) var log: [Entry] = []
    var banner: Entry?

    var current: PrototypeJob? { queue.first }
    var canUndo: Bool { log.contains(where: \.removed) }
    var lastRemoved: Entry? { log.last(where: \.removed) }

    func act(_ action: Action, on job: PrototypeJob) {
        let index = queue.firstIndex(of: job) ?? 0
        if action == .exclude {
            queue.removeAll { $0.company == job.company }
        } else if action.removesCard {
            queue.removeAll { $0 == job }
        }
        let entry = Entry(action: action, job: job, index: index, removed: action.removesCard)
        log.append(entry)
        banner = action == .skip ? nil : entry
    }

    func markSeen(_ job: PrototypeJob) {
        guard !log.contains(where: { $0.job == job }) else { return }
        log.append(Entry(action: .skip, job: job, index: 0, removed: false))
    }

    func undo() {
        guard let i = log.lastIndex(where: \.removed) else { return }
        let entry = log.remove(at: i)
        let restored = entry.action == .exclude
            ? PrototypeJob.fixtures.filter { $0.company == entry.job.company }
            : [entry.job]
        for job in restored.reversed() where !queue.contains(job) {
            queue.insert(job, at: min(entry.index, queue.count))
        }
        banner = nil
    }

    func reset() {
        queue = PrototypeJob.fixtures
        log = []
        banner = nil
    }

    var stateLine: String {
        let counts = Dictionary(grouping: log, by: \.action).mapValues(\.count)
        let parts = counts.sorted { $0.key.rawValue < $1.key.rawValue }.map { "\($0.key.rawValue) \($0.value)" }
        return (["\(queue.count) left"] + parts).joined(separator: " · ")
    }
}
#endif
