#if DEBUG
import SwiftUI

struct VariantSignal: View {
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
            VStack(alignment: .leading, spacing: 14) {
                HStack(alignment: .firstTextBaseline) {
                    Text(job.company).font(.subheadline.weight(.semibold)).foregroundStyle(.secondary)
                    Spacer()
                    BandChip(band: job.band)
                    Text(job.score, format: .number).font(.subheadline.monospacedDigit()).foregroundStyle(.secondary)
                }
                Text(job.title).font(.title2.weight(.semibold)).lineLimit(isExpanded ? nil : 3)
                FitStrip(job: job)
            }
            .padding(20)
            Divider()
            if isExpanded {
                ScrollView {
                    JobDetailBody(job: job).padding(20)
                }
            } else {
                Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 14) {
                    fact("Salary", job.salary ?? "—", mono: job.salary != nil, faint: job.salary == nil)
                    fact("Location", job.location)
                    fact("Mode", job.arrangement)
                    fact("Posted", "\(job.age) ago", mono: true)
                    if let gate = job.gate {
                        GridRow {
                            Text("Gate").font(.subheadline).foregroundStyle(.secondary)
                            Label(gate.label, systemImage: gate.effect.symbol)
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(gate.effect.colour)
                        }
                    }
                }
                .padding(20)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            Divider()
            Button(action: toggle) {
                HStack {
                    Text(isExpanded ? "Show summary" : "Details and full breakdown")
                    Spacer()
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                }
                .font(.subheadline.weight(.medium))
                .padding(.horizontal, 20)
                .frame(minHeight: 50)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .foregroundStyle(.tint)
        }
        .frame(maxHeight: .infinity)
        .cardSurface()
    }

    private func fact(_ label: String, _ value: String, mono: Bool = false, faint: Bool = false) -> some View {
        GridRow {
            Text(label).font(.subheadline).foregroundStyle(.secondary)
            Text(value)
                .font(mono ? .body.monospacedDigit() : .body)
                .foregroundStyle(faint ? .secondary : .primary)
        }
    }
}

private struct FitStrip: View {
    let job: PrototypeJob
    @State private var shown = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var rows: [PrototypeJob.Row] {
        let rank: [PrototypeJob.Effect: Int] = [.gated: 0, .misses: 1, .unknown: 2, .level: 3, .meets: 4, .boost: 5]
        return job.rows.sorted { rank[$0.effect, default: 9] < rank[$1.effect, default: 9] }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 3) {
                ForEach(Array(rows.enumerated()), id: \.element) { index, row in
                    Capsule()
                        .fill(fill(row.effect))
                        .frame(height: 8)
                        .scaleEffect(x: shown ? 1 : 0, anchor: .leading)
                        .animation(reduceMotion ? nil : .spring(duration: 0.4, bounce: 0.1).delay(Double(index) * 0.05), value: shown)
                }
            }
            Text(summary)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .onAppear { shown = true }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(summary)
    }

    private func fill(_ effect: PrototypeJob.Effect) -> Color {
        switch effect {
        case .gated: Color(hex: 0xdc2626)
        case .misses: Color(hex: 0xdc2626).opacity(0.45)
        case .meets: Color(hex: 0x059669)
        case .boost: Color(hex: 0x059669)
        case .level, .unknown: Color(.systemGray4)
        }
    }

    private var summary: String {
        let count = { (e: PrototypeJob.Effect) in job.rows.filter { $0.effect == e }.count }
        return [
            count(.meets) + count(.boost) > 0 ? "\(count(.meets) + count(.boost)) met" : nil,
            count(.misses) > 0 ? "\(count(.misses)) missed" : nil,
            count(.unknown) > 0 ? "\(count(.unknown)) unknown" : nil,
        ].compactMap { $0 }.joined(separator: " · ")
    }
}
#endif
