import Foundation

struct AppNotification: Decodable, Identifiable, Hashable {
    let id: String
    let kind: NotificationKind
    let jobID: String
    let title: String
    let body: String
    let job: NotificationJob
    let createdAt: Date
    let readAt: Date?

    var isUnread: Bool { readAt == nil }

    enum CodingKeys: String, CodingKey {
        case id, kind, title, body, job
        case jobID = "job_id"
        case createdAt = "created_at"
        case readAt = "read_at"
    }
}

enum NotificationKind: String, Decodable, Hashable {
    case newMatch = "new_match"
    case unknown

    init(from decoder: any Decoder) throws {
        self = try NotificationKind(rawValue: decoder.singleValueContainer().decode(String.self)) ?? .unknown
    }

    var verb: String? {
        switch self {
        case .newMatch: "is hiring for"
        case .unknown: nil
        }
    }

    var symbol: String {
        switch self {
        case .newMatch: "sparkle"
        case .unknown: "bell.fill"
        }
    }
}

struct NotificationJob: Decodable, Hashable {
    let title: String
    let companyName: String
    let location: String
    let salaryRaw: String
    let band: Band?
    let score: Int?

    var facts: String { [location, salaryRaw].filter { !$0.isEmpty }.joined(separator: " · ") }

    enum CodingKeys: String, CodingKey {
        case title, location, band, score
        case companyName = "company_name"
        case salaryRaw = "salary_raw"
    }
}

extension NotificationJob {
    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        title = try c.decode(String.self, forKey: .title)
        companyName = try c.decode(String.self, forKey: .companyName)
        location = try c.decode(String.self, forKey: .location)
        salaryRaw = try c.decode(String.self, forKey: .salaryRaw)
        band = try Band(rawValue: c.decode(String.self, forKey: .band))
        score = try c.decodeIfPresent(Int.self, forKey: .score)
    }
}

extension AppNotification {
    static let samples: [AppNotification] = [
        AppNotification(
            id: "n1", kind: .newMatch, jobID: "sample", title: "New match", body: "",
            job: NotificationJob(
                title: "Senior Go Engineer", companyName: "Monzo", location: "London", salaryRaw: "£85,000–£100,000",
                band: .great, score: 92),
            createdAt: Date(timeIntervalSinceNow: -360), readAt: nil),
        AppNotification(
            id: "n2", kind: .newMatch, jobID: "sample", title: "New match", body: "",
            job: NotificationJob(
                title: "Platform Engineer", companyName: "Octopus Energy", location: "Remote UK", salaryRaw: "",
                band: .good, score: 78),
            createdAt: Date(timeIntervalSinceNow: -86_400), readAt: nil),
        AppNotification(
            id: "n3", kind: .unknown, jobID: "sample", title: "Weekly digest", body: "12 new jobs this week",
            job: NotificationJob(title: "", companyName: "", location: "", salaryRaw: "", band: nil, score: nil),
            createdAt: Date(timeIntervalSinceNow: -4 * 86_400), readAt: .now),
    ]
}
