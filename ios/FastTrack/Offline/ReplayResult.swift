struct ReplayResult: Equatable {
    let synced: Int
    let skipped: Int

    var note: String { "\(synced) synced, \(skipped) skipped" }
}
