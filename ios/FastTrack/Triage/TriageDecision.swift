import Foundation

enum TriageAction: Hashable {
    case shortlist(draftCV: Bool)
    case dismiss(reasons: [GradeReason])
    case skip
    case excludeCompany
    case reclassify(to: CompanyKind)
    case note(String)

    var keepsCard: Bool {
        switch self {
        case .note, .reclassify: true
        case .shortlist, .dismiss, .skip, .excludeCompany: false
        }
    }
}

enum UndoReceipt: Hashable {
    case application(id: String)
    case grade
    case seen
    case exclusion(companyID: String, removeName: Bool)
    case reclassify(companyID: String, previous: CompanyKind)
    case note(previous: String)
}

struct TriageDecision: Identifiable, Hashable {
    let id: UUID
    let job: Job
    let action: TriageAction
    var receipt: UndoReceipt?
}
