import SwiftUI

struct ApplicationInPlaceOfSheet: View {
    let role: DraftRoles.Role
    let candidate: DraftRoles.Bullet
    let model: ApplicationModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                Section("Adding") {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(candidate.text).font(.subheadline)
                        ApplicationBulletMeta(bullet: candidate)
                    }
                }
                Section {
                    ForEach(role.bullets) { old in
                        Button {
                            Task { await model.replace(old, with: candidate, in: role) }
                            dismiss()
                        } label: {
                            ApplicationBudgetOption(
                                bullet: old, result: role.budgetLabel(forLines: role.linesAfter(replacing: old, with: candidate)),
                                over: role.linesAfter(replacing: old, with: candidate) > role.budget)
                        }
                        .buttonStyle(.plain)
                    }
                    if role.fits(candidate) {
                        Button("Use the ^[\(role.free) free line](inflect: true) instead", systemImage: "plus") {
                            Task { await model.add(candidate, to: role) }
                            dismiss()
                        }
                        .frame(minHeight: 44)
                    }
                } header: {
                    Text("In place of")
                } footer: {
                    Text("You can go over \(role.employer)'s \(role.budget) lines for now, then remove something before you keep the CV.")
                }
            }
            .navigationTitle("Add from your bank")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
        }
        .presentationDetents([.medium, .large])
    }
}

struct ApplicationBudgetOption: View {
    let bullet: DraftRoles.Bullet
    let result: String
    let over: Bool
    var showsOrigin = false

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            VStack(alignment: .leading, spacing: 4) {
                Text(bullet.text).font(.subheadline).multilineTextAlignment(.leading)
                ApplicationBulletMeta(bullet: bullet, showsOrigin: showsOrigin)
            }
            Spacer(minLength: 0)
            if over {
                DotLabel(text: result, hue: .red)
            } else {
                Text(result).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
            }
        }
        .frame(minHeight: 44)
        .contentShape(.rect)
    }
}
