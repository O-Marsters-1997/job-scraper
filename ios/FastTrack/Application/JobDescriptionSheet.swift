import SwiftUI

struct JobDescriptionSheet: View {
    let job: Job?

    var body: some View {
        NavigationStack {
            ScrollView {
                if let job {
                    VStack(alignment: .leading, spacing: 20) {
                        MetaLines(job: job, limitsTitle: false)
                        JobDetailBody(job: job)
                    }
                    .padding(16)
                }
            }
            .overlay {
                if job == nil { ContentUnavailableView("Job unavailable", systemImage: "exclamationmark.triangle") }
            }
            .navigationTitle("Job")
            .navigationBarTitleDisplayMode(.inline)
        }
        .presentationDetents([.fraction(0.45), .large])
        .presentationBackgroundInteraction(.enabled(upThrough: .fraction(0.45)))
    }
}
