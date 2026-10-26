#if DEBUG
import SwiftUI

struct BandScoreLine: View {
    let job: PrototypeJob

    var body: some View {
        HStack(spacing: 8) {
            BandChip(band: job.band)
            Text(job.score, format: .number)
                .font(.subheadline.monospacedDigit())
                .foregroundStyle(.secondary)
            if job.unknownCount > 0 {
                Text("· \(job.unknownCount) unknown")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Text(job.age)
                .font(.footnote.monospacedDigit())
                .foregroundStyle(.secondary)
        }
    }
}

struct BandChip: View {
    let band: PrototypeJob.Band
    @Environment(\.colorScheme) private var scheme

    var body: some View {
        Text(band.rawValue)
            .font(.footnote.weight(.medium))
            .padding(.horizontal, 10)
            .padding(.vertical, 3)
            .foregroundStyle(ink)
            .background(band.colour.opacity(0.14), in: .capsule)
            .overlay(Capsule().strokeBorder(band.colour.opacity(0.3)))
    }

    private var ink: Color {
        if #available(iOS 18, *) {
            return band.colour.mix(with: scheme == .dark ? .white : .black, by: 0.35)
        }
        return band.colour
    }
}

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

struct GroupedBreakdown: View {
    let job: PrototypeJob
    private let order: [PrototypeJob.Effect] = [.gated, .boost, .meets, .misses, .level, .unknown]

    var body: some View {
        ForEach(order, id: \.self) { effect in
            let rows = job.rows.filter { $0.effect == effect }
            if !rows.isEmpty {
                Section(effect.groupTitle) {
                    ForEach(rows, id: \.self) { BreakdownRowView(row: $0) }
                }
            }
        }
    }
}

struct TriageActionBar: View {
    let job: PrototypeJob
    let store: TriagePrototypeStore
    var onDetails: (() -> Void)?

    var body: some View {
        HStack(spacing: 12) {
            Button("Dismiss", systemImage: "xmark") { act(.dismiss) }
                .buttonStyle(.bordered)
                .tint(.secondary)
                .controlSize(.large)
            TriageMoreMenu(job: job, store: store, onDetails: onDetails) {
                Label("More", systemImage: "ellipsis")
            }
            .labelStyle(.iconOnly)
            .buttonStyle(.bordered)
            .buttonBorderShape(.circle)
            .tint(.secondary)
            .controlSize(.large)
            Button("Shortlist", systemImage: "checkmark") { act(.shortlist) }
                .buttonStyle(.borderedProminent)
                .tint(.brandTint)
                .controlSize(.large)
        }
        .lineLimit(1)
    }

    private func act(_ action: TriagePrototypeStore.Action) {
        withAnimation(.easeOut(duration: 0.25)) { store.act(action, on: job) }
    }
}

struct TriageMoreMenu<MenuLabel: View>: View {
    let job: PrototypeJob
    let store: TriagePrototypeStore
    var onDetails: (() -> Void)?
    @ViewBuilder let label: () -> MenuLabel

    @State private var showReasons = false
    @State private var showNote = false

    var body: some View {
        Menu {
            Button("Shortlist and draft CV", systemImage: "doc.badge.plus") { act(.shortlistDraft) }
            Button("Dismiss with reason…", systemImage: "text.badge.xmark") { showReasons = true }
            Button("Add note…", systemImage: "note.text") { showNote = true }
            if let onDetails {
                Button("Details", systemImage: "doc.text", action: onDetails)
            }
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

struct PrototypeJobDetail: View {
    let job: PrototypeJob
    let store: TriagePrototypeStore

    var body: some View {
        List {
            Section {
                VStack(alignment: .leading, spacing: 12) {
                    BandScoreLine(job: job)
                    MetaLines(job: job)
                }
                .padding(.vertical, 4)
            }
            Section {
                TriageActionBar(job: job, store: store)
                    .frame(maxWidth: .infinity)
                    .listRowBackground(Color.clear)
            }
            Section("Description") {
                Text(job.description).font(.body)
            }
            GroupedBreakdown(job: job)
            Section("Note") {
                Text("No note yet").foregroundStyle(.secondary)
            }
            Section("Recruiter contacts") {
                Text("None logged").foregroundStyle(.secondary)
                Button("Log contact", systemImage: "person.badge.plus") {}
            }
            Section {
                Link("Open original posting", destination: URL(string: "https://example.com")!)
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ShareLink(item: URL(string: "https://example.com/jobs/\(job.id)")!)
        }
    }
}

extension View {
    func cardSurface() -> some View {
        background(Color(.secondarySystemGroupedBackground), in: .rect(cornerRadius: 16, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
    }
}
#endif
