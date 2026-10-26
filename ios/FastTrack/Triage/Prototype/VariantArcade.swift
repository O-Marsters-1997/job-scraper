#if DEBUG
import SwiftUI

struct VariantArcade: View {
    let store: TriagePrototypeStore
    @State private var progress: CGFloat = 0
    @State private var seal: TriagePrototypeStore.Entry?
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var behind: Int { min(max(store.queue.count - 1, 0), 3) }
    private var pressure: Int { Int(min(abs(progress), 1) * 4) }

    var body: some View {
        VStack(spacing: 18) {
            ZStack(alignment: .bottom) {
                DeckEdges(colours: Array(repeating: .clear, count: behind), radius: 28)
                TriageCardHost(store: store, progress: $progress) { job, isExpanded, toggle in
                    TactileCard(job: job, isExpanded: isExpanded, toggle: toggle, radius: 28) {
                        header(job)
                    }
                }
                if let seal {
                    SealView(entry: seal)
                        .frame(maxHeight: .infinity)
                        .transition(reduceMotion ? .opacity : .scale(scale: 1.6).combined(with: .opacity))
                        .id(seal.id)
                }
            }
            .padding(.horizontal)
            .padding(.bottom, CGFloat(behind) * 7)
            if let job = store.current {
                buttons(job)
            }
        }
        .padding(.bottom, 8)
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
        .sensoryFeedback(trigger: pressure) { old, new in
            new > old ? .impact(flexibility: .soft, intensity: 0.25 + Double(new) * 0.18) : nil
        }
        .onChange(of: store.lastRemoved?.id) {
            guard let entry = store.lastRemoved, entry.action != .skip else { return }
            withAnimation(.spring(duration: 0.35, bounce: 0.45)) { seal = entry }
        }
        .task(id: seal?.id) {
            guard seal != nil else { return }
            try? await Task.sleep(for: .milliseconds(650))
            withAnimation(.easeOut(duration: 0.2)) { seal = nil }
        }
    }

    private func header(_ job: PrototypeJob) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(job.band.rawValue)
                .font(.title2.weight(.heavy))
                .fontDesign(.rounded)
                .foregroundStyle(job.band.colour)
            Text(job.score, format: .number)
                .font(.title2.weight(.semibold).monospacedDigit())
                .fontDesign(.rounded)
                .foregroundStyle(.secondary)
            Spacer()
            Text("\(store.queue.count)")
                .font(.headline.monospacedDigit())
                .fontDesign(.rounded)
                .contentTransition(.numericText(countsDown: true))
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .background(Color(.tertiarySystemGroupedBackground), in: .capsule)
                .accessibilityLabel("\(store.queue.count) left")
        }
        .padding(.horizontal, 22)
        .padding(.vertical, 18)
        .background(job.band.colour.opacity(0.1))
    }

    private func buttons(_ job: PrototypeJob) -> some View {
        HStack(spacing: 24) {
            Button { act(.dismiss, job) } label: {
                Image(systemName: "xmark")
                    .font(.title.weight(.bold))
                    .foregroundStyle(.secondary)
                    .frame(width: 72, height: 72)
                    .background(Color(.secondarySystemGroupedBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator), lineWidth: 1.5))
            }
            .buttonStyle(SquishStyle())
            .accessibilityLabel("Dismiss")
            TriageMoreMenu(job: job, store: store) {
                Image(systemName: "ellipsis")
                    .font(.body.weight(.bold))
                    .foregroundStyle(.secondary)
                    .frame(width: 50, height: 50)
                    .background(Color(.secondarySystemGroupedBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator), lineWidth: 1.5))
                    .accessibilityLabel("More")
            }
            .tint(.secondary)
            Button { act(.shortlist, job) } label: {
                Image(systemName: "checkmark")
                    .font(.title.weight(.heavy))
                    .foregroundStyle(.white)
                    .frame(width: 72, height: 72)
                    .background(Color.brandTint, in: .circle)
            }
            .buttonStyle(SquishStyle())
            .accessibilityLabel("Shortlist")
        }
    }

    private func act(_ action: TriagePrototypeStore.Action, _ job: PrototypeJob) {
        withAnimation(.snappy) { store.act(action, on: job) }
    }
}

private struct SquishStyle: ButtonStyle {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed && !reduceMotion ? 0.86 : 1)
            .animation(.spring(duration: 0.25, bounce: 0.55), value: configuration.isPressed)
            .sensoryFeedback(.impact(flexibility: .solid, intensity: 0.6), trigger: configuration.isPressed) { _, pressed in pressed }
    }
}

private struct SealView: View {
    let entry: TriagePrototypeStore.Entry

    var body: some View {
        let positive = entry.action.isPositive
        VStack(spacing: 6) {
            Image(systemName: positive ? "checkmark" : "xmark")
                .font(.system(.largeTitle, design: .rounded, weight: .heavy))
            Text(entry.action.rawValue)
                .font(.subheadline.weight(.bold))
                .fontDesign(.rounded)
        }
        .foregroundStyle(.white)
        .frame(width: 132, height: 132)
        .background(positive ? Color.brandTint : Color(.systemGray), in: .circle)
        .rotationEffect(.degrees(positive ? -8 : 8))
        .allowsHitTesting(false)
        .accessibilityHidden(true)
    }
}
#endif
