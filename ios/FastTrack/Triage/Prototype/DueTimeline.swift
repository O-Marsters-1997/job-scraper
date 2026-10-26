#if DEBUG
import SwiftUI

struct DueTimeline: View {
    let actions: DueActions
    @Environment(TriagePrototypeStore.self) private var store

    private var groups: [(String, [DueItem])] {
        let overdue = store.dueChases.filter { $0.overdueDays > 0 }.map(DueItem.chase)
        let today = store.dueChases.filter { $0.overdueDays == 0 }.map(DueItem.chase)
        let ready = store.drafts.filter { $0.state == .ready }.map(DueItem.draft)
        let drafting = store.drafts.filter { $0.state != .ready }.map(DueItem.draft)
        let anytime = store.tasks.map(DueItem.task)
        return [("Overdue", overdue), ("Today", today), ("Ready to send", ready), ("Drafting", drafting), ("Anytime", anytime)]
            .filter { !$0.1.isEmpty }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(Date.now.formatted(.dateTime.weekday(.wide).day().month(.wide).locale(Locale(identifier: "en_GB"))))
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(.secondary)
                    Text(store.dueSummary)
                        .font(.title2.weight(.semibold))
                        .contentTransition(.numericText())
                }
                .padding(.bottom, 28)
                ForEach(groups, id: \.0) { title, items in
                    group(title, items)
                }
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 24)
            .animation(.spring(duration: 0.4, bounce: 0.12), value: store.dueItems)
        }
        .background(Color(.systemGroupedBackground))
    }

    private func group(_ title: String, _ items: [DueItem]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 14) {
                Circle()
                    .fill(title == "Overdue" ? Color.red : Color(.tertiaryLabel))
                    .frame(width: 10, height: 10)
                    .frame(width: 40)
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(title == "Overdue" ? Color.red : Color.secondary)
            }
            .padding(.vertical, 8)
            ForEach(items) { item in
                row(item)
                    .transition(.asymmetric(insertion: .opacity, removal: .opacity.combined(with: .offset(x: -40))))
            }
        }
        .padding(.bottom, 12)
    }

    private func row(_ item: DueItem) -> some View {
        HStack(alignment: .top, spacing: 14) {
            ZStack(alignment: .top) {
                Rectangle()
                    .fill(Color(.separator))
                    .frame(width: 2)
                    .frame(maxHeight: .infinity)
                Group {
                    if case .task(let task) = item {
                        TaskCheck(task: task)
                            .background(Color(.systemGroupedBackground), in: .circle)
                    } else {
                        Monogram(job: item.job, size: 40)
                            .background(Color(.systemGroupedBackground), in: .rect(cornerRadius: 10))
                    }
                }
            }
            .frame(width: 40)
            VStack(alignment: .leading, spacing: 10) {
                NavigationLink(value: item.job) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(item.job.title).font(.headline).multilineTextAlignment(.leading)
                        HStack(spacing: 6) {
                            Text(item.job.company).foregroundStyle(.secondary)
                            Text("·").foregroundStyle(.tertiary)
                            Text(item.kind == "Chase" || item.kind == "CV draft" ? item.status : item.kind)
                                .foregroundStyle(item.kind == "Chase" || item.kind == "CV draft" ? item.accent : .secondary)
                                .monospacedDigit()
                        }
                        .font(.subheadline)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .contentShape(.rect)
                }
                .buttonStyle(.plain)
                if case .task = item {} else {
                    DueItemActions(item: item, actions: actions, style: .compact)
                }
            }
            .padding(.top, 2)
            .padding(.bottom, 22)
        }
        .fixedSize(horizontal: false, vertical: true)
    }
}
#endif
