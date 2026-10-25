import SwiftUI

@main
struct FastTrackApp: App {
    @State private var session = Session(client: .live)

    var body: some Scene {
        WindowGroup {
            #if DEBUG
            if ProcessInfo.processInfo.arguments.contains("-prototype-triage") {
                TriagePrototype()
            } else {
                root
            }
            #else
            root
            #endif
        }
    }

    private var root: some View {
        RootView()
            .environment(session)
            .environment(\.apiClient, .live)
    }
}
