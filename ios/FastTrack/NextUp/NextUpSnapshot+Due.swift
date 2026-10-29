import Foundation

extension NextUpSnapshot {
    init(jobs: [DueJob], at: Date) {
        self.init(writtenAt: at, items: jobs.compactMap { job in
            let ready = job.dueDraft?.state == .ready
            guard job.chaseBy != nil || ready else { return nil }
            return Item(jobID: job.jobID, title: job.title, companyName: job.companyName, score: job.score, chaseBy: job.chaseBy, readyToApply: ready)
        })
    }
}
