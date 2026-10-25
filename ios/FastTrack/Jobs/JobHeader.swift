import SwiftUI

struct JobHeader: View {
    let job: Job

    var body: some View {
        Section {
            Text(job.title).font(.title2.bold())
            Text(job.subtitle).foregroundStyle(.secondary)
            if let score = job.score {
                LabeledContent("Score") {
                    Text(job.band.isEmpty ? "\(score)" : "\(score) · \(job.band)")
                }
            }
            if let url = URL(string: job.url) {
                Link("Open listing", destination: url)
            }
        }
    }
}
