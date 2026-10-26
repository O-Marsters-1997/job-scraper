import SwiftUI

struct RootView: View {
    @Environment(Session.self) private var session
    @Environment(\.apiClient) private var client
    @State private var path: [Route] = []

    var body: some View {
        Group {
            switch session.state {
            case .loading:
                ProgressView()
            case .signedOut:
                LoginView()
            case .signedIn:
                NavigationStack(path: $path) {
                    TriageView(client: client)
                        .navigationDestination(for: Route.self, destination: destination)
                }
            case .unreachable(let message):
                ContentUnavailableView {
                    Label("Can't reach FastTrack", systemImage: "wifi.exclamationmark")
                } description: {
                    Text(message)
                } actions: {
                    Button("Try again", action: retry)
                }
            }
        }
        .task { await session.restore() }
        .onOpenURL(perform: open)
    }

    @ViewBuilder
    private func destination(for route: Route) -> some View {
        switch route {
        case .job(let id): JobDetailView(id: id)
        }
    }

    private func open(_ url: URL) {
        guard let route = Route(url: url) else { return }
        path = [route]
    }

    private func retry() {
        Task { await session.restore() }
    }
}
