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
    enum Kind: String, Encodable { case jobDismissed = "job_dismissed" }

    let type: Kind
    let subjectID: String
    let reason: String
    let undone: Bool

    enum CodingKeys: String, CodingKey {
        case type, reason, undone
        case subjectID = "subject_id"
    }
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

struct NoteInput: Encodable {
    let body: String
}

struct DueList: Decodable {
    let jobs: [DueJob]
}

enum ContactChannel: String, Codable, CaseIterable {
    case linkedinMessage = "linkedin_message", email, call, inbound
}

enum ContactOutcome: String, Codable, CaseIterable {
    case noReply = "no_reply", rejected, spoke, submitted
}

struct RecruiterContact: Decodable, Identifiable, Hashable {
    let id: String
    let contactedOn: Date
    let channel: ContactChannel
    let outcome: ContactOutcome

    enum CodingKeys: String, CodingKey {
        case id, channel, outcome
        case contactedOn = "contacted_on"
    }
}

struct ChaseInput: Encodable {
    let chaseBy: String

    enum CodingKeys: String, CodingKey { case chaseBy = "chase_by" }
}

struct ContactInput: Encodable {
    let contactedOn: String
    let channel: ContactChannel
    let outcome: ContactOutcome

    enum CodingKeys: String, CodingKey {
        case channel, outcome
        case contactedOn = "contacted_on"
    }
}

struct TaskDoneInput: Encodable {
    let done: Bool
}

extension Date {
    static var dayFormat: Date.ISO8601FormatStyle {
        Date.ISO8601FormatStyle(timeZone: .current).year().month().day()
    }

    var dayString: String { formatted(Date.dayFormat) }
}
