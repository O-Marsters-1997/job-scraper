import Foundation

extension Job {
    private static let hiddenEffects: Set<ScoreEffect> = [.neutral, .retired, .unknown, .level]

    var gate: ScoreRow? { breakdown.first { $0.effect.isGate } }

    var topRows: [ScoreRow] {
        let shown = breakdown.filter { !Self.hiddenEffects.contains($0.effect) }
        return Array((shown.filter { $0.effect.isGate } + shown.filter { !$0.effect.isGate }).prefix(3))
    }

    var ageDays: Int { max(Calendar.current.dateComponents([.day], from: effectiveAge, to: .now).day ?? 0, 0) }
    var age: String { ageDays < 7 ? "\(ageDays)d" : "\(ageDays / 7)w" }

    var metaLine: String {
        [location, workArrangement, salaryRaw].filter { !$0.isEmpty }.joined(separator: " · ")
    }

    var accessibilitySummary: String {
        [title, companyName, location, workArrangement, salaryRaw, "\(ageDays) days old", band?.title,
         score.map { "score \($0)" }, gate.map { "gate: \($0.label)" }]
            .compactMap { $0 }
            .filter { !$0.isEmpty }
            .joined(separator: ", ")
    }
}
