import Foundation

struct NextUpSnapshot: Codable, Equatable {
    struct Item: Codable, Equatable {
        let jobID: String
        let title: String
        let companyName: String
        let score: Int?
        let chaseBy: Date?
        let readyToApply: Bool
    }

    let writtenAt: Date
    let items: [Item]

    static let appGroup = "group.com.ollymarsters.fasttrack"
    static let fileName = "next-up.json"
}

extension NextUpSnapshot {
    init(jobs: [DueJob], at: Date) {
        self.init(writtenAt: at, items: jobs.compactMap { job in
            let ready = job.dueDraft?.state == .ready
            guard job.chaseBy != nil || ready else { return nil }
            return Item(jobID: job.jobID, title: job.title, companyName: job.companyName, score: job.score, chaseBy: job.chaseBy, readyToApply: ready)
        })
    }

    func isStale(at: Date) -> Bool {
        at.timeIntervalSince(writtenAt) > 24 * 60 * 60
    }
}
