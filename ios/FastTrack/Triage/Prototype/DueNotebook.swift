#if DEBUG
import SwiftUI

struct DueNotebook: View {
    let actions: DueActions
    @Environment(TriagePrototypeStore.self) private var store
    @State private var opened: Set<PrototypeJob.ID>?

    private var byJob: [(PrototypeJob, [DueItem])] {
        var order: [PrototypeJob] = []
        var groups: [PrototypeJob.ID: [DueItem]] = [:]
        for item in store.dueItems {
            if groups[item.job.id] == nil { order.append(item.job) }
            groups[item.job.id, default: []].append(item)
        }
        return order.map { ($0, groups[$0.id] ?? []) }
    }

    private var comingUp: [PrototypeChase] {
        store.chases.filter { $0.chaseBy > .now }.sorted { $0.chaseBy < $1.chaseBy }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                header
                if !comingUp.isEmpty { upcoming }
                VStack(spacing: 10) {
                    ForEach(byJob, id: \.0.id) { job, items in
                        JobCard(job: job, items: items, actions: actions, isOpen: opened?.contains(job.id) ?? false) {
                            var set = opened ?? []
                            if set.contains(job.id) { set.remove(job.id) } else { set.insert(job.id) }
                            withAnimation(.snappy(duration: 0.22)) { opened = set }
                        }
                        .transition(.opacity)
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 8)
            .padding(.bottom, 24)
            .animation(.snappy(duration: 0.25), value: store.dueItems)
            .animation(.snappy(duration: 0.25), value: store.chases)
        }
        .background(Color(.systemBackground))
        .onAppear {
            if opened == nil { opened = Set(byJob.prefix(1).map(\.0.id)) }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(Date.now.formatted(.dateTime.weekday(.wide).day().month(.wide).locale(Locale(identifier: "en_GB"))))
                .font(.system(.title, design: .serif))
            Text(store.dueSummary)
                .font(.footnote)
                .foregroundStyle(.secondary)
                .contentTransition(.numericText())
        }
    }

    private var upcoming: some View {
        VStack(alignment: .leading, spacing: 6) {
            SectionLabel(title: "Coming up", count: comingUp.count)
            ScrollView(.horizontal) {
                HStack(spacing: 6) {
                    ForEach(comingUp) { chase in
                        HStack(spacing: 6) {
                            Text(chase.chaseBy.shortGB).font(.caption.monospacedDigit()).foregroundStyle(.orange)
                            Text("Chase \(chase.job.company)").font(.footnote.weight(.medium))
                        }
                        .padding(.horizontal, 10)
                        .frame(minHeight: 30)
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(Color(.separator)))
                    }
                }
            }
            .scrollIndicators(.hidden)
        }
    }
}

private struct SectionLabel: View {
    let title: String
    let count: Int

    var body: some View {
        HStack {
            Text(title)
            Spacer()
            Text(count, format: .number).monospacedDigit()
        }
        .font(.footnote.weight(.medium))
        .foregroundStyle(.secondary)
    }
}

private struct JobCard: View {
    let job: PrototypeJob
    let items: [DueItem]
    let actions: DueActions
    let isOpen: Bool
    let toggle: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Button(action: toggle) {
                VStack(alignment: .leading, spacing: 6) {
                    HStack(alignment: .firstTextBaseline, spacing: 10) {
                        Text(job.company).font(.system(.title3, design: .serif))
                        HStack(alignment: .firstTextBaseline, spacing: 4) {
                            Text(job.band.rawValue).font(.caption.weight(.semibold)).foregroundStyle(job.band.colour)
                            Text(job.score, format: .number).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                        }
                        Spacer(minLength: 6)
                        Text(items.count, format: .number).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
                        Image(systemName: "chevron.right")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.tertiary)
                            .rotationEffect(.degrees(isOpen ? 90 : 0))
                    }
                    HStack(spacing: 8) {
                        Text(job.title).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
                        Spacer(minLength: 4)
                        if let urgent { DotLabel(text: urgent.0, hue: urgent.1).opacity(isOpen ? 0 : 1) }
                    }
                }
                .padding(.horizontal, 14)
                .padding(.vertical, 14)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityHint(isOpen ? "Collapse" : "Show \(items.count) to do")

            if isOpen {
                VStack(spacing: 0) {
                    ForEach(items) { item in
                        Divider().padding(.leading, item.id == items.first?.id ? 0 : 45)
                        TaskRow(item: item, actions: actions)
                    }
                }
                .padding(.bottom, 4)
                .transition(.opacity.animation(.easeOut(duration: 0.12)))
            }
        }
        .clipShape(.rect(cornerRadius: 14, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous).strokeBorder(Color(.separator)))
    }

    private var urgent: (String, Color)? {
        for item in items {
            switch item {
            case .chase(let c): return (c.overdueDays > 0 ? "\(c.overdueDays)d overdue" : "Chase today", c.overdueDays > 0 ? .red : .orange)
            case .draft(let d) where d.state == .failed: return ("Draft failed", .red)
            case .draft(let d) where d.state == .ready: return ("CV ready", .brandTint)
            default: continue
            }
        }
        return nil
    }
}

private struct TaskRow: View {
    let item: DueItem
    let actions: DueActions
    @Environment(TriagePrototypeStore.self) private var store
    @State private var struck = false

    var body: some View {
        HStack(alignment: .center, spacing: 0) {
            checkbox
            Menu {
                menuItems
            } label: {
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline)
                        .strikethrough(struck)
                        .foregroundStyle(struck ? .secondary : .primary)
                        .multilineTextAlignment(.leading)
                    meta
                }
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                .padding(.vertical, 8)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
        }
        .padding(.leading, 1)
        .padding(.trailing, 14)
    }

    private var title: String {
        switch item {
        case .chase: "Chase the application"
        case .draft(let d): d.state == .ready ? "Apply with tailored CV" : "Tailored CV"
        case .task(let t): t.type
        }
    }

    @ViewBuilder
    private var meta: some View {
        switch item {
        case .chase(let c):
            HStack(spacing: 6) {
                DotLabel(text: c.overdueDays > 0 ? "Overdue \(c.overdueDays)d" : "Today", hue: c.overdueDays > 0 ? .red : .orange)
                Text(c.chaseBy.shortGB).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
            }
        case .draft(let d):
            switch d.state {
            case .ready:
                HStack(spacing: 6) {
                    DotLabel(text: "CV ready", hue: .brandTint)
                    Text(d.template).font(.caption).foregroundStyle(.secondary)
                }
            case .failed: DotLabel(text: "Couldn't draft", hue: .red)
            case .pending: Text("Drafting…").font(.caption).foregroundStyle(.secondary)
            }
        case .task: EmptyView()
        }
    }

    @ViewBuilder
    private var checkbox: some View {
        Group {
            switch item {
            case .draft(let d) where d.state == .pending:
                ProgressView().controlSize(.small)
            case .draft(let d) where d.state == .failed:
                Button("Retry draft", systemImage: "arrow.clockwise") { withAnimation(.snappy) { store.retry(d) } }
                    .labelStyle(.iconOnly)
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(.red)
                    .frame(width: 18, height: 18)
                    .overlay(RoundedRectangle(cornerRadius: 5, style: .continuous).strokeBorder(.red.opacity(0.5), lineWidth: 1.5))
                    .frame(width: 44, height: 44)
                    .contentShape(.rect)
            default:
                Button(action: complete) {
                    RoundedRectangle(cornerRadius: 5, style: .continuous)
                        .strokeBorder(struck ? Color.brandTint : Color(.tertiaryLabel), lineWidth: 1.5)
                        .background(struck ? Color.brandTint : .clear, in: .rect(cornerRadius: 5, style: .continuous))
                        .overlay {
                            if struck { Image(systemName: "checkmark").font(.caption2.weight(.bold)).foregroundStyle(.white) }
                        }
                        .frame(width: 18, height: 18)
                        .frame(width: 44, height: 44)
                        .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .sensoryFeedback(.success, trigger: struck) { _, new in new }
                .accessibilityLabel("Complete: \(title)")
            }
        }
        .frame(width: 44, height: 44)
    }

    @ViewBuilder
    private var menuItems: some View {
        switch item {
        case .chase(let c):
            Button("Mark chased", systemImage: "checkmark") { complete() }
            Menu("Snooze", systemImage: "clock") {
                ForEach([(2, "2 days"), (7, "1 week"), (14, "2 weeks")], id: \.0) { days, label in
                    Button(label) { withAnimation(.snappy) { store.snooze(c, days: days) } }
                }
            }
            Button("Log contact", systemImage: "person.badge.plus") { actions.logContact(c.job) }
        case .draft(let d):
            if d.state == .ready {
                Button("Mark applied", systemImage: "paperplane") { complete() }
                Button("View PDF", systemImage: "doc.text") { actions.openPDF(d) }
            } else if d.state == .failed {
                Button("Retry draft", systemImage: "arrow.clockwise") { store.retry(d) }
            }
        case .task:
            Button("Mark done", systemImage: "checkmark") { complete() }
        }
        Divider()
        Button("Open job", systemImage: "arrow.up.right") { actions.openJob(item.job) }
    }

    private func complete() {
        struck = true
        Task {
            try? await Task.sleep(for: .milliseconds(500))
            withAnimation(.snappy(duration: 0.25)) {
                switch item {
                case .chase(let c): store.completeChase(c)
                case .task(let t): store.completeTask(t)
                case .draft(let d): store.markApplied(d)
                }
            }
        }
    }
}

struct DotLabel: View {
    let text: String
    let hue: Color

    var body: some View {
        HStack(spacing: 4) {
            Circle().fill(hue).frame(width: 6, height: 6)
            Text(text).font(.caption.weight(.medium).monospacedDigit()).foregroundStyle(.primary)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 2)
        .background(hue.opacity(0.1), in: .rect(cornerRadius: 5, style: .continuous))
    }
}
#endif
