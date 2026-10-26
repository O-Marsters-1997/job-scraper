#if DEBUG
import SwiftUI

enum DueItem: Identifiable, Hashable {
    case chase(PrototypeChase)
    case draft(PrototypeDraft)
    case task(PrototypeTask)

    var id: UUID {
        switch self {
        case .chase(let c): c.id
        case .draft(let d): d.id
        case .task(let t): t.id
        }
    }

    var job: PrototypeJob {
        switch self {
        case .chase(let c): c.job
        case .draft(let d): d.job
        case .task(let t): t.job
        }
    }

    var kind: String {
        switch self {
        case .chase: "Chase"
        case .draft: "CV draft"
        case .task(let t): t.type
        }
    }

    var accent: Color {
        switch self {
        case .chase(let c): c.overdueDays > 0 ? .red : .orange
        case .draft(let d): d.state == .failed ? .red : .brandTint
        case .task: .secondary
        }
    }

    var status: String {
        switch self {
        case .chase(let c):
            c.overdueDays > 0 ? "Overdue by \(c.overdueDays) \(c.overdueDays == 1 ? "day" : "days")" : "Chase today"
        case .draft(let d):
            switch d.state {
            case .pending: "Drafting…"
            case .ready: "Ready · \(d.template)"
            case .failed: "Couldn't draft"
            }
        case .task: "Open task"
        }
    }
}

extension TriagePrototypeStore {
    var dueChases: [PrototypeChase] { chases.filter { $0.chaseBy <= .now }.sorted { $0.chaseBy < $1.chaseBy } }

    var dueItems: [DueItem] {
        dueChases.map(DueItem.chase)
            + drafts.sorted { rank($0) < rank($1) }.map(DueItem.draft)
            + tasks.map(DueItem.task)
    }

    var dueSummary: String {
        let parts = [
            dueChases.count > 0 ? "\(dueChases.count) \(dueChases.count == 1 ? "chase" : "chases")" : nil,
            drafts.filter { $0.state == .ready }.count > 0 ? "\(drafts.filter { $0.state == .ready }.count) CV ready" : nil,
            tasks.count > 0 ? "\(tasks.count) \(tasks.count == 1 ? "task" : "tasks")" : nil,
        ].compactMap { $0 }
        return parts.isEmpty ? "Nothing due" : parts.joined(separator: ", ")
    }

    private func rank(_ draft: PrototypeDraft) -> Int {
        switch draft.state {
        case .ready: 0
        case .failed: 1
        case .pending: 2
        }
    }
}

struct DueActions {
    let openPDF: (PrototypeDraft) -> Void
    let logContact: (PrototypeJob) -> Void
    let openJob: (PrototypeJob) -> Void
}
#endif
