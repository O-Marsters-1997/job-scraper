import SwiftUI

struct JobNoteSection: View {
    let jobID: String
    @Environment(\.apiClient) private var client
    @State private var text = ""
    @State private var saved = ""
    @State private var isLoaded = false
    @State private var failure: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Note").font(.headline).accessibilityAddTraits(.isHeader)
            TextField("Add a note", text: $text, axis: .vertical)
                .lineLimit(3...12)
                .padding(12)
                .overlay(RoundedRectangle(cornerRadius: 12, style: .continuous).strokeBorder(Color(.separator)))
                .disabled(!isLoaded)
            if let failure {
                Text(failure).font(.footnote).foregroundStyle(.red)
            }
            if isLoaded {
                Button("Save note", action: save)
                    .disabled(text == saved)
                    .frame(minHeight: 44)
            } else if failure != nil {
                Button("Try again") { Task { await load() } }
                    .frame(minHeight: 44)
            }
        }
        .task(id: jobID) { await load() }
    }

    private func load() async {
        do {
            saved = try await client.note(jobID).body
            text = saved
            isLoaded = true
            failure = nil
        } catch is CancellationError {
            return
        } catch {
            failure = "Couldn't load the note."
        }
    }

    private func save() {
        let body = text
        Task {
            do {
                try await client.putNote(jobID, body)
                saved = body
                failure = nil
            } catch {
                failure = "Couldn't save the note."
            }
        }
    }
}

#Preview {
    JobNoteSection(jobID: "sample").padding().environment(\.apiClient, .mock())
}
