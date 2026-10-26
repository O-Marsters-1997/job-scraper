#if DEBUG
import SwiftUI

struct JobDetailBody: View {
    let job: PrototypeJob
    @Environment(TriagePrototypeStore.self) private var store
    @State private var editingNote = false
    @State private var loggingContact = false
    private let order: [PrototypeJob.Effect] = [.gated, .boost, .meets, .misses, .level, .unknown]

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            section("Description") {
                Text(job.description)
                    .font(.body)
                    .lineSpacing(3)
            }
            section("Why this score") {
                VStack(alignment: .leading, spacing: 16) {
                    ForEach(order, id: \.self) { effect in
                        let rows = job.rows.filter { $0.effect == effect }
                        if !rows.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Text(effect.groupTitle)
                                    .font(.footnote.weight(.medium))
                                    .foregroundStyle(.secondary)
                                ForEach(rows, id: \.self) { BreakdownRowView(row: $0) }
                            }
                        }
                    }
                }
            }
            section("Note") {
                if let note = store.notes[job.id] {
                    Text(note)
                }
                Button(store.notes[job.id] == nil ? "Add note" : "Edit note", systemImage: "note.text") { editingNote = true }
                    .font(.subheadline.weight(.medium))
            }
            section("Recruiter contacts") {
                if store.offline {
                    Text("Available when online").foregroundStyle(.secondary)
                } else {
                    let contacts = store.contacts[job.id] ?? []
                    if contacts.isEmpty {
                        Text("None logged").foregroundStyle(.secondary)
                    }
                    ForEach(contacts) { contact in
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(contact.name)
                                Text(contact.channel).font(.footnote).foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text(contact.date.shortGB).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
                        }
                    }
                    Button("Log contact", systemImage: "person.badge.plus") { loggingContact = true }
                        .font(.subheadline.weight(.medium))
                }
            }
            Link(destination: URL(string: "https://example.com/jobs/\(job.id)")!) {
                Label("Open original posting", systemImage: "arrow.up.right")
                    .font(.subheadline.weight(.medium))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .sheet(isPresented: $editingNote) {
            NoteSheet(initial: store.notes[job.id] ?? "") { store.saveNote($0, for: job) }
        }
        .sheet(isPresented: $loggingContact) {
            ContactSheet { store.logContact($0, for: job) }
        }
    }

    private func section(_ title: String, @ViewBuilder content: () -> some View) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title).font(.headline)
            content()
        }
    }
}

struct ContactSheet: View {
    let onSave: (PrototypeContact) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var channel = "LinkedIn"
    @State private var date = Date.now

    var body: some View {
        NavigationStack {
            Form {
                TextField("Name", text: $name)
                Picker("Channel", selection: $channel) {
                    ForEach(["LinkedIn", "Email", "Phone", "In person"], id: \.self) { Text($0) }
                }
                DatePicker("Date", selection: $date, displayedComponents: .date)
            }
            .navigationTitle("Log contact")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        dismiss()
                        onSave(PrototypeContact(name: name, channel: channel, date: date))
                    }
                    .disabled(name.isEmpty)
                }
            }
        }
        .presentationDetents([.medium])
    }
}
#endif
