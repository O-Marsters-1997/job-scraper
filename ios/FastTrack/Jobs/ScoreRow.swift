struct ScoreRow: Decodable, Identifiable {
    let key: String
    let label: String
    let effect: String

    var id: String { key }
    var meets: Bool { effect == "meets" }
}
