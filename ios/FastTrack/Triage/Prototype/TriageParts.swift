#if DEBUG
import SwiftUI

struct MetaLines: View {
    let job: PrototypeJob
    var titleFont: Font = .title2.weight(.semibold)

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(job.title)
                .font(titleFont)
                .lineLimit(3)
            Text(job.company)
                .font(.headline)
                .foregroundStyle(.secondary)
            Text("\(job.location) · \(job.arrangement)")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            if let salary = job.salary {
                Text(salary)
                    .font(.subheadline.monospacedDigit())
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct BreakdownRowView: View {
    let row: PrototypeJob.Row

    var body: some View {
        Label {
            Text(row.effect == .gated ? "Gate: \(row.label)" : row.label)
                .foregroundStyle(row.effect == .unknown ? .secondary : .primary)
        } icon: {
            Image(systemName: row.effect.symbol).foregroundStyle(row.effect.colour)
        }
        .font(.subheadline)
    }
}

struct TriageMoreMenu<MenuLabel: View>: View {
    let job: PrototypeJob
    let store: TriagePrototypeStore
    @ViewBuilder let label: () -> MenuLabel

    @State private var showReasons = false
    @State private var showNote = false

    var body: some View {
        Menu {
            Button("Shortlist and draft CV", systemImage: "doc.badge.plus") { act(.shortlistDraft) }
            Button("Dismiss with reason…", systemImage: "text.badge.xmark") { showReasons = true }
            Button("Add note…", systemImage: "note.text") { showNote = true }
            Divider()
            Button("Exclude \(job.company)", systemImage: "nosign", role: .destructive) { act(.exclude) }
        } label: {
            label()
        }
        .sheet(isPresented: $showReasons) {
            DismissReasonSheet { act(.dismissReason) }
        }
        .sheet(isPresented: $showNote) {
            NoteSheet { act(.note) }
        }
    }

    private func act(_ action: TriagePrototypeStore.Action) {
        withAnimation(.easeOut(duration: 0.25)) { store.act(action, on: job) }
    }
}

struct DismissReasonSheet: View {
    let onDismissJob: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var selected: Set<String> = []
    private let reasons = ["Seniority", "Role", "Tech", "Domain", "Location", "Salary", "Contract", "Company size"]

    var body: some View {
        NavigationStack {
            List(reasons, id: \.self) { reason in
                Button {
                    if selected.contains(reason) { selected.remove(reason) } else { selected.insert(reason) }
                } label: {
                    HStack {
                        Text(reason).foregroundStyle(.primary)
                        Spacer()
                        if selected.contains(reason) { Image(systemName: "checkmark").foregroundStyle(.tint) }
                    }
                }
            }
            .navigationTitle("Why not?")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Dismiss") {
                        dismiss()
                        onDismissJob()
                    }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }
}

struct NoteSheet: View {
    let onSave: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var text = ""

    var body: some View {
        NavigationStack {
            TextEditor(text: $text)
                .padding(.horizontal)
                .navigationTitle("Note")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Save") {
                            dismiss()
                            onSave()
                        }
                        .disabled(text.isEmpty)
                    }
                }
        }
        .presentationDetents([.medium])
    }
}

struct UndoBanner: View {
    let store: TriagePrototypeStore

    var body: some View {
        if let entry = store.banner {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(entry.action.rawValue).font(.footnote.weight(.semibold))
                    Text(entry.job.title).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                if entry.action == .shortlist {
                    Button("Draft CV") { store.act(.draft, on: entry.job) }
                        .font(.footnote.weight(.semibold))
                }
                if entry.removed {
                    Button("Undo") { withAnimation { store.undo() } }
                        .font(.footnote.weight(.semibold))
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(.regularMaterial, in: .rect(cornerRadius: 14, style: .continuous))
            .padding(.horizontal)
            .transition(.move(edge: .bottom).combined(with: .opacity))
            .task(id: entry.id) {
                try? await Task.sleep(for: .seconds(4))
                withAnimation { if store.banner == entry { store.banner = nil } }
            }
        }
    }
}

struct QueueClearView: View {
    let store: TriagePrototypeStore

    var body: some View {
        ContentUnavailableView {
            Label("Queue clear", systemImage: "checkmark.circle")
        } description: {
            Text("New jobs arrive after the next scrape.")
        } actions: {
            Button("Reset prototype queue") { store.reset() }
        }
    }
}
#endif
