import Foundation

struct Job: Codable, Identifiable, Hashable {
    let id: String
    let title: String
    let companyID: String
    let companySlug: String
    let location: String
    let workArrangement: String
    let salaryRaw: String
    let url: String
    let postedAt: Date?
    let firstDiscoveredAt: Date
    let band: Band?
    let score: Int?
    let breakdown: [ScoreRow]
    let description: String
    var companyKind: CompanyKind

    var companyName: String { companySlug.replacing("-", with: " ").capitalized }
    var effectiveAge: Date { postedAt ?? firstDiscoveredAt }

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case title = "Title"
        case companyID = "CompanyID"
        case companySlug = "CompanySlug"
        case location = "Location"
        case workArrangement = "WorkArrangement"
        case salaryRaw = "SalaryRaw"
        case url = "URL"
        case postedAt = "PostedAt"
        case firstDiscoveredAt = "FirstDiscoveredAt"
        case band = "Band"
        case score = "SuitabilityScore"
        case breakdown = "Breakdown"
        case description = "Description"
        case companyKind = "CompanyKind"
    }
}

extension Job {
    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        title = try c.decode(String.self, forKey: .title)
        companyID = try c.decode(String.self, forKey: .companyID)
        companySlug = try c.decode(String.self, forKey: .companySlug)
        location = try c.decode(String.self, forKey: .location)
        workArrangement = try c.decode(String.self, forKey: .workArrangement)
        salaryRaw = try c.decode(String.self, forKey: .salaryRaw)
        url = try c.decode(String.self, forKey: .url)
        postedAt = try c.decodeIfPresent(Date.self, forKey: .postedAt)
        firstDiscoveredAt = try c.decode(Date.self, forKey: .firstDiscoveredAt)
        band = try c.decodeIfPresent(String.self, forKey: .band).flatMap { Band(rawValue: $0.lowercased()) }
        score = try c.decodeIfPresent(Int.self, forKey: .score)
        breakdown = try c.decodeIfPresent([ScoreRow].self, forKey: .breakdown) ?? []
        description = try c.decode(String.self, forKey: .description)
        companyKind = (try? c.decode(CompanyKind.self, forKey: .companyKind)) ?? .unclassified
    }
}

extension Job {
    static let sample = Job(
        id: "sample",
        title: "Senior Go Engineer",
        companyID: "company-monzo",
        companySlug: "monzo",
        location: "London",
        workArrangement: "Hybrid",
        salaryRaw: "£85,000–£100,000",
        url: "https://example.com/jobs/1",
        postedAt: Date(timeIntervalSinceNow: -2 * 86_400),
        firstDiscoveredAt: Date(timeIntervalSinceNow: -2 * 86_400),
        band: .great,
        score: 92,
        breakdown: [
            ScoreRow(key: "work:remote", label: "Remote UK", effect: .meets),
            ScoreRow(key: "tech:go", label: "Go", effect: .meets),
            ScoreRow(key: "domain:fintech", label: "Fintech", effect: .misses),
        ],
        description: "Own services end to end, from design through on-call.",
        companyKind: .direct
    )
}
