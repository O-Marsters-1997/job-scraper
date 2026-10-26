enum GradeReason: String, CaseIterable, Codable {
    case seniority, role, tech, domain, location, salary, contract
    case companySize = "company_size"
    case recruiter, culture, other
}
