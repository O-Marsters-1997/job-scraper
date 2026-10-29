import Foundation

extension Job {
    var gate: ScoreRow? { breakdown.first { $0.effect.isGate } }

    var topRows: [ScoreRow] {
        let negatives = breakdown.filter { $0.effect.isGate }
            + breakdown.filter { $0.stance == "avoid" && $0.effect == .misses }
        let positives = breakdown.filter { $0.effect == .favourite } + breakdown.filter { $0.effect == .meets }
        return Array((negatives + positives).prefix(3))
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
            parts.append("\(band.title) is \(range); this job scores \(score).")
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
