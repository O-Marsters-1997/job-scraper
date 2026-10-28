import SwiftUI

struct NoteSheet: View {
    let load: () async throws -> String
    let onSave: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var text = ""
    @State private var isLoaded = false

    var body: some View {
        NavigationStack {
            TextEditor(text: $text)
                .padding(.horizontal)
                .accessibilityLabel("Note")
                .overlay { if !isLoaded { ProgressView() } }
                .disabled(!isLoaded)
                .navigationTitle("Note")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Save") {
                            dismiss()
                            onSave(text)
                        }
                        .disabled(!isLoaded)
                    }
                }
        }
        .presentationDetents([.medium])
        .task {
            do {
                text = try await load()
                isLoaded = true
            } catch {
                dismiss()
            }
        }
    }
}
