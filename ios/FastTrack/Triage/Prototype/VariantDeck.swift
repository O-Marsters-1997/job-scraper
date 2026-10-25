#if DEBUG
import SwiftUI

struct VariantDeck: View {
    let store: TriagePrototypeStore
    @State private var flipped: PrototypeJob.ID?
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        VStack(spacing: 16) {
            ZStack {
                if store.queue.isEmpty {
                    QueueClearView(store: store)
                }
                ForEach(Array(store.queue.prefix(3).enumerated().reversed()), id: \.element.id) { depth, job in
                    card(job, depth: depth)
                }
            }
            .padding(.horizontal)
            if let job = store.current {
                TriageActionBar(job: job, store: store) { withAnimation { flipped = job.id } }
            }
        }
        .padding(.bottom, 8)
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
    }

    @ViewBuilder
    private func card(_ job: PrototypeJob, depth: Int) -> some View {
        let isTop = depth == 0
        let isFlipped = flipped == job.id
        ZStack {
            DeckFront(job: job)
                .opacity(isFlipped ? 0 : 1)
            DeckBack(job: job) { withAnimation(.snappy) { flipped = nil } }
                .opacity(isFlipped ? 1 : 0)
                .rotation3DEffect(.degrees(reduceMotion ? 0 : 180), axis: (0, 1, 0))
        }
        .rotation3DEffect(.degrees(isFlipped && !reduceMotion ? 180 : 0), axis: (0, 1, 0))
        .cardSurface()
        .scaleEffect(1 - CGFloat(depth) * 0.04)
        .offset(y: CGFloat(depth) * 14)
        .allowsHitTesting(isTop)
        .accessibilityHidden(!isTop)
        .swipeToDecide(allowsUp: true, isEnabled: isTop && !isFlipped) { dir in
            store.act(dir == .right ? .shortlist : dir == .left ? .dismiss : .skip, on: job)
        }
        .onTapGesture { if isTop && !isFlipped { withAnimation(.snappy) { flipped = job.id } } }
        .transition(.asymmetric(insertion: .scale(scale: 0.96).combined(with: .opacity), removal: removal))
        .accessibilityElement(children: isFlipped ? .contain : .ignore)
        .accessibilityLabel(job.accessibilitySummary)
        .accessibilityAction(named: "Shortlist") { store.act(.shortlist, on: job) }
        .accessibilityAction(named: "Dismiss") { store.act(.dismiss, on: job) }
        .accessibilityAction(named: "Skip") { store.act(.skip, on: job) }
        .accessibilityAction(named: "Details") { flipped = job.id }
    }

    private var removal: AnyTransition {
        guard !reduceMotion, let last = store.lastRemoved else { return .opacity }
        switch last.action {
        case .shortlist, .shortlistDraft: return .offset(x: 600).combined(with: .opacity)
        case .skip: return .offset(y: -900).combined(with: .opacity)
        default: return .offset(x: -600).combined(with: .opacity)
        }
    }
}

private struct DeckFront: View {
    let job: PrototypeJob

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            BandScoreLine(job: job)
            MetaLines(job: job)
            Divider()
            VStack(alignment: .leading, spacing: 10) {
                ForEach(job.topRows, id: \.self) { BreakdownRowView(row: $0) }
            }
            Spacer(minLength: 0)
            HStack {
                Text(job.source.uppercased())
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                Spacer()
                Label("Tap for details", systemImage: "hand.tap")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(20)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }
}

private struct DeckBack: View {
    let job: PrototypeJob
    let onClose: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(job.title).font(.headline).lineLimit(1)
                Spacer()
                Button("Back to summary", systemImage: "arrow.uturn.backward", action: onClose)
                    .labelStyle(.iconOnly)
            }
            .padding(20)
            Divider()
            List {
                Section("Description") { Text(job.description) }
                GroupedBreakdown(job: job)
            }
            .listStyle(.plain)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .clipShape(.rect(cornerRadius: 16, style: .continuous))
    }
}
#endif
