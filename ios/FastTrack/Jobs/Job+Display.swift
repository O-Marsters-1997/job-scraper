import Foundation

extension Job {
    private static let hiddenEffects: Set<ScoreEffect> = [.neutral, .retired]

    var gate: ScoreRow? { breakdown.first { $0.effect.isGate } }
    var unknownCount: Int { breakdown.filter { $0.effect == .unknown }.count }

    var topRows: [ScoreRow] {
        let shown = breakdown.filter { !Self.hiddenEffects.contains($0.effect) }
        return Array((shown.filter { $0.effect.isGate } + shown.filter { !$0.effect.isGate }).prefix(3))
    }

    var age: String { JobAge(since: effectiveAge).label }

    var metaLine: String {
        [location, workArrangement, salaryRaw].filter { !$0.isEmpty }.joined(separator: " · ")
    }

    var accessibilitySummary: String {
        [title, companyName, location, workArrangement, salaryRaw, JobAge(since: effectiveAge).spoken, band?.title,
         score.map { "score \($0)" }, gate.map { "gate: \($0.label)" }]
            .compactMap { $0 }
            .filter { !$0.isEmpty }
            .joined(separator: ", ")
    }
}
