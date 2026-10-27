import SwiftUI

struct MainTabs: View {
    private enum Tab { case triage, due, prototype }

    @State private var tab = Tab.triage
    @State private var due: DueModel
    @Binding var path: [Route]
    let client: APIClient

    init(client: APIClient, path: Binding<[Route]>) {
        self.client = client
        _path = path
        _due = State(initialValue: DueModel(client: client))
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
            #if DEBUG
            AgencyStackPrototype()
                .tabItem { Label("Prototype", systemImage: "flask") }
                .tag(Tab.prototype)
            #endif
        }
        .onChange(of: path) { if !path.isEmpty { tab = .triage } }
        .task { await due.load() }
    }
}
