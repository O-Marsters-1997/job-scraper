import SwiftUI

struct JobDetailView: View {
    let id: String
    @Environment(\.apiClient) private var client
    @State private var state: LoadState<Job> = .idle

    var body: some View {
        ScrollView {
            if let job = state.value {
                VStack(alignment: .leading, spacing: 28) {
                    VStack(alignment: .leading, spacing: 12) {
                        BandRow(job: job)
                        MetaLines(job: job, limitsTitle: false)
                    }
                    JobDetailBody(job: job)
                    JobNoteSection(jobID: job.id)
                    RecruiterContactsSection(jobID: job.id)
                }
                .padding(16)
            }
        }
        .overlay { LoadStateOverlay(state: state, failureTitle: "Couldn't load job") }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if let url = state.value.flatMap({ URL(string: $0.url) }) {
                ToolbarItem(placement: .primaryAction) { ShareLink(item: url) }
            }
        }
        .task(id: id) { await load() }
    }

    private func load() async {
        state = .loading
        do {
            state = .loaded(try await client.job(id))
        } catch is CancellationError {
            return
        } catch {
            state = .failed(error)
        }
    }
}

#Preview {
    NavigationStack { JobDetailView(id: Job.sample.id) }
        .environment(\.apiClient, .mock())
}
