import Foundation

struct AppNotification: Decodable, Identifiable, Hashable {
    let id: String
    let jobID: String
    let title: String
    let body: String
    let createdAt: Date
    let readAt: Date?

    var isUnread: Bool { readAt == nil }

    enum CodingKeys: String, CodingKey {
        case id, title, body
        case jobID = "job_id"
        case createdAt = "created_at"
        case readAt = "read_at"
    }
}
