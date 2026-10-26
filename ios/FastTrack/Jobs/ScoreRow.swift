struct ScoreRow: Decodable, Hashable {
    let key: String
    let label: String
    let effect: ScoreEffect
}
