import SwiftUI

struct JobRow: View {
    let job: Job

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(job.title).font(.headline)
                Spacer()
                if let score = job.score {
                    Text(score, format: .number)
                        .font(.headline.monospacedDigit())
                        .accessibilityLabel("Score \(score)")
                }
            }
            Text(job.subtitle)
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
    }
}

#Preview {
    List { JobRow(job: .sample) }
}
