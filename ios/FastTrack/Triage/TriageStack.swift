enum TriageStack: String, Codable, CaseIterable {
    case direct, agency
}

extension Job {
    var stack: TriageStack { companyKind.isAgency ? .agency : .direct }
}
