import Foundation

struct DueJob: Decodable, Identifiable, Hashable {
    enum DraftState: String, Decodable { case pending, ready, failed }

    struct Task: Decodable, Hashable {
        let id: String
        let typeName: String
        let systemKey: String?
        let notes: String

        enum CodingKeys: String, CodingKey {
            case id, notes
            case typeName = "type_name"
            case systemKey = "system_key"
        }
    }

    struct Draft: Decodable, Hashable {
        let id: String
        let state: DraftState
        let cvTitle: String

        enum CodingKeys: String, CodingKey {
            case id, state
            case cvTitle = "cv_title"
        }
    }

    var id: String { jobID }
    let jobID: String
    let title: String
    let companySlug: String
    let companyKind: CompanyKind
    let band: Band?
    let score: Int?
    let applicationID: String
    let chaseBy: Date?
    let applyTaskID: String?
    let tasks: [Task]
    let draft: Draft?
    var keptDraftID: String? = nil

    var isAtDesk: Bool { tasks.contains { $0.systemKey == "finish" } }
    var showsApplyRow: Bool { applyTaskID != nil && draft == nil && keptDraftID == nil && !isAtDesk }
    var isReadyToApply: Bool { applyTaskID != nil && draft == nil && keptDraftID != nil && !isAtDesk }
    var keptDraft: Draft? { keptDraftID.map { Draft(id: $0, state: .ready, cvTitle: "Kept CV") } }
    var companyName: String { companySlug.replacing("-", with: " ").capitalized }

    func matches(_ query: String) -> Bool {
        let query = query.trimmingCharacters(in: .whitespaces)
        return query.isEmpty || title.localizedStandardContains(query) || companyName.localizedStandardContains(query)
    }

    enum CodingKeys: String, CodingKey {
        case title, band, score, tasks, draft
        case jobID = "job_id"
        case companySlug = "company_slug"
        case companyKind = "company_kind"
        case applicationID = "application_id"
        case chaseBy = "chase_by"
        case applyTaskID = "apply_task_id"
        case keptDraftID = "kept_draft_id"
    }
}

extension DueJob {
    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        jobID = try c.decode(String.self, forKey: .jobID)
        title = try c.decode(String.self, forKey: .title)
        companySlug = try c.decode(String.self, forKey: .companySlug)
        companyKind = (try? c.decode(CompanyKind.self, forKey: .companyKind)) ?? .unclassified
        band = Band(rawValue: try c.decode(String.self, forKey: .band).lowercased())
        score = try c.decodeIfPresent(Int.self, forKey: .score)
        applicationID = try c.decode(String.self, forKey: .applicationID)
        chaseBy = try c.decodeIfPresent(Date.self, forKey: .chaseBy)
        applyTaskID = try c.decodeIfPresent(String.self, forKey: .applyTaskID)
        tasks = try c.decode([Task].self, forKey: .tasks)
        draft = try c.decodeIfPresent(Draft.self, forKey: .draft)
        keptDraftID = try c.decodeIfPresent(String.self, forKey: .keptDraftID)
    }

    func chasing(by date: Date?) -> DueJob {
        DueJob(jobID: jobID, title: title, companySlug: companySlug, companyKind: companyKind, band: band, score: score,
               applicationID: applicationID,
               chaseBy: date, applyTaskID: applyTaskID, tasks: tasks, draft: draft, keptDraftID: keptDraftID)
    }

    func withTasks(_ tasks: [Task]) -> DueJob {
        DueJob(jobID: jobID, title: title, companySlug: companySlug, companyKind: companyKind, band: band, score: score,
               applicationID: applicationID,
               chaseBy: chaseBy, applyTaskID: applyTaskID, tasks: tasks, draft: draft, keptDraftID: keptDraftID)
    }

    func withApplyTask(_ id: String?) -> DueJob {
        DueJob(jobID: jobID, title: title, companySlug: companySlug, companyKind: companyKind, band: band, score: score,
               applicationID: applicationID,
               chaseBy: chaseBy, applyTaskID: id, tasks: tasks, draft: draft, keptDraftID: keptDraftID)
    }

    static let sample = DueJob(
        jobID: "sample", title: "Senior Go Engineer", companySlug: "monzo", companyKind: .direct, band: .great, score: 92,
        applicationID: "application-sample", chaseBy: Calendar.current.date(byAdding: .day, value: -2, to: .now),
        applyTaskID: nil, tasks: [Task(id: "task-1", typeName: "Reach out", systemKey: nil, notes: "")],
        draft: Draft(id: "draft-1", state: .ready, cvTitle: "Backend")
    )
}
