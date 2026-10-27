import SwiftUI

struct DueView: View {
    let model: DueModel
    @State private var contactFor: DueJob?
    @State private var openJobID: String?
    @State private var openApplication: DueJob?
    @State private var query = ""
    @State private var showingShortlisted = false

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Due")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Button("Undo", systemImage: "arrow.uturn.backward", action: undo)
                            .disabled(model.undoStack.isEmpty)
                    }
                }
                .searchable(text: $query)
                .navigationDestination(item: $openJobID) { JobDetailView(id: $0) }
                .navigationDestination(isPresented: $showingShortlisted) { ShortlistedView(model: model) }
                .navigationDestination(item: $openApplication) { job in
                    if let draft = job.draft {
                        ApplicationScreen(job: job, draft: draft, onFinished: { await model.load() })
                    }
                }
                .safeAreaInset(edge: .bottom, spacing: 8) {
                    DueUndoBanner(entry: model.banner, onUndo: undo, onExpire: model.dismissBanner)
                        .animation(.snappy, value: model.banner?.id)
                }
                .sheet(item: $contactFor) { job in
                    ContactSheet { await model.logContact(job, channel: $0, outcome: $1) }
                }
                .alert("Couldn't save that", isPresented: Bindable(model).hasFailure) {
                    Button("OK") {}
                } message: {
                    Text(model.failureMessage)
                }
                .task { await model.load() }
        }
    }

    private func undo() {
        _ = withAnimation { model.undo() }
    }

    @ViewBuilder
    private var content: some View {
        switch model.loadState {
        case .idle, .loading:
            ProgressView()
        case .failed:
            ContentUnavailableView {
                Label("Due list needs a connection", systemImage: "wifi.slash")
            } description: {
                Text("Triage keeps working offline.")
            } actions: {
                Button("Try again") { Task { await model.load() } }
            }
        case .loaded:
            let jobs = model.dueJobs(matching: query)
            if model.dueJobs.isEmpty && model.comingUp.isEmpty {
                VStack {
                    ContentUnavailableView(
                        "Nothing due", systemImage: "checkmark.circle",
                        description: Text("Chases, open Tasks and ready CV drafts show up here."))
                    NotDraftedRow(count: model.shortlisted.count) { showingShortlisted = true }
                        .padding(.horizontal, 16)
                }
            } else if jobs.isEmpty && !query.isEmpty {
                ContentUnavailableView.search(text: query)
            } else {
                DueNotebook(model: model, jobs: jobs, showShortlisted: { showingShortlisted = true },
                            logContact: { contactFor = $0 }, openJob: { openJobID = $0.id },
                            openApplication: { openApplication = $0 })
            }
        }
    }
}

#Preview {
    DueView(model: DueModel(client: .mock(due: [.sample])))
}
