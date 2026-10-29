import SwiftUI

struct JobDescriptionSheet<Actions: View>: View {
    let job: Job?
    let actions: Actions?

    var body: some View {
        NavigationStack {
            ScrollView {
                if let job {
                    VStack(alignment: .leading, spacing: 20) {
                        HStack(alignment: .firstTextBaseline) {
                            MetaLines(job: job, limitsTitle: false)
                            if let actions {
                                Menu { actions } label: {
                                    Label("Job actions", systemImage: "ellipsis")
                                        .labelStyle(.iconOnly)
                                        .font(.title3)
                                }
                            }
                        }
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

extension JobDescriptionSheet where Actions == EmptyView {
    init(job: Job?) {
        self.init(job: job, actions: nil)
    }
}
