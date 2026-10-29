import Foundation
import Testing
@testable import FastTrack

struct JobTopRowsTests {
    private func job(_ breakdown: [ScoreRow]) -> Job {
        let s = Job.sample
        return Job(id: s.id, title: s.title, companyID: s.companyID, companySlug: s.companySlug, location: s.location,
                   workArrangement: s.workArrangement, salaryRaw: s.salaryRaw, url: s.url, postedAt: s.postedAt,
                   firstDiscoveredAt: s.firstDiscoveredAt, band: s.band, score: s.score, breakdown: breakdown,
                   description: s.description, companyKind: s.companyKind)
    }

    @Test func gateThenFavouriteThenLargestImpactWithinCap() {
        let rows = [
            ScoreRow(key: "a", label: "A", effect: .meets, impact: 0.2),
            ScoreRow(key: "b", label: "B", effect: .misses, impact: -2),
            ScoreRow(key: "fav", label: "Fav", effect: .favourite),
            ScoreRow(key: "c", label: "C", effect: .meets, impact: 1),
            ScoreRow(key: "gate", label: "G", effect: .gated),
        ]

        #expect(job(rows).topRows.map(\.key) == ["gate", "fav", "b"])
    }

    @Test func missingImpactKeepsStoredOrder() throws {
        let json = #"[{"key":"a","label":"A","effect":"meets"},{"key":"b","label":"B","effect":"meets"}]"#
        let rows = try JSONDecoder().decode([ScoreRow].self, from: Data(json.utf8))

        #expect(job(rows).topRows.map(\.key) == ["a", "b"])
    }
}
