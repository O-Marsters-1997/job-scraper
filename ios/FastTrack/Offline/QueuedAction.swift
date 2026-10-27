import Foundation

struct QueuedAction: Codable, Identifiable {
    enum Kind: Codable {
        case shortlist(draftCV: Bool)
        case dismiss(reasons: [GradeReason])
        case skip
        case excludeCompany(companyID: String)
        case reclassify(companyID: String, to: CompanyKind)
        case note(body: String)
        case undo(of: UUID)
    }

    let id: UUID
    let jobID: String
    let kind: Kind
    let queuedAt: Date
}
