import Foundation

struct DraftRoles: Decodable {
    let draftID: String
    let kept: Bool
    let spills: Bool
    let roles: [Role]

    struct Role: Decodable, Identifiable {
        let positionID: String?
        let employer: String
        let title: String
        let budget: Int
        let bullets: [Bullet]
        let bank: [Bullet]
        var excluded: [Bullet] = []

        var id: String { positionID ?? "\(employer)|\(title)" }
        var isLinked: Bool { positionID != nil }
        var used: Int { bullets.reduce(0) { $0 + $1.lines } }
        var over: Int { max(used - budget, 0) }
        var free: Int { max(budget - used, 0) }
        var rankedBank: [Bullet] { bank.sorted { ($0.lean ?? -1) > ($1.lean ?? -1) } }

        func fits(_ new: Bullet) -> Bool { used + new.lines <= budget }
        func linesAfter(replacing old: Bullet, with new: Bullet) -> Int { used - old.lines + new.lines }
        func budgetLabel(forLines lines: Int) -> String { lines > budget ? "\(lines - budget) over" : "\(lines)/\(budget)" }

        func replacing(_ old: Bullet, with new: Bullet) -> [Bullet] { bullets.map { $0.text == old.text ? new : $0 } }
        func adding(_ new: Bullet) -> [Bullet] { bullets + [new] }
        func removing(_ old: Bullet) -> [Bullet] { bullets.filter { $0.text != old.text } }

        func moving(_ bullet: Bullet, by offset: Int) -> [Bullet] {
            guard let i = bullets.firstIndex(where: { $0.text == bullet.text }), bullets.indices.contains(i + offset) else { return bullets }
            var moved = bullets
            moved.swapAt(i, i + offset)
            return moved
        }

        func moving(_ bullet: Bullet, onto target: Bullet) -> [Bullet] {
            guard let from = bullets.firstIndex(where: { $0.text == bullet.text }),
                let to = bullets.firstIndex(where: { $0.text == target.text }), from != to
            else { return bullets }
            var moved = bullets
            moved.insert(moved.remove(at: from), at: to)
            return moved
        }

        enum CodingKeys: String, CodingKey {
            case employer, title, budget, bullets, bank, excluded
            case positionID = "positionId"
        }
    }

    struct Bullet: Decodable, Identifiable, Hashable {
        let slotID: String
        let achievementID: String?
        let text: String
        let lines: Int
        let lean: Double?
        let onSource: Bool
        let replacedFrom: String?

        var id: String { slotID.isEmpty ? achievementID ?? text : slotID }
        var fitPercent: Int? { lean.map { Int((($0 + 1) * 50).rounded()) } }

        enum CodingKeys: String, CodingKey {
            case text, lines, lean, onSource, replacedFrom
            case slotID = "slotId"
            case achievementID = "achievementId"
        }
    }

    var linesOver: Int { roles.reduce(0) { $0 + $1.over } }
    var canKeep: Bool { !kept && !spills && linesOver == 0 }

    enum CodingKeys: String, CodingKey {
        case kept, spills, roles
        case draftID = "draftId"
    }
}

extension DraftRoles {
    static let sample = DraftRoles(
        draftID: "draft-1", kept: false, spills: false,
        roles: [
            Role(
                positionID: "position-1", employer: "Acme", title: "Engineer", budget: 3,
                bullets: [
                    Bullet(
                        slotID: "s0", achievementID: "achievement-2", text: "Cut p99 latency by moving queries to Postgres",
                        lines: 1, lean: 0.5, onSource: false, replacedFrom: "Built and maintained the public APIs"),
                    Bullet(
                        slotID: "s1", achievementID: "achievement-3", text: "Led the payments migration", lines: 2, lean: -0.3,
                        onSource: true, replacedFrom: nil),
                ],
                bank: [
                    Bullet(
                        slotID: "", achievementID: "achievement-1", text: "Mentored four engineers", lines: 1, lean: 0.2,
                        onSource: false, replacedFrom: nil)
                ])
        ])
}
