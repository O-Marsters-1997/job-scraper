import SwiftUI

struct DismissReasonSheet: View {
    let onDismissJob: ([GradeReason]) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var selected: Set<GradeReason> = []

    var body: some View {
        NavigationStack {
            List(GradeReason.allCases, id: \.self) { reason in
                Button {
                    if selected.contains(reason) { selected.remove(reason) } else { selected.insert(reason) }
                } label: {
                    HStack {
                        Text(reason.title).foregroundStyle(.primary)
                        Spacer()
                        if selected.contains(reason) { Image(systemName: "checkmark").foregroundStyle(.tint) }
                    }
                }
                .accessibilityAddTraits(selected.contains(reason) ? .isSelected : [])
            }
            .navigationTitle("Why not?")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Dismiss") {
                        dismiss()
                        onDismissJob(GradeReason.allCases.filter(selected.contains))
                    }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }
}

private extension GradeReason {
    var title: String {
        switch self {
        case .companySize: "Company size"
        default: rawValue.capitalized
        }
    }
}
