import SwiftUI

struct FinishAtDeskSheet: View {
    let model: ApplicationModel
    @Environment(\.dismiss) private var dismiss
    @State private var note = ""

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Optional note for the desk", text: $note, axis: .vertical).lineLimit(3...6)
                } footer: {
                    Text("Adds a \"Finish application\" task. The draft stays as it is for you to open at the desk.")
                }
            }
            .navigationTitle("Finish at desk")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add task") {
                        Task {
                            await model.finishAtDesk(note: note)
                            if model.exit != nil { dismiss() }
                        }
                    }
                }
            }
        }
        .presentationDetents([.medium])
    }
}
