import Foundation

struct Application: Decodable {
    let id: String
    let jobID: String

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case jobID = "JobID"
    }
}

struct EventInput: Encodable {
    enum Kind: String, Encodable {
        case jobDismissed = "job_dismissed"
        case jobOpened = "job_opened"
    }

    let type: Kind
    let subjectID: String
    let reason: String
    let undone: Bool

    enum CodingKeys: String, CodingKey {
        case type, reason, undone
        case subjectID = "subject_id"
    }
}

struct ImpressionBatchInput: Encodable {
    struct Item: Encodable {
        let jobID: String
        let position: Int

        enum CodingKeys: String, CodingKey {
            case position
            case jobID = "job_id"
        }
    }

    let surface: String
    let sort: String
    let items: [Item]
}

struct CreateApplicationInput: Encodable {
    let jobID: String

    enum CodingKeys: String, CodingKey { case jobID = "job_id" }
}

struct GradeInput: Encodable {
    let grade: String
    let reasons: [GradeReason]
}

struct SeenInput: Encodable {
    let jobIDs: [String]
    let seen: Bool

    enum CodingKeys: String, CodingKey {
        case jobIDs = "JobIDs"
        case seen = "Seen"
    }
}

struct JobPage: Decodable {
    let items: [Job]
    let total: Int?
}

struct CompanyExclusion: Decodable {
    let name: String
    let added: Bool
}

struct JobNote: Decodable {
    let jobID: String
    let body: String

    enum CodingKeys: String, CodingKey {
        case jobID = "JobID"
        case body = "Body"
    }
}

struct EmptyBody: Encodable {}

struct UnexcludeInput: Encodable {
    let removeName: Bool
}

struct CompanyKindInput: Encodable {
    let kind: CompanyKind
}

struct NoteInput: Encodable {
    let body: String
}

struct DueList: Decodable {
    let jobs: [DueJob]
}

struct ChaseInput: Encodable {
    let chaseBy: String

    enum CodingKeys: String, CodingKey { case chaseBy = "chase_by" }
}


struct TaskType: Decodable {
    let id: String
    let name: String
    let systemKey: String?

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case name = "Name"
        case systemKey = "SystemKey"
    }
}

struct CreateTaskInput: Encodable {
    let typeID: String
    let jobID: String
    let notes: String

    enum CodingKeys: String, CodingKey {
        case notes
        case typeID = "type_id"
        case jobID = "job_id"
    }
}

struct TaskDoneInput: Encodable {
    let done: Bool
}

struct CVTemplate: Decodable {
    let docID: String
    let tabID: String
    let title: String
    let isDefault: Bool

    enum CodingKeys: String, CodingKey {
        case title = "Title"
        case docID = "DocID"
        case tabID = "TabID"
        case isDefault = "Default"
    }
}

struct DraftRef: Decodable {
    let id: String
}

struct DraftStatus: Decodable {
    let status: String
    let lastError: String

    var isKeeping: Bool { status == "keeping" }
}

struct DraftInput: Encodable {
    let jobID: String
    let docID: String
    let tabID: String
    let mode = "generate"

    enum CodingKeys: String, CodingKey {
        case mode
        case jobID = "jobId"
        case docID = "docId"
        case tabID = "tabId"
    }
}

struct DraftBulletEdit: Encodable {
    let text: String
    let achievementIDs: [String]

    enum CodingKeys: String, CodingKey {
        case text
        case achievementIDs = "achievementIds"
    }
}

struct DraftPositionEdit: Encodable {
    let positionID: String
    let bullets: [DraftBulletEdit]

    enum CodingKeys: String, CodingKey {
        case bullets
        case positionID = "positionId"
    }
}

struct DraftSlotsInput: Encodable {
    let positions: [DraftPositionEdit]
}

struct CoverLetter: Decodable, Identifiable {
    let id: String
    let createdAt: Date
    let doneAt: Date?
}

extension Date {
    static var dayFormat: Date.ISO8601FormatStyle {
        Date.ISO8601FormatStyle(timeZone: .current).year().month().day()
    }

    var dayString: String { formatted(Date.dayFormat) }
}
