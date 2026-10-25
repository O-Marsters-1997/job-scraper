#if DEBUG
import SwiftUI

struct VariantPager: View {
    let store: TriagePrototypeStore
    @State private var position: PrototypeJob.ID?

    var body: some View {
        ScrollView(.vertical) {
            LazyVStack(spacing: 0) {
                ForEach(store.queue) { job in
                    page(job)
                        .containerRelativeFrame(.vertical)
                        .id(job.id)
                }
            }
            .scrollTargetLayout()
        }
        .scrollTargetBehavior(.paging)
        .scrollPosition(id: $position)
        .scrollIndicators(.hidden)
        .overlay { if store.queue.isEmpty { QueueClearView(store: store) } }
        .onChange(of: position) { old, new in
            guard let old, let new,
                  let oldIndex = store.queue.firstIndex(where: { $0.id == old }),
                  let newIndex = store.queue.firstIndex(where: { $0.id == new }),
                  newIndex > oldIndex else { return }
            store.markSeen(store.queue[oldIndex])
        }
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: PrototypeJob.self) { PrototypeJobDetail(job: $0, store: store) }
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Text("\(store.queue.count) left")
                    .font(.subheadline.monospacedDigit())
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func page(_ job: PrototypeJob) -> some View {
        VStack(spacing: 16) {
            NavigationLink(value: job) {
                VStack(alignment: .leading, spacing: 16) {
                    BandScoreLine(job: job)
                    MetaLines(job: job)
                    Divider()
                    VStack(alignment: .leading, spacing: 10) {
                        ForEach(job.topRows, id: \.self) { BreakdownRowView(row: $0) }
                    }
                    Spacer(minLength: 0)
                    HStack {
                        Spacer()
                        Text("Details").font(.subheadline)
                        Image(systemName: "chevron.right").font(.footnote.weight(.semibold))
                    }
                    .foregroundStyle(.secondary)
                }
                .padding(20)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .cardSurface()
            }
            .buttonStyle(.plain)
            .swipeToDecide { dir in act(dir == .right ? .shortlist : .dismiss, on: job) }
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(job.accessibilitySummary)
            .accessibilityAction(named: "Shortlist") { act(.shortlist, on: job) }
            .accessibilityAction(named: "Dismiss") { act(.dismiss, on: job) }
            TriageActionBar(job: job, store: store)
        }
        .padding(.horizontal)
        .padding(.vertical, 12)
    }

    private func act(_ action: TriagePrototypeStore.Action, on job: PrototypeJob) {
        withAnimation { store.act(action, on: job) }
    }
}
#endif
