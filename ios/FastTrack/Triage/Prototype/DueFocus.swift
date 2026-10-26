#if DEBUG
import SwiftUI

struct DueFocus: View {
    let actions: DueActions
    @Environment(TriagePrototypeStore.self) private var store
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        let items = store.dueItems
        ScrollView {
            VStack(alignment: .leading, spacing: 28) {
                if let next = items.first {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack(alignment: .firstTextBaseline) {
                            Text("Next up").font(.title3.weight(.bold))
                            Spacer()
                            Text("1 of \(items.count)")
                                .font(.subheadline.monospacedDigit())
                                .foregroundStyle(.secondary)
                                .contentTransition(.numericText(countsDown: true))
                        }
                        ZStack(alignment: .bottom) {
                            DeckEdges(colours: items.dropFirst().prefix(3).map { $0.accent.opacity(0.12) }, radius: 24)
                            hero(next)
                                .id(next.id)
                                .transition(.asymmetric(
                                    insertion: reduceMotion ? .opacity : .scale(scale: 0.94).combined(with: .opacity),
                                    removal: reduceMotion ? .opacity : .offset(x: 500).combined(with: .opacity)))
                        }
                        .padding(.bottom, CGFloat(min(items.count - 1, 3)) * 7)
                    }
                }
                if items.count > 1 {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Then").font(.title3.weight(.bold)).padding(.bottom, 6)
                        ForEach(items.dropFirst()) { item in
                            compactRow(item)
                            if item != items.last { Divider().padding(.leading, 44) }
                        }
                    }
                }
            }
            .padding(.horizontal)
            .padding(.bottom, 24)
            .animation(.spring(duration: 0.45, bounce: 0.15), value: items)
        }
        .background(Color(.systemGroupedBackground))
    }

    private func hero(_ item: DueItem) -> some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack(spacing: 14) {
                Monogram(job: item.job, size: 56)
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.kind).font(.subheadline.weight(.semibold)).foregroundStyle(.secondary)
                    Text(item.status)
                        .font(.title3.weight(.bold).monospacedDigit())
                        .foregroundStyle(item.accent)
                }
            }
            NavigationLink(value: item.job) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.job.title).font(.title2.weight(.semibold)).multilineTextAlignment(.leading)
                    Text("\(item.job.company) · \(item.job.location)").font(.body).foregroundStyle(.secondary)
                    if let salary = item.job.salary {
                        Text(salary).font(.subheadline.monospacedDigit())
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            DueItemActions(item: item, actions: actions, style: .prominent)
        }
        .padding(22)
        .background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 24, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 24, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
    }

    private func compactRow(_ item: DueItem) -> some View {
        HStack(spacing: 12) {
            if case .task(let task) = item {
                TaskCheck(task: task).frame(width: 32)
            } else {
                Monogram(job: item.job, size: 32)
            }
            NavigationLink(value: item.job) {
                HStack {
                    VStack(alignment: .leading, spacing: 1) {
                        Text(item.job.title).font(.subheadline.weight(.semibold)).lineLimit(1)
                        Text(item.kind == "Chase" || item.kind == "CV draft" ? item.status : "\(item.kind) · \(item.job.company)")
                            .font(.footnote.monospacedDigit())
                            .foregroundStyle(item.kind == "Chase" || item.kind == "CV draft" ? item.accent : .secondary)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 8)
                    Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(.tertiary)
                }
                .frame(minHeight: 44)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
        }
    }
}
#endif
