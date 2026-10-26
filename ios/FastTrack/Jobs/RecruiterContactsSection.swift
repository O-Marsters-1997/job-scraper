import SwiftUI

struct RecruiterContactsSection: View {
    let jobID: String
    @Environment(\.apiClient) private var client
    @State private var contacts: [RecruiterContact] = []
    @State private var isLogging = false
    @State private var failure: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Recruiter contacts").font(.headline).accessibilityAddTraits(.isHeader)
            ForEach(contacts) { contact in
                HStack {
                    Text("\(contact.channel.title) · \(contact.outcome.title)")
                    Spacer(minLength: 8)
                    Text(contact.contactedOn, format: .dateTime.day().month()).foregroundStyle(.secondary)
                }
                .font(.subheadline)
                .frame(minHeight: 40)
                .accessibilityElement(children: .combine)
            }
            if contacts.isEmpty && failure == nil {
                Text("No contacts logged.").font(.subheadline).foregroundStyle(.secondary)
            }
            if let failure {
                Text(failure).font(.footnote).foregroundStyle(.red)
            }
            Button("Log contact", systemImage: "plus") { isLogging = true }
                .frame(minHeight: 44)
        }
        .task(id: jobID) { await load() }
        .sheet(isPresented: $isLogging) {
            ContactSheet { channel, outcome in
                do {
                    _ = try await client.logContact(jobID, channel, outcome)
                    await load()
                    return true
                } catch {
                    failure = "Couldn't log that contact."
                    return false
                }
            }
        }
    }

    private func load() async {
        do {
            contacts = try await client.recruiterContacts(jobID)
            failure = nil
        } catch is CancellationError {
            return
        } catch {
            failure = "Couldn't load contacts."
        }
    }
}

#Preview {
    RecruiterContactsSection(jobID: "sample").padding().environment(\.apiClient, .mock())
}
