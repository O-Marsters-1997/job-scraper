#if DEBUG
import SwiftUI

struct DuePasses: View {
    let actions: DueActions
    @Environment(TriagePrototypeStore.self) private var store
    @State private var fanned: Set<String> = []

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 32) {
                Text(store.dueSummary)
                    .font(.title3.weight(.medium))
                    .foregroundStyle(.secondary)
                    .contentTransition(.numericText())
                stack("Chases", store.dueChases.map(DueItem.chase))
                stack("CV drafts", store.drafts.map(DueItem.draft))
                stack("Open tasks", store.tasks.map(DueItem.task))
            }
            .padding(.horizontal)
            .padding(.bottom, 24)
            .animation(.spring(duration: 0.45, bounce: 0.18), value: store.dueItems)
            .animation(.spring(duration: 0.45, bounce: 0.18), value: fanned)
        }
        .background(Color(.systemGroupedBackground))
    }

    @ViewBuilder
    private func stack(_ title: String, _ items: [DueItem]) -> some View {
        if let first = items.first {
            let isFanned = fanned.contains(title) || items.count == 1
            VStack(alignment: .leading, spacing: 12) {
                Button {
                    if fanned.contains(title) { fanned.remove(title) } else { fanned.insert(title) }
                } label: {
                    HStack(alignment: .firstTextBaseline) {
                        Text(title).font(.title3.weight(.bold))
                        Text("\(items.count)")
                            .font(.title3.monospacedDigit())
                            .foregroundStyle(.secondary)
                        Spacer()
                        if items.count > 1 {
                            Text(isFanned ? "Stack" : "Show all")
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(.tint)
                        }
                    }
                    .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .disabled(items.count == 1)
                if isFanned {
                    ForEach(items) { item in
                        DuePass(item: item, actions: actions)
                            .transition(.asymmetric(insertion: .opacity.combined(with: .offset(y: -20)), removal: .opacity.combined(with: .scale(scale: 0.95))))
                    }
                } else {
                    ZStack(alignment: .bottom) {
                        DeckEdges(colours: items.dropFirst().prefix(3).map { $0.accent.opacity(0.12) }, radius: 22)
                        DuePass(item: first, actions: actions)
                    }
                    .padding(.bottom, CGFloat(min(items.count - 1, 3)) * 7)
                    .onTapGesture { fanned.insert(title) }
                }
            }
        }
    }
}

private struct DuePass: View {
    let item: DueItem
    let actions: DueActions

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(spacing: 12) {
                Monogram(job: item.job, size: 40)
                VStack(alignment: .leading, spacing: 1) {
                    Text(item.kind).font(.footnote.weight(.semibold)).foregroundStyle(.secondary)
                    Text(item.status).font(.subheadline.weight(.semibold).monospacedDigit()).foregroundStyle(item.accent)
                }
                Spacer()
                if case .chase(let chase) = item {
                    Text(chase.chaseBy.shortGB).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
                }
            }
            NavigationLink(value: item.job) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.job.title).font(.headline).multilineTextAlignment(.leading)
                    Text(item.job.company).font(.subheadline).foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            DueItemActions(item: item, actions: actions, style: .compact)
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 22, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 22, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
    }
}
#endif
