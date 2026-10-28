import SwiftUI

@main
struct FastTrackApp: App {
    @State private var session = Session(client: .live)

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(session)
                .environment(\.apiClient, .live)
                .tint(.brandTint)
        }
    }
}
