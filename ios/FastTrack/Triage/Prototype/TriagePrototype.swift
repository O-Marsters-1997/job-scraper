#if DEBUG
import SwiftUI

struct TriagePrototype: View {
    private static let variants = [("1", "Wallet"), ("2", "Arcade"), ("3", "Instrument")]

    @AppStorage("triagePrototypeVariant") private var variant = "1"
    @State private var store = TriagePrototypeStore()

    var body: some View {
        TabView {
            NavigationStack {
                Group {
                    switch variant {
                    case "2": VariantArcade(store: store)
                    case "3": VariantInstrument(store: store)
                    default: VariantWallet(store: store)
                    }
                }
                .safeAreaInset(edge: .bottom, spacing: 8) {
                    VStack(spacing: 8) {
                        UndoBanner(store: store)
                        switcher
                    }
                    .animation(.snappy, value: store.banner)
                }
                .background(Color(.systemGroupedBackground))
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Button("Undo", systemImage: "arrow.uturn.backward") { withAnimation { store.undo() } }
                            .disabled(!store.canUndo)
                    }
                }
            }
            .tabItem { Label("Triage", systemImage: "rectangle.stack") }

            NavigationStack {
                ContentUnavailableView("Due list", systemImage: "checklist", description: Text("Not part of this prototype."))
                    .navigationTitle("Due")
            }
            .tabItem { Label("Due", systemImage: "checklist") }
            .badge(3)
        }
        .tint(.brandTint)
        .sensoryFeedback(trigger: store.log.count) { old, new in
            guard new > old, let last = store.log.last, last.removed else { return nil }
            if last.action.isPositive { return .success }
            return last.action == .skip ? nil : .impact(weight: .light)
        }
    }

    private var switcher: some View {
        let index = Self.variants.firstIndex { $0.0 == variant } ?? 0
        return VStack(spacing: 2) {
            HStack(spacing: 12) {
                Button("Previous variant", systemImage: "chevron.left") { cycle(index, by: -1) }
                Text("\(Self.variants[index].0) — \(Self.variants[index].1)")
                    .font(.footnote.weight(.semibold))
                    .frame(minWidth: 180)
                Button("Next variant", systemImage: "chevron.right") { cycle(index, by: 1) }
                Button("Reset queue", systemImage: "arrow.counterclockwise") { store.reset() }
            }
            .labelStyle(.iconOnly)
            Text(store.stateLine)
                .font(.caption2.monospacedDigit())
                .lineLimit(1)
                .opacity(0.8)
        }
        .foregroundStyle(Color(.systemBackground))
        .tint(Color(.systemBackground))
        .padding(.horizontal, 14)
        .padding(.vertical, 6)
        .background(Color(.label).opacity(0.85), in: .capsule)
        .padding(.bottom, 4)
    }

    private func cycle(_ index: Int, by step: Int) {
        let next = (index + step + Self.variants.count) % Self.variants.count
        variant = Self.variants[next].0
    }
}
#endif
