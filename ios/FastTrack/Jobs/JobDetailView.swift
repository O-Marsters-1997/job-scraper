import SwiftUI

struct JobDetailView: View {
    let id: String
    @Environment(\.apiClient) private var client
    @State private var state: LoadState<Job> = .idle

    var body: some View {
        List {
            if let job = state.value {
                JobHeader(job: job)
                ScoreBreakdownSection(rows: job.breakdown)
            }
        }
        .overlay { LoadStateOverlay(state: state, failureTitle: "Couldn't load job") }
        .navigationBarTitleDisplayMode(.inline)
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
