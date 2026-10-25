import SwiftUI

struct JobsView: View {
    @Environment(\.apiClient) private var client
    @State private var state: LoadState<[Job]> = .idle

    var body: some View {
        List(state.value ?? []) { job in
            NavigationLink(value: Route.job(id: job.id)) {
                JobRow(job: job)
            }
        }
        .overlay {
            if state.value?.isEmpty == true {
                ContentUnavailableView("No scored jobs yet", systemImage: "tray", description: Text("Jobs appear here once they've been scored."))
            } else {
                LoadStateOverlay(state: state, failureTitle: "Couldn't load jobs")
            }
        }
        .navigationTitle("Jobs")
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        if state.value == nil { state = .loading }
        do {
            state = .loaded(try await client.jobs())
        } catch is CancellationError {
            return
        } catch {
            state = .failed(error)
        }
    }
}

#Preview("Loaded") {
    NavigationStack { JobsView() }
        .environment(\.apiClient, .mock())
}

#Preview("Failed") {
    NavigationStack { JobsView() }
        .environment(\.apiClient, .mock(failure: .status(500)))
}
