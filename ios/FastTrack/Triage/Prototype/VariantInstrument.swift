#if DEBUG
import SwiftUI

struct VariantInstrument: View {
    let store: TriagePrototypeStore
    @State private var progress: CGFloat = 0
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var behind: Int { min(max(store.queue.count - 1, 0), 3) }
    private var detent: Int { Int((progress * 4).rounded(.towardZero)) }

    var body: some View {
        VStack(spacing: 18) {
            ZStack(alignment: .bottom) {
                DeckEdges(colours: Array(repeating: .clear, count: behind), radius: 16)
                TriageCardHost(store: store, progress: $progress) { job, isExpanded, toggle in
                    TactileCard(job: job, isExpanded: isExpanded, toggle: toggle, radius: 16) {
                        ScoreGauge(job: job)
                    }
                }
            }
            .padding(.horizontal)
            .padding(.bottom, CGFloat(behind) * 7)
            if let job = store.current {
                rocker(job)
                    .padding(.horizontal, 28)
            }
        }
        .padding(.bottom, 8)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                Text("\(store.queue.count) in queue")
                    .font(.subheadline.weight(.medium).monospacedDigit())
                    .contentTransition(.numericText(countsDown: true))
                    .animation(.snappy, value: store.queue.count)
            }
        }
        .sensoryFeedback(.selection, trigger: detent) { _, new in new != 0 }
    }

    private func rocker(_ job: PrototypeJob) -> some View {
        let right = max(min(progress, 1), 0)
        let left = max(min(-progress, 1), 0)
        return HStack(spacing: 0) {
            Button { act(.dismiss, job) } label: {
                Label("Dismiss", systemImage: "xmark")
                    .frame(maxWidth: .infinity, minHeight: 56)
                    .background(Color(.systemGray).opacity(left * 0.35))
                    .contentShape(.rect)
            }
            .foregroundStyle(.secondary)
            Divider().frame(height: 28)
            TriageMoreMenu(job: job, store: store) {
                Image(systemName: "ellipsis")
                    .frame(width: 56, height: 56)
                    .contentShape(.rect)
                    .accessibilityLabel("More")
            }
            .foregroundStyle(.secondary)
            Divider().frame(height: 28)
            Button { act(.shortlist, job) } label: {
                Label("Shortlist", systemImage: "checkmark")
                    .fontWeight(.semibold)
                    .frame(maxWidth: .infinity, minHeight: 56)
                    .background(Color.brandTint.opacity(0.12 + right * 0.3))
                    .contentShape(.rect)
            }
            .foregroundStyle(Color.brandTint)
        }
        .buttonStyle(.plain)
        .font(.body)
        .labelStyle(.titleAndIcon)
        .lineLimit(1)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(.capsule)
        .overlay(Capsule().strokeBorder(Color(.separator)))
        .rotation3DEffect(.degrees(reduceMotion ? 0 : Double(progress.clamped(to: -1...1)) * 10), axis: (0, 1, 0), perspective: 0.6)
        .animation(.interactiveSpring, value: progress)
    }

    private func act(_ action: TriagePrototypeStore.Action, _ job: PrototypeJob) {
        withAnimation(.snappy) { store.act(action, on: job) }
    }
}

private struct ScoreGauge: View {
    let job: PrototypeJob
    private let cuts: [(Int, String)] = [(45, "Fair"), (65, "Good"), (80, "Great")]
    @State private var shown = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                Text(job.band.rawValue)
                    .font(.headline)
                    .foregroundStyle(job.band.colour)
                Text(job.score, format: .number)
                    .font(.headline.monospacedDigit())
                Spacer()
                Text("\(job.company) · \(job.age)")
                    .font(.footnote.monospacedDigit())
                    .foregroundStyle(.secondary)
            }
            GeometryReader { proxy in
                let w = proxy.size.width
                ZStack(alignment: .topLeading) {
                    ForEach(0...20, id: \.self) { i in
                        let isCut = cuts.contains { $0.0 == i * 5 }
                        Rectangle()
                            .fill(isCut ? Color.primary.opacity(0.6) : Color(.separator))
                            .frame(width: 1, height: isCut ? 14 : (i % 2 == 0 ? 9 : 6))
                            .offset(x: w * CGFloat(i) / 20)
                    }
                    ForEach(cuts, id: \.0) { cut, label in
                        Text(label)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .fixedSize()
                            .offset(x: w * CGFloat(cut) / 100 + 3, y: 14)
                    }
                    Capsule()
                        .fill(job.band.colour)
                        .frame(width: 3, height: 22)
                        .offset(x: w * CGFloat(shown ? job.score : 0) / 100 - 1, y: -4)
                }
            }
            .frame(height: 30)
        }
        .padding(.horizontal, 22)
        .padding(.top, 20)
        .padding(.bottom, 8)
        .onAppear {
            withAnimation(reduceMotion ? nil : .spring(duration: 0.6, bounce: 0.2).delay(0.1)) { shown = true }
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(job.band.rawValue), score \(job.score) of 100")
    }
}

private extension CGFloat {
    func clamped(to range: ClosedRange<CGFloat>) -> CGFloat { Swift.min(Swift.max(self, range.lowerBound), range.upperBound) }
}
#endif
