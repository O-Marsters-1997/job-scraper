import SwiftUI

struct JobActionsMenu: View {
    let job: Job
    let onDecide: (TriageAction) -> Void
    let onDismissWithReason: () -> Void
    let onAddNote: () -> Void

    var body: some View {
        Button("Shortlist and draft CV", systemImage: "doc.badge.plus") { onDecide(.shortlist(draftCV: true)) }
        Button("Dismiss with reason…", systemImage: "text.badge.xmark", action: onDismissWithReason)
        Button("Add note…", systemImage: "note.text", action: onAddNote)
        if job.companyKind.isAgency {
            Button("Not an agency", systemImage: "building.2") { onDecide(.reclassify(to: .direct)) }
        } else {
            Button("This is an agency", systemImage: "person.2") { onDecide(.reclassify(to: .agency)) }
        }
        Button("Exclude company", systemImage: "nosign", role: .destructive) { onDecide(.excludeCompany) }
    }
}
