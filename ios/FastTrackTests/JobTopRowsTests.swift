import Testing
@testable import FastTrack

struct JobTopRowsTests {
    private func job(_ rows: [ScoreRow]) -> Job {
        var job = Job.sample
        job = Job(
            id: job.id, title: job.title, companyID: job.companyID, companySlug: job.companySlug,
            location: job.location, workArrangement: job.workArrangement, salaryRaw: job.salaryRaw,
            url: job.url, postedAt: job.postedAt, firstDiscoveredAt: job.firstDiscoveredAt,
            band: job.band, score: job.score, breakdown: rows, description: job.description,
            companyKind: job.companyKind
        )
        return job
    }

    @Test func unmetNiceToHaveIsNeverShown() {
        let rows = [ScoreRow(key: "role:backend", label: "Backend", effect: .misses, stance: "nice")]
        #expect(job(rows).topRows.isEmpty)
    }

    @Test func avoidHitShowsAndNeutralAvoidDoesNot() {
        let hit = ScoreRow(key: "domain:gambling", label: "Gambling", effect: .misses, stance: "avoid")
        let clear = ScoreRow(key: "domain:crypto", label: "Crypto", effect: .neutral, stance: "avoid")
        #expect(job([clear, hit]).topRows.map(\.key) == ["domain:gambling"])
    }

    @Test func ordersGateThenAvoidThenFavouriteThenMatchesCappedAtThree() {
        let rows = [
            ScoreRow(key: "tech:go", label: "Go", effect: .meets),
            ScoreRow(key: "company:fav", label: "Favourite", effect: .favourite),
            ScoreRow(key: "domain:gambling", label: "Gambling", effect: .misses, stance: "avoid"),
            ScoreRow(key: "gate:visa", label: "Visa", effect: .gated),
            ScoreRow(key: "work:hybrid", label: "Hybrid", effect: .meets),
        ]
        #expect(job(rows).topRows.map(\.key) == ["gate:visa", "domain:gambling", "company:fav"])
    }
}
