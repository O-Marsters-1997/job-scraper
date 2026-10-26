import Foundation

struct DueJob: Decodable, Identifiable, Hashable {
    enum DraftState: String, Decodable { case pending, ready, failed }

    struct Task: Decodable, Hashable {
        let id: String
        let typeName: String
        let notes: String

        enum CodingKeys: String, CodingKey {
            case id, notes
            case typeName = "type_name"
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
    let band: Band?
    let score: Int?
    let applicationID: String
    let chaseBy: Date?
    let applyTaskID: String?
    let tasks: [Task]
    let draft: Draft?

    var companyName: String { companySlug.replacing("-", with: " ").capitalized }

    enum CodingKeys: String, CodingKey {
        case title, band, score, tasks, draft
        case jobID = "job_id"
        case companySlug = "company_slug"
        case applicationID = "application_id"
        case chaseBy = "chase_by"
        case applyTaskID = "apply_task_id"
    }
}

extension DueJob {
    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        jobID = try c.decode(String.self, forKey: .jobID)
        title = try c.decode(String.self, forKey: .title)
        companySlug = try c.decode(String.self, forKey: .companySlug)
        band = Band(rawValue: try c.decode(String.self, forKey: .band).lowercased())
        score = try c.decodeIfPresent(Int.self, forKey: .score)
        applicationID = try c.decode(String.self, forKey: .applicationID)
        chaseBy = try c.decodeIfPresent(Date.self, forKey: .chaseBy)
        applyTaskID = try c.decodeIfPresent(String.self, forKey: .applyTaskID)
        tasks = try c.decode([Task].self, forKey: .tasks)
        draft = try c.decodeIfPresent(Draft.self, forKey: .draft)
    }

    func chasing(by date: Date?) -> DueJob {
        DueJob(jobID: jobID, title: title, companySlug: companySlug, band: band, score: score, applicationID: applicationID,
               chaseBy: date, applyTaskID: applyTaskID, tasks: tasks, draft: draft)
    }

    func withTasks(_ tasks: [Task]) -> DueJob {
        DueJob(jobID: jobID, title: title, companySlug: companySlug, band: band, score: score, applicationID: applicationID,
               chaseBy: chaseBy, applyTaskID: applyTaskID, tasks: tasks, draft: draft)
    }

    static let sample = DueJob(
        jobID: "sample", title: "Senior Go Engineer", companySlug: "monzo", band: .great, score: 92,
        applicationID: "application-sample", chaseBy: Calendar.current.date(byAdding: .day, value: -2, to: .now),
        applyTaskID: nil, tasks: [Task(id: "task-1", typeName: "Reach out", notes: "")],
        draft: Draft(id: "draft-1", state: .ready, cvTitle: "Backend")
    )
}
