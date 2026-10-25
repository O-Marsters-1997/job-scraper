struct Job: Decodable, Identifiable {
    let id: String
    let title: String
    let companySlug: String
    let location: String
    let url: String
    let band: String
    let score: Int?
    let breakdown: [ScoreRow]?

    var subtitle: String { "\(companySlug) · \(location)" }

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case title = "Title"
        case companySlug = "CompanySlug"
        case location = "Location"
        case url = "URL"
        case band = "Band"
        case score = "SuitabilityScore"
        case breakdown = "Breakdown"
    }
}

extension Job {
    static let sample = Job(
        id: "sample",
        title: "Senior Go Engineer",
        companySlug: "monzo",
        location: "London",
        url: "https://example.com/jobs/1",
        band: "Great",
        score: 92,
        breakdown: [
            ScoreRow(key: "work:remote", label: "Remote UK", effect: "meets"),
            ScoreRow(key: "tech:go", label: "Go", effect: "meets"),
            ScoreRow(key: "domain:fintech", label: "Fintech", effect: "misses"),
        ]
    )
}
