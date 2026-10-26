#if DEBUG
import SwiftUI

struct VariantEvidence: View {
    let store: TriagePrototypeStore

    var body: some View {
        VStack(spacing: 12) {
            TriageCardHost(store: store) { job, isExpanded, toggle in
                card(job, isExpanded: isExpanded, toggle: toggle)
            }
            .padding(.horizontal)
            if let job = store.current {
                TriageActionBar(job: job, store: store)
            }
        }
        .padding(.bottom, 8)
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func card(_ job: PrototypeJob, isExpanded: Bool, toggle: @escaping () -> Void) -> some View {
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
            if isExpanded {
                ScrollView {
                    VStack(alignment: .leading, spacing: 24) {
                        titleBlock(job, chevron: "chevron.down", toggle: toggle)
                        Divider()
                        JobDetailBody(job: job)
                    }
                    .padding(20)
                }
            } else {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(job.rows.filter { $0.effect != .gated }, id: \.self) { BreakdownRowView(row: $0) }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(20)
                Divider()
                titleBlock(job, chevron: "chevron.up", toggle: toggle)
                    .padding(20)
            }
        }
        .frame(maxHeight: .infinity)
        .cardSurface()
        .accessibilityElement(children: isExpanded ? .contain : .ignore)
        .accessibilityLabel(job.accessibilitySummary)
    }

    private func titleBlock(_ job: PrototypeJob, chevron: String, toggle: @escaping () -> Void) -> some View {
        Button(action: toggle) {
            HStack(alignment: .bottom) {
                MetaLines(job: job, titleFont: .headline)
                Image(systemName: chevron).foregroundStyle(.secondary)
            }
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
    }
}
#endif
