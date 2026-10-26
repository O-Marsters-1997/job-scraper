import SwiftUI

struct ContactSheet: View {
    let log: (ContactChannel, ContactOutcome) async -> Bool
    @Environment(\.dismiss) private var dismiss
    @State private var channel = ContactChannel.email
    @State private var outcome = ContactOutcome.noReply
    @State private var isSubmitting = false

    private func submit() {
        isSubmitting = true
        Task {
            if await log(channel, outcome) { dismiss() }
            isSubmitting = false
        }
    }

    var body: some View {
        NavigationStack {
            Form {
                Picker("Channel", selection: $channel) {
                    ForEach(ContactChannel.allCases, id: \.self) { Text($0.title).tag($0) }
                }
                Picker("Outcome", selection: $outcome) {
                    ForEach(ContactOutcome.allCases, id: \.self) { Text($0.title).tag($0) }
                }
            }
            .navigationTitle("Log contact")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Log", action: submit).disabled(isSubmitting)
                }
            }
        }
        .presentationDetents([.medium])
    }
}

extension ContactChannel {
    var title: String {
        switch self {
        case .linkedinMessage: "LinkedIn message"
        case .email: "Email"
        case .call: "Call"
        case .inbound: "Inbound"
        }
    }
}

extension ContactOutcome {
    var title: String {
        switch self {
        case .noReply: "No reply"
        case .rejected: "Rejected"
        case .spoke: "Spoke"
        case .submitted: "Submitted"
        }
    }
}
