struct ReplayResult: Equatable {
    let synced: Int
    let skipped: Int

    var isEmpty: Bool { synced == 0 && skipped == 0 }
    var note: String { "\(synced) synced, \(skipped) skipped" }
}
