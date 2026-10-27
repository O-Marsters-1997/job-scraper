import Foundation

struct ScoringBands: Decodable, Hashable {
    struct Cuts: Decodable, Hashable {
        let great: Int
        let good: Int
        let fair: Int
    }

    let bands: Cuts
    let fittedAt: Date?
}
