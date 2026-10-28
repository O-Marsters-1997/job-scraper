import SwiftUI

struct JobDescriptionSheet: View {
    let job: Job?
    var actions: AnyView?

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
            .toolbar {
                if let actions {
                    ToolbarItem(placement: .topBarTrailing) {
                        Menu { actions } label: { Label("Job actions", systemImage: "ellipsis") }
                    }
                }
            }
        }
        .presentationDetents([.fraction(0.45), .large])
        .presentationBackgroundInteraction(.enabled(upThrough: .fraction(0.45)))
    }
}
