import SwiftUI

struct JobHeader: View {
    let job: Job

    var body: some View {
        Section {
            Text(job.title).font(.title2.bold())
            Text("\(job.companyName) · \(job.location)").foregroundStyle(.secondary)
            if let score = job.score {
                LabeledContent("Score") {
                    HStack {
                        if let band = job.band { DotLabel(text: band.title, hue: band.colour) }
                        Text(score, format: .number).monospacedDigit()
                    }
                }
            }
            if let url = URL(string: job.url) {
                Link("Open listing", destination: url)
            }
        }
    }
}
