#if DEBUG
import SwiftUI

struct TriageCardHost<Card: View>: View {
    let store: TriagePrototypeStore
    var showsStamp = true
    @Binding var progress: CGFloat
    @ViewBuilder let card: (_ job: PrototypeJob, _ isExpanded: Bool, _ toggle: @escaping () -> Void) -> Card

    @State private var expandedID: PrototypeJob.ID? =
        ProcessInfo.processInfo.arguments.contains("-prototype-expanded") ? PrototypeJob.fixtures.first?.id : nil
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        ZStack {
            if let job = store.current {
                let isExpanded = expandedID == job.id
                card(job, isExpanded) { toggle(job) }
                    .swipeToDecide(allowsUp: !isExpanded, showsStamp: showsStamp, onProgress: { progress = $0 }) { dir in
                        expandedID = nil
                        withAnimation(.snappy) {
                            store.act(dir == .right ? .shortlist : dir == .left ? .dismiss : .skip, on: job)
                        }
                    }
                    .id(job.id)
                    .transition(.asymmetric(insertion: insertion, removal: removal))
                    .accessibilityAction(named: "Shortlist") { store.act(.shortlist, on: job) }
                    .accessibilityAction(named: "Dismiss") { store.act(.dismiss, on: job) }
                    .accessibilityAction(named: "Skip") { store.act(.skip, on: job) }
                    .accessibilityAction(named: isExpanded ? "Collapse details" : "Details") { toggle(job) }
            } else {
                QueueClearView(store: store)
            }
        }
        .onChange(of: store.current?.id) { expandedID = nil }
        .toolbar {
            if let job = store.current {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Skip") { withAnimation(.snappy) { store.act(.skip, on: job) } }
                }
            }
        }
    }

    private func toggle(_ job: PrototypeJob) {
        withAnimation(reduceMotion ? .easeInOut(duration: 0.2) : .spring(duration: 0.45, bounce: 0.12)) {
            expandedID = expandedID == job.id ? nil : job.id
        }
    }

    private var insertion: AnyTransition {
        reduceMotion ? .opacity : .scale(scale: 0.94).combined(with: .opacity)
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
#endif
