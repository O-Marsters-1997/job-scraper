#if DEBUG
import SwiftUI

struct VariantEvidence: View {
    let store: TriagePrototypeStore
    @State private var detailJob: PrototypeJob?

    var body: some View {
        VStack(spacing: 12) {
            ProgressView(value: Double(PrototypeJob.fixtures.count - store.queue.count),
                         total: Double(PrototypeJob.fixtures.count))
                .tint(.brandTint)
                .padding(.horizontal)
            if let job = store.current {
                card(job)
                    .id(job.id)
                    .transition(.asymmetric(insertion: .move(edge: .bottom).combined(with: .opacity), removal: .opacity))
                    .padding(.horizontal)
                TriageActionBar(job: job, store: store) { detailJob = job }
            } else {
                QueueClearView(store: store)
            }
        }
        .padding(.bottom, 8)
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if let job = store.current {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Skip") { withAnimation { store.act(.skip, on: job) } }
                }
            }
        }
        .sheet(item: $detailJob) { job in
            NavigationStack { PrototypeJobDetail(job: job, store: store) }
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.visible)
        }
    }

    private func card(_ job: PrototypeJob) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .firstTextBaseline) {
                Text(job.band.rawValue)
                    .font(.largeTitle.weight(.bold))
                    .foregroundStyle(job.band.colour)
                Text(job.score, format: .number)
                    .font(.title3.monospacedDigit())
                    .foregroundStyle(.secondary)
                Spacer()
                Text(job.age).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
            }
            .padding(20)
            if let gate = job.gate {
                Label("Gate: \(gate.label)", systemImage: gate.effect.symbol)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(gate.effect.colour)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 20)
                    .padding(.vertical, 10)
                    .background(gate.effect.colour.opacity(0.1))
            }
            ScrollView {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(job.rows.filter { $0.effect != .gated }, id: \.self) { BreakdownRowView(row: $0) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(20)
            }
            .scrollBounceBehavior(.basedOnSize)
            Divider()
            Button { detailJob = job } label: {
                HStack(alignment: .bottom) {
                    MetaLines(job: job, titleFont: .headline)
                    Image(systemName: "chevron.up").foregroundStyle(.secondary)
                }
                .padding(20)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
        }
        .frame(maxHeight: .infinity)
        .cardSurface()
        .swipeToDecide { dir in
            withAnimation { store.act(dir == .right ? .shortlist : .dismiss, on: job) }
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(job.accessibilitySummary)
        .accessibilityAction(named: "Shortlist") { store.act(.shortlist, on: job) }
        .accessibilityAction(named: "Dismiss") { store.act(.dismiss, on: job) }
        .accessibilityAction(named: "Details") { detailJob = job }
    }
}
#endif
