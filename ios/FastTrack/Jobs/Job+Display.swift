import Foundation

extension Job {
    private static let hiddenEffects: Set<ScoreEffect> = [.neutral, .retired, .unknown, .level]

    var gate: ScoreRow? { breakdown.first { $0.effect.isGate } }

    var topRows: [ScoreRow] {
        let shown = breakdown.filter { !Self.hiddenEffects.contains($0.effect) }
        return Array((shown.filter { $0.effect.isGate } + shown.filter { !$0.effect.isGate }).prefix(3))
    }

    func bandReason(_ scoring: ScoringBands?) -> String? {
        var parts: [String] = []
        if let band, let score, let scoring {
            let cuts = scoring.bands
            let range = switch band {
            case .great: "\(cuts.great) and up"
            case .good: "\(cuts.good)–\(cuts.great - 1)"
            case .fair: "\(cuts.fair)–\(cuts.good - 1)"
            case .poor: "below \(cuts.fair)"
            }
            let source = scoring.fittedAt == nil ? "the default" : "your fitted"
            parts.append("\(band.title) is \(range) in \(source) bands; this job scores \(score).")
        }
        if let gate {
            parts.append(gate.effect == .blocked
                ? "Blocked by \(gate.displayLabel), so it scores 0."
                : "Gated by \(gate.displayLabel), which caps the score.")
        }
        return parts.isEmpty ? nil : parts.joined(separator: " ")
    }

    var age: String { JobAge(since: effectiveAge).label }

    var metaLine: String {
        [location, workArrangement, salaryRaw].filter { !$0.isEmpty }.joined(separator: " · ")
    }

    var accessibilitySummary: String {
        [title, companyName, location, workArrangement, salaryRaw, JobAge(since: effectiveAge).spoken, band?.title, companyKind.badge,
         score.map { "score \($0)" }, gate.map { "gate: \($0.label)" }]
            .compactMap { $0 }
            .filter { !$0.isEmpty }
            .joined(separator: ", ")
    }
}
