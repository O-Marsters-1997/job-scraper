import SwiftUI

struct RootView: View {
    @Environment(Session.self) private var session
    @Environment(\.apiClient) private var client
    @State private var path: [Route] = []
    @State private var openedNotification: String?
    @State private var dueFocus: DueFocus?

    var body: some View {
        Group {
            switch session.state {
            case .loading:
                ProgressView()
            case .signedOut:
                LoginView()
            case .signedIn:
                MainTabs(client: client, path: $path, openedNotification: $openedNotification,
                         dueFocus: $dueFocus)
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

    private func open(_ url: URL) {
        guard let route = Route(url: url) else {
            if let focus = DueFocus(url: url) { dueFocus = focus }
            return
        }
        path = [route]
        openedNotification = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems?.first { $0.name == "n" }?.value
    }

    private func retry() {
        Task { await session.restore() }
    }
}
