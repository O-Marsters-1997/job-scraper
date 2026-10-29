struct ScoreRow: Codable, Hashable {
    let key: String
    let label: String
    let effect: ScoreEffect
    var impact: Double? = nil

    private static let levels = ["Junior", "Mid", "Senior", "Lead / Staff", "Principal / Head"]

    private static func levelName(_ value: Substring) -> String {
        let index = min(max(Int((Double(value) ?? 1).rounded()), 1), levels.count) - 1
        return levels[index]
    }

    var displayLabel: String {
        guard let m = label.firstMatch(of: /^Seniority: job ≈ ([\d.]+) · you ([\d.]+)/) else { return label }
        let job = Self.levelName(m.1)
        return effect == .meets ? "Seniority fits: \(job)" : "Seniority: \(job) role, you aim for \(Self.levelName(m.2))"
    }
}
