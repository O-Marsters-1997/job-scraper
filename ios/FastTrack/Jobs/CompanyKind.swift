enum CompanyKind: String, Codable {
    case unclassified, direct, agency
    case likelyAgency = "likely_agency"
    case likelyDirect = "likely_direct"

    var isAgency: Bool { self == .agency || self == .likelyAgency }

    var badge: String? {
        switch self {
        case .agency: "Agency"
        case .likelyAgency: "Maybe recruiter"
        default: nil
        }
    }
}
