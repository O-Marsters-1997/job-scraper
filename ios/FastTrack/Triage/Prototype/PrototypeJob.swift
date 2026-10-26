#if DEBUG
import SwiftUI

struct PrototypeJob: Identifiable, Hashable {
    enum Band: String { case great = "Great", good = "Good", fair = "Fair", poor = "Poor" }
    enum Effect { case gated, meets, misses, boost, level, unknown }
    struct Row: Hashable { let label: String; let effect: Effect }

    let id: String
    let title: String
    let company: String
    let location: String
    let arrangement: String
    let salary: String?
    let ageDays: Int
    let band: Band
    let score: Int
    let source: String
    let rows: [Row]
    let description: String

    var topRows: [Row] {
        let gates = rows.filter { $0.effect == .gated }
        return Array((gates + rows.filter { $0.effect != .gated }).prefix(3))
    }
    var gate: Row? { rows.first { $0.effect == .gated } }
    var unknownCount: Int { rows.filter { $0.effect == .unknown }.count }
    var age: String { ageDays < 7 ? "\(ageDays)d" : "\(ageDays / 7)w" }

    var accessibilitySummary: String {
        [title, company, location, arrangement, salary, "\(ageDays) days old", band.rawValue, "score \(score)",
         gate.map { "gate: \($0.label)" }].compactMap { $0 }.joined(separator: ", ")
    }
}

extension PrototypeJob {
    private static let lorem = """
        We're looking for an engineer to own services end to end, from design through on-call. You'll work in a \
        small team shipping weekly, with real say over architecture.

        What you'll do
        • Design and run Go services handling millions of requests a day
        • Pair with product on scoping and trade-offs
        • Improve our deploy pipeline and observability

        What we're looking for
        • 5+ years building backend systems
        • Comfortable with Postgres, queues and distributed tracing
        • Happy to mentor and be mentored
        """

    static let fixtures: [PrototypeJob] = [
        .init(id: "1", title: "Senior Backend Engineer", company: "Monzo", location: "London", arrangement: "Hybrid",
              salary: "£85,000–£100,000", ageDays: 2, band: .great, score: 88, source: "greenhouse",
              rows: [.init(label: "Go", effect: .meets), .init(label: "Fintech", effect: .boost),
                     .init(label: "Series B–D", effect: .meets), .init(label: "Senior", effect: .level),
                     .init(label: "On-call", effect: .unknown)], description: lorem),
        .init(id: "2", title: "Platform Engineer (Contract, 6 months)", company: "Cleo", location: "Remote UK",
              arrangement: "Remote", salary: "£600/day", ageDays: 1, band: .fair, score: 61, source: "ashby",
              rows: [.init(label: "Contract role", effect: .gated), .init(label: "Kubernetes", effect: .meets),
                     .init(label: "Remote UK", effect: .meets), .init(label: "Go", effect: .misses)], description: lorem),
        .init(id: "3", title: "Staff Software Engineer, Payments Infrastructure and Reliability", company: "GoCardless",
              location: "London", arrangement: "Hybrid", salary: nil, ageDays: 5, band: .good, score: 76, source: "lever",
              rows: [.init(label: "Payments", effect: .boost), .init(label: "Staff level", effect: .misses),
                     .init(label: "Go", effect: .meets), .init(label: "Salary", effect: .unknown),
                     .init(label: "Visa", effect: .unknown)], description: lorem),
        .init(id: "4", title: "Full Stack Engineer", company: "Tide", location: "Manchester", arrangement: "On-site",
              salary: "£60,000–£70,000", ageDays: 9, band: .poor, score: 34, source: "workable",
              rows: [.init(label: "Manchester on-site", effect: .gated), .init(label: "React", effect: .misses),
                     .init(label: "Fintech", effect: .boost)], description: lorem),
        .init(id: "5", title: "Backend Engineer, Growth", company: "Lottie", location: "London", arrangement: "Hybrid",
              salary: "£70,000–£85,000", ageDays: 3, band: .good, score: 72, source: "ashby",
              rows: [.init(label: "Go", effect: .meets), .init(label: "Seed–Series A", effect: .misses),
                     .init(label: "Healthtech", effect: .level)], description: lorem),
        .init(id: "6", title: "Senior Go Developer", company: "Paddle", location: "Remote UK", arrangement: "Remote",
              salary: "£90,000–£110,000", ageDays: 14, band: .great, score: 91, source: "greenhouse",
              rows: [.init(label: "Go", effect: .meets), .init(label: "Remote UK", effect: .meets),
                     .init(label: "Senior", effect: .meets), .init(label: "SaaS", effect: .boost)], description: lorem),
    ]
}

extension PrototypeJob.Band {
    var colour: Color {
        switch self {
        case .great: Color(hex: 0x059669)
        case .good: Color(hex: 0x10b981)
        case .fair: Color(hex: 0xd97706)
        case .poor: Color(hex: 0xdc2626)
        }
    }
}

extension PrototypeJob.Effect {
    var symbol: String {
        switch self {
        case .gated: "xmark.octagon.fill"
        case .meets: "checkmark.circle"
        case .misses: "xmark.circle"
        case .boost: "star.circle"
        case .level: "equal.circle"
        case .unknown: "questionmark.circle"
        }
    }
    var colour: Color {
        switch self {
        case .gated, .misses: Color(hex: 0xdc2626)
        case .meets, .boost: Color(hex: 0x059669)
        case .level, .unknown: .secondary
        }
    }
    var groupTitle: String {
        switch self {
        case .gated: "Gates"
        case .boost: "Boosts"
        case .meets: "Met"
        case .misses: "Missed"
        case .level: "Level"
        case .unknown: "Unknown"
        }
    }
}

extension Color {
    init(hex: UInt32) {
        self.init(red: Double((hex >> 16) & 0xff) / 255, green: Double((hex >> 8) & 0xff) / 255, blue: Double(hex & 0xff) / 255)
    }

    static let brandTint = Color(UIColor { $0.userInterfaceStyle == .dark
        ? UIColor(red: 0x7b / 255, green: 0x5c / 255, blue: 0xf0 / 255, alpha: 1)
        : UIColor(red: 0x66 / 255, green: 0x45 / 255, blue: 0xd9 / 255, alpha: 1) })
}
#endif
