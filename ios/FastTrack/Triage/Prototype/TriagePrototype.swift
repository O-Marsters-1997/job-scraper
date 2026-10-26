#if DEBUG
import SwiftUI

struct TriagePrototype: View {
    @State private var store = TriagePrototypeStore()
    @State private var tab = ProcessInfo.processInfo.arguments.contains("-prototype-due") ? 1 : 0

    var body: some View {
        Group {
            if store.signedIn {
                tabs.transition(.opacity)
            } else {
                PrototypeLogin { withAnimation(.easeInOut(duration: 0.3)) { store.signedIn = true } }
                    .transition(.opacity)
            }
        }
        .environment(store)
        .tint(.brandTint)
        .sensoryFeedback(trigger: store.log.count) { old, new in
            guard new > old, let last = store.log.last, last.removed else { return nil }
            if last.action.isPositive { return .success }
            return last.action == .skip ? nil : .impact(weight: .light)
        }
    }

    private var tabs: some View {
        TabView(selection: $tab) {
            NavigationStack {
                VariantWallet(store: store)
                    .safeAreaInset(edge: .top, spacing: 0) { offlineRow }
                    .safeAreaInset(edge: .bottom, spacing: 8) { bottomStack }
                    .background(Color(.systemBackground))
                    .toolbar {
                        ToolbarItem(placement: .topBarLeading) {
                            Button("Undo", systemImage: "arrow.uturn.backward") { withAnimation { store.undo() } }
                                .disabled(!store.canUndo)
                        }
                    }
            }
            .tabItem { Label("Triage", systemImage: "rectangle.stack") }
            .tag(0)

            NavigationStack {
                DueListView()
                    .safeAreaInset(edge: .bottom, spacing: 8) { bottomStack }
            }
            .tabItem { Label("Due", systemImage: "checklist") }
            .badge(store.offline ? 0 : store.dueBadge)
            .tag(1)
        }
    }

    @ViewBuilder
    private var offlineRow: some View {
        if store.offline {
            Label(store.queuedCount > 0 ? "Offline · \(store.queuedCount) queued" : "Offline", systemImage: "wifi.slash")
                .font(.footnote.weight(.medium))
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
                .background(Color(.tertiarySystemGroupedBackground))
                .transition(.move(edge: .top).combined(with: .opacity))
        }
    }

    private var bottomStack: some View {
        VStack(spacing: 8) {
            if let note = store.syncNote {
                Text(note)
                    .font(.footnote.weight(.semibold))
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                    .background(Color(.systemBackground), in: .capsule)
                    .overlay(Capsule().strokeBorder(Color(.separator)))
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                    .task {
                        try? await Task.sleep(for: .seconds(4))
                        withAnimation { store.syncNote = nil }
                    }
            }
            UndoBanner(store: store)
            controls
        }
        .animation(.snappy, value: store.banner)
        .animation(.snappy, value: store.syncNote)
        .animation(.snappy, value: store.offline)
    }

    private var controls: some View {
        VStack(spacing: 2) {
            HStack(spacing: 16) {
                Text("PROTOTYPE").font(.caption2.weight(.bold)).tracking(0.5)
                Button(store.offline ? "Go online" : "Go offline", systemImage: store.offline ? "wifi" : "wifi.slash") {
                    store.offline.toggle()
                }
                Button("Reset", systemImage: "arrow.counterclockwise") { store.reset() }
                Button("Sign out", systemImage: "rectangle.portrait.and.arrow.right") {
                    withAnimation { store.signedIn = false }
                }
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
}
#endif
