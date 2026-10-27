import SwiftUI

struct MainTabs: View {
    private enum Tab { case triage, due, notifications }

    @State private var tab = Tab.triage
    @State private var due: DueModel
    @State private var notifications: NotificationsModel
    @Binding var path: [Route]
    @Binding var openedNotification: String?
    @Environment(\.scenePhase) private var scenePhase
    let client: APIClient

    init(client: APIClient, path: Binding<[Route]>, openedNotification: Binding<String?>) {
        self.client = client
        _path = path
        _openedNotification = openedNotification
        _due = State(initialValue: DueModel(client: client))
        _notifications = State(initialValue: NotificationsModel(client: client))
    }

    var body: some View {
        TabView(selection: $tab) {
            NavigationStack(path: $path) {
                TriageView(client: client)
                    .navigationDestination(for: Route.self) { route in
                        switch route {
                        case .job(let id): JobDetailView(id: id)
                        }
                    }
            }
            .tabItem { Label("Triage", systemImage: "rectangle.stack") }
            .tag(Tab.triage)
            DueView(model: due)
                .tabItem { Label("Due", systemImage: "checklist") }
                .badge(due.badge)
                .tag(Tab.due)
            NotificationsView(model: notifications)
                .tabItem { Label("Notifications", systemImage: "bell") }
                .badge(notifications.badge)
                .tag(Tab.notifications)
        }
        .onChange(of: path) { if !path.isEmpty { tab = .triage } }
        .task { await due.load() }
        .task { await notifications.load() }
        .onChange(of: openedNotification, initial: true) {
            guard let id = openedNotification else { return }
            openedNotification = nil
            Task { await notifications.markRead(id) }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { Task { await notifications.load() } }
        }
    }
}
