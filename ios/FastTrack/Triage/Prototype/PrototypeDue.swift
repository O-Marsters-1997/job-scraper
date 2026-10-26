#if DEBUG
import SwiftUI

struct PrototypeChase: Identifiable, Hashable {
    let id = UUID()
    let job: PrototypeJob
    var chaseBy: Date
    var overdueDays: Int {
        let cal = Calendar.current
        return max(cal.dateComponents([.day], from: cal.startOfDay(for: chaseBy), to: cal.startOfDay(for: .now)).day ?? 0, 0)
    }
}

struct PrototypeTask: Identifiable, Hashable {
    let id = UUID()
    let type: String
    let job: PrototypeJob
}

struct PrototypeDraft: Identifiable, Hashable {
    enum State { case pending, ready, failed }
    let id = UUID()
    let job: PrototypeJob
    let template: String
    var state: State
}

struct PrototypeContact: Identifiable, Hashable {
    let id = UUID()
    let name: String
    let channel: String
    let date: Date
}

extension PrototypeJob {
    static let applied: [PrototypeJob] = [
        .init(id: "a1", title: "Senior Platform Engineer", company: "Wise", location: "London", arrangement: "Hybrid",
              salary: "£95,000–£115,000", ageDays: 20, band: .great, score: 86, source: "greenhouse",
              rows: [.init(label: "Go", effect: .meets), .init(label: "Fintech", effect: .boost)], description: fixtures[0].description),
        .init(id: "a2", title: "Backend Engineer", company: "Octopus Energy", location: "London", arrangement: "Hybrid",
              salary: "£70,000–£90,000", ageDays: 25, band: .good, score: 74, source: "workable",
              rows: [.init(label: "Python", effect: .misses), .init(label: "Climate", effect: .boost)], description: fixtures[0].description),
        .init(id: "a3", title: "Software Engineer, Infrastructure", company: "Synthesia", location: "Remote UK", arrangement: "Remote",
              salary: nil, ageDays: 12, band: .good, score: 70, source: "ashby",
              rows: [.init(label: "Kubernetes", effect: .meets), .init(label: "Salary", effect: .unknown)], description: fixtures[0].description),
        .init(id: "a4", title: "Go Engineer", company: "Marshmallow", location: "London", arrangement: "Hybrid",
              salary: "£80,000–£95,000", ageDays: 6, band: .great, score: 83, source: "lever",
              rows: [.init(label: "Go", effect: .meets), .init(label: "Insurtech", effect: .level)], description: fixtures[0].description),
    ]
}

enum DueFixtures {
    static func chases() -> [PrototypeChase] {
        let day: TimeInterval = 86_400
        return [
            .init(job: PrototypeJob.applied[0], chaseBy: .now - 3 * day),
            .init(job: PrototypeJob.applied[1], chaseBy: .now - 1 * day),
            .init(job: PrototypeJob.applied[2], chaseBy: .now),
        ]
    }

    static func tasks() -> [PrototypeTask] {
        [
            .init(type: "Reach out", job: PrototypeJob.applied[0]),
            .init(type: "Prep", job: PrototypeJob.applied[2]),
            .init(type: "Apply", job: PrototypeJob.applied[3]),
        ]
    }

    static func drafts() -> [PrototypeDraft] {
        [
            .init(job: PrototypeJob.applied[3], template: "Backend · 2 page", state: .ready),
            .init(job: PrototypeJob.applied[1], template: "Default", state: .failed),
        ]
    }
}

extension Date {
    var shortGB: String { formatted(.dateTime.day().month(.abbreviated).locale(Locale(identifier: "en_GB"))) }
}
#endif
