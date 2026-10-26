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

    var symbol: String {
        switch self {
        case .chase: "arrow.uturn.forward"
        case .draft: "doc.richtext"
        case .task: "circle"
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
}

struct DueItemActions: View {
    enum Style { case regular, compact, prominent }

    let item: DueItem
    let actions: DueActions
    var style: Style = .regular
    @Environment(TriagePrototypeStore.self) private var store

    var body: some View {
        if style == .prominent {
            VStack(spacing: 10) {
                primary
                HStack(spacing: 10) { secondaries }
            }
        } else {
            HStack(spacing: 8) {
                primary
                secondaries
            }
        }
    }

    @ViewBuilder
    private var primary: some View {
        switch item {
        case .chase(let chase):
            pill("Done", "checkmark", primary: true) { store.completeChase(chase) }
        case .draft(let draft):
            switch draft.state {
            case .pending: ProgressView().frame(height: style == .prominent ? 52 : 36)
            case .ready: pill("Mark applied", "paperplane", primary: true) { store.markApplied(draft) }
            case .failed: pill("Try again", "arrow.clockwise", primary: true) { store.retry(draft) }
            }
        case .task(let task):
            pill("Done", "checkmark", primary: true) { store.completeTask(task) }
        }
    }

    @ViewBuilder
    private var secondaries: some View {
        switch item {
        case .chase(let chase):
            Menu {
                ForEach([(2, "2 days"), (7, "1 week"), (14, "2 weeks")], id: \.0) { days, label in
                    Button(label) { withAnimation(.snappy) { store.snooze(chase, days: days) } }
                }
            } label: {
                pillLabel("Snooze", "moon.zzz", primary: false)
            }
            pill("Log contact", "person.badge.plus", primary: false) { actions.logContact(chase.job) }
        case .draft(let draft) where draft.state == .ready:
            pill("View PDF", "doc.text.magnifyingglass", primary: false) { actions.openPDF(draft) }
        default:
            EmptyView()
        }
    }

    private func pill(_ title: String, _ symbol: String, primary: Bool, action: @escaping () -> Void) -> some View {
        Button { withAnimation(.snappy) { action() } } label: { pillLabel(title, symbol, primary: primary) }
            .buttonStyle(.plain)
    }

    @ViewBuilder
    private func pillLabel(_ title: String, _ symbol: String, primary: Bool) -> some View {
        let iconOnly = style == .compact && !primary
        Label(title, systemImage: symbol)
            .font((style == .prominent ? Font.body : .subheadline).weight(.semibold))
            .lineLimit(1)
            .foregroundStyle(primary ? Color.white : Color.primary)
            .modifier(IconOnly(isOn: iconOnly))
            .padding(.horizontal, iconOnly ? 0 : (style == .prominent ? 18 : 14))
            .frame(minWidth: iconOnly ? 44 : nil, minHeight: style == .prominent ? 52 : 44)
            .frame(maxWidth: style == .prominent ? .infinity : nil)
            .background(primary ? AnyShapeStyle(Color.brandTint) : AnyShapeStyle(Color(.tertiarySystemFill)), in: .capsule)
            .accessibilityLabel(title)
    }
}

private struct IconOnly: ViewModifier {
    let isOn: Bool

    func body(content: Content) -> some View {
        if isOn { content.labelStyle(.iconOnly) } else { content.labelStyle(.titleAndIcon) }
    }
}

struct TaskCheck: View {
    let task: PrototypeTask
    @Environment(TriagePrototypeStore.self) private var store
    @State private var struck = false

    var body: some View {
        Button {
            struck = true
            Task {
                try? await Task.sleep(for: .milliseconds(900))
                withAnimation(.snappy) { store.completeTask(task) }
            }
        } label: {
            Image(systemName: struck ? "checkmark.circle.fill" : "circle")
                .font(.title2)
                .foregroundStyle(struck ? Color.brandTint : Color.secondary)
                .contentTransition(.symbolEffect(.replace))
                .frame(width: 44, height: 44)
        }
        .buttonStyle(.plain)
        .sensoryFeedback(.success, trigger: struck) { _, new in new }
        .accessibilityLabel("Mark \(task.type) done")
    }
}
#endif
