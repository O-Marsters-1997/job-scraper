#if DEBUG
import SwiftUI

struct VariantWallet: View {
    let store: TriagePrototypeStore
    @State private var progress: CGFloat = 0
    @State private var expanded = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var upcoming: [PrototypeJob] { Array(store.queue.dropFirst().prefix(3)) }

    var body: some View {
        VStack(spacing: 22) {
            ZStack(alignment: .bottom) {
                DeckEdges(colours: upcoming.map { $0.band.colour.opacity(0.22) }, radius: 22)
                    .opacity(expanded ? 0 : 1)
                TriageCardHost(store: store, showsStamp: false, progress: $progress) { job, isExpanded, toggle in
                    TactileCard(job: job, isExpanded: isExpanded, toggle: toggle, radius: 22) {
                        header(job)
                    }
                    .onChange(of: isExpanded, initial: true) { expanded = isExpanded }
                    .overlay { decisionWash }
                }
            }
            .padding(.horizontal)
            .padding(.bottom, expanded ? 0 : CGFloat(upcoming.count) * 7)
            if let job = store.current {
                buttons(job)
                    .scaleEffect(expanded && !reduceMotion ? 0.9 : 1)
                    .opacity(expanded ? 0.6 : 1)
            }
        }
        .animation(.spring(duration: 0.4, bounce: 0.15), value: expanded)
        .padding(.bottom, 8)
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func header(_ job: PrototypeJob) -> some View {
        HStack(spacing: 14) {
            Text(String(job.company.prefix(1)))
                .font(.title2.weight(.bold))
                .fontDesign(.rounded)
                .foregroundStyle(job.band.colour)
                .frame(width: 48, height: 48)
                .background(job.band.colour.opacity(0.14), in: .rect(cornerRadius: 12, style: .continuous))
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 2) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text(job.band.rawValue)
                        .font(.headline)
                        .foregroundStyle(job.band.colour)
                    Text(job.score, format: .number)
                        .font(.headline.monospacedDigit())
                        .foregroundStyle(.secondary)
                }
                Text("\(job.source.capitalized) · \(job.age) ago")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            if let gate = job.gate {
                Image(systemName: gate.effect.symbol)
                    .font(.title3)
                    .foregroundStyle(gate.effect.colour)
                    .accessibilityLabel("Gate: \(gate.label)")
            }
        }
        .padding(.horizontal, 22)
        .padding(.top, 20)
        .padding(.bottom, 4)
    }

    private var decisionWash: some View {
        let amount = min(abs(progress), 1)
        return RoundedRectangle(cornerRadius: 22, style: .continuous)
            .fill(progress > 0 ? Color.brandTint : Color(.systemGray))
            .opacity(amount * 0.12)
            .allowsHitTesting(false)
    }

    private func buttons(_ job: PrototypeJob) -> some View {
        let right = max(min(progress, 1), 0)
        let left = max(min(-progress, 1), 0)
        return HStack(spacing: 28) {
            Button { act(.dismiss, job) } label: {
                Image(systemName: "xmark")
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(left > 0.99 ? Color.white : Color.secondary)
                    .frame(width: 64, height: 64)
                    .background(Color(.systemGray).opacity(left), in: .circle)
                    .background(Color(.secondarySystemGroupedBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator)))
                    .scaleEffect(1 + left * 0.15)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Dismiss")
            TriageMoreMenu(job: job, store: store) {
                Image(systemName: "ellipsis")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .frame(width: 48, height: 48)
                    .background(Color(.secondarySystemGroupedBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator)))
                    .accessibilityLabel("More")
            }
            .tint(.secondary)
            .opacity(Double(1 - max(left, right) * 0.6))
            Button { act(.shortlist, job) } label: {
                Image(systemName: "checkmark")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.white)
                    .frame(width: 64, height: 64)
                    .background(Color.brandTint, in: .circle)
                    .scaleEffect(1 + right * 0.15)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Shortlist")
        }
        .animation(.interactiveSpring, value: progress)
    }

    private func act(_ action: TriagePrototypeStore.Action, _ job: PrototypeJob) {
        withAnimation(.snappy) { store.act(action, on: job) }
    }
}
#endif
