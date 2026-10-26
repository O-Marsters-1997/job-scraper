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
                DeckEdges(count: upcoming.count)
                    .opacity(expanded ? 0 : 1)
                TriageCardHost(store: store, showsStamp: false, progress: $progress) { job, isExpanded, toggle in
                    TactileCard(job: job, isExpanded: isExpanded, toggle: toggle) {
                        BandRow(job: job)
                            .padding(.horizontal, 16)
                            .padding(.top, 16)
                    }
                    .onChange(of: isExpanded, initial: true) { expanded = isExpanded }
                    .overlay { decisionWash }
                }
            }
            .padding(.horizontal)
            .padding(.bottom, expanded ? 0 : CGFloat(upcoming.count) * 6)
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

    private var decisionWash: some View {
        let amount = min(abs(progress), 1)
        return RoundedRectangle(cornerRadius: 14, style: .continuous)
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
                    .background(Color(.systemBackground), in: .circle)
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
                    .background(Color(.systemBackground), in: .circle)
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
