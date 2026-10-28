import Foundation

struct OfflineStore {
    var loadCards: (_ stack: TriageStack) throws -> [Job]
    var saveCards: (_ cards: [Job], _ stack: TriageStack) throws -> Void
    var loadQueue: () throws -> [QueuedAction]
    var saveQueue: ([QueuedAction]) throws -> Void
}

extension OfflineStore {
    static let live = files(in: .applicationSupportDirectory)

    static func files(in directory: URL) -> OfflineStore {
        func cards(_ stack: TriageStack) -> URL {
            directory.appending(path: stack == .direct ? "offline-cards.json" : "offline-cards-\(stack.rawValue).json")
        }
        let queue = directory.appending(path: "offline-queue.json")
        return OfflineStore(
            loadCards: { try load(cards($0)) },
            saveCards: { try save($0, to: cards($1)) },
            loadQueue: { try load(queue) },
            saveQueue: { try save($0, to: queue) }
        )
    }

    static func memory(cards: [Job] = [], queue: [QueuedAction] = []) -> OfflineStore {
        let box = Box(cards: [.direct: cards], queue: queue)
        return OfflineStore(
            loadCards: { box.cards[$0] ?? [] },
            saveCards: { box.cards[$1] = $0 },
            loadQueue: { box.queue },
            saveQueue: { box.queue = $0 }
        )
    }

    private final class Box {
        var cards: [TriageStack: [Job]]
        var queue: [QueuedAction]

        init(cards: [TriageStack: [Job]], queue: [QueuedAction]) {
            self.cards = cards
            self.queue = queue
        }
    }

    private static func load<T: Decodable>(_ url: URL) throws -> [T] {
        guard FileManager.default.fileExists(atPath: url.path) else { return [] }
        return try JSONDecoder.api.decode([T].self, from: Data(contentsOf: url))
    }

    private static func save(_ values: some Encodable, to url: URL) throws {
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        try encoder.encode(values).write(to: url, options: .atomic)
    }
}
