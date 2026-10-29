import SwiftUI
import WidgetKit

struct MainTabs: View {
    private enum Tab { case triage, due, notifications }

    @State private var tab = Tab.triage
    @State private var triageRefresh = 0
    @State private var due: DueModel
    @State private var notifications: NotificationsModel
    @Binding var path: [Route]
    @Binding var openedNotification: String?
    @Binding var dueFocus: DueFocus?
    @Environment(\.scenePhase) private var scenePhase
    let client: APIClient

    init(client: APIClient, path: Binding<[Route]>, openedNotification: Binding<String?>,
         dueFocus: Binding<DueFocus?>) {
        self.client = client
        _path = path
        _openedNotification = openedNotification
        _dueFocus = dueFocus
        _due = State(initialValue: DueModel(client: client, saveSnapshot: Self.saveSnapshot))
        _notifications = State(initialValue: NotificationsModel(client: client))
    }

    private static func saveSnapshot(_ snapshot: NextUpSnapshot) {
        guard let url = NextUpSnapshot.sharedFileURL, (try? snapshot.write(to: url)) != nil else { return }
        WidgetCenter.shared.reloadTimelines(ofKind: NextUpSnapshot.widgetKind)
    }

    var body: some View {
        TabView(selection: $tab) {
            NavigationStack(path: $path) {
                TriageView(client: client, refresh: triageRefresh)
                    .navigationDestination(for: Route.self) { route in
                        switch route {
                        case .job(let id): JobDetailView(id: id)
                        }
                    }
            }
            .tabItem { Label("Triage", systemImage: "rectangle.stack") }
            .tag(Tab.triage)
            DueView(model: due, focus: $dueFocus)
                .tabItem { Label("Due", systemImage: "checklist") }
                .badge(due.badge)
                .tag(Tab.due)
            NotificationsView(model: notifications, onDecided: {
                triageRefresh += 1
                Task { await due.load() }
            })
                .tabItem { Label("Notifications", systemImage: "bell") }
                .badge(notifications.badge)
                .tag(Tab.notifications)
        }
        .onChange(of: dueFocus, initial: true) { if dueFocus != nil { tab = .due } }
        .onOpenURL { if $0 == DueFocus.dueTabURL { tab = .due } }
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
