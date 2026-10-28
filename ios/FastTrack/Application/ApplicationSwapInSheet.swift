import SwiftUI

struct ApplicationSwapInSheet: View {
    let role: DraftRoles.Role
    let old: DraftRoles.Bullet
    let model: ApplicationModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                Section("Replacing") {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(old.text).font(.subheadline)
                        ApplicationBulletMeta(bullet: old)
                    }
                }
                Section("From your bank, best fit first") {
                    ForEach(role.rankedBank) { candidate in
                        let after = role.linesAfter(replacing: old, with: candidate)
                        Button {
                            Task { await model.replace(old, with: candidate, in: role) }
                            dismiss()
                        } label: {
                            ApplicationBudgetOption(
                                bullet: candidate, result: role.budgetLabel(forLines: after), over: after > role.budget,
                                showsOrigin: true)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .navigationTitle(role.employer)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
        }
        .presentationDetents([.medium, .large])
    }
}
