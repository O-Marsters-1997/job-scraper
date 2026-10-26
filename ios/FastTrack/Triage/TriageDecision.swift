import Foundation

enum TriageAction: Hashable {
    case shortlist
    case dismiss(reasons: [GradeReason])
    case skip
    case excludeCompany
    case note(String)
}

enum UndoReceipt: Hashable {
    case application(id: String)
    case grade
    case seen
    case exclusion(companyID: String, removeName: Bool)
    case note(previous: String)
}

struct TriageDecision: Identifiable, Hashable {
    let id: UUID
    let job: Job
    let action: TriageAction
    var receipt: UndoReceipt?
}
