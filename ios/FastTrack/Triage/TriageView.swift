import SwiftUI

struct TriageView: View {
    @State private var model: TriageModel
    @State private var progress: CGFloat = 0
    @State private var expandedID: Job.ID?
    @State private var showReasons = false
    @State private var showNote = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.scenePhase) private var scenePhase
    @Environment(Session.self) private var session

    init(client: APIClient, store: OfflineStore = .live) {
        _model = State(initialValue: TriageModel(client: client, store: store))
    }

    var body: some View {
        content
            .navigationTitle("Triage")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Undo", systemImage: "arrow.uturn.backward", action: undo)
                        .disabled(model.undoStack.isEmpty)
                }
                if model.current != nil {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button("Skip") { decide(.skip) }
                    }
                }
            }
            .safeAreaInset(edge: .top, spacing: 0) {
                OfflineRow(
                    isOffline: model.isOffline,
                    queuedCount: model.queued.count,
                    syncNote: model.syncNote,
                    onSync: sync,
                    onClearNote: model.clearSyncNote)
            }
            .onChange(of: scenePhase) {
                if scenePhase == .active, model.isOffline || !model.queued.isEmpty { sync() }
            }
            .onChange(of: model.sessionExpired) {
                if model.sessionExpired { session.expire() }
            }
            .safeAreaInset(edge: .bottom, spacing: 8) {
                UndoBanner(decision: model.banner, onUndo: undo, onExpire: model.dismissBanner)
                    .animation(.snappy, value: model.banner)
            }
            .sheet(isPresented: $showReasons) {
                DismissReasonSheet { decide(.dismiss(reasons: $0)) }
            }
            .sheet(isPresented: $showNote) {
                NoteSheet(load: model.currentNote) { decide(.note($0)) }
            }
            .alert("Couldn't save that", isPresented: $model.hasFailure) {
                Button("OK") {}
            } message: {
                Text(model.failureMessage)
            }
            .sensoryFeedback(trigger: model.undoStack.map(\.id), feedback)
            .task { if model.loadState.isIdle { await model.load() } }
    }

    @ViewBuilder
    private var content: some View {
        switch model.loadState {
        case .idle, .loading:
            ProgressView()
        case .failed(let error):
            ContentUnavailableView {
                Label("Couldn't load jobs", systemImage: "exclamationmark.triangle")
            } description: {
                Text(error.localizedDescription)
            } actions: {
                Button("Try again") { Task { await model.load() } }
            }
        case .loaded:
            if let job = model.current {
                deck(job)
            } else {
                ContentUnavailableView {
                    Label(
                        model.isOffline ? "No more cached jobs" : "Queue clear",
                        systemImage: model.isOffline ? "wifi.slash" : "checkmark.circle")
                } description: {
                    Text(model.isOffline ? "Reconnect to load more." : "New jobs arrive after the next scrape.")
                } actions: {
                    Button("Check again", action: sync)
                }
            }
        }
    }

    private func deck(_ job: Job) -> some View {
        let isExpanded = expandedID == job.id
        return VStack(spacing: 22) {
            ZStack(alignment: .bottom) {
                DeckEdges(count: model.upcoming.count)
                    .opacity(isExpanded ? 0 : 1)
                TriageCard(job: job, isExpanded: isExpanded) { toggle(job) }
                    .overlay { decisionWash }
                    .swipeToDecide(allowsUp: !isExpanded, onProgress: { progress = $0 }) { direction in
                        switch direction {
                        case .right: decide(.shortlist)
                        case .left: decide(.dismiss(reasons: []))
                        case .up: decide(.skip)
                        }
                    }
                    .id(job.id)
                    .transition(.asymmetric(insertion: insertion, removal: removal))
                    .accessibilityAction(named: "Shortlist") { decide(.shortlist) }
                    .accessibilityAction(named: "Dismiss") { decide(.dismiss(reasons: [])) }
                    .accessibilityAction(named: "Skip") { decide(.skip) }
                    .accessibilityAction(named: isExpanded ? "Collapse details" : "Details") { toggle(job) }
            }
            .padding(.horizontal)
            .padding(.bottom, isExpanded ? 0 : CGFloat(model.upcoming.count) * 6)
            actionBar
                .scaleEffect(isExpanded && !reduceMotion ? 0.9 : 1)
                .opacity(isExpanded ? 0.6 : 1)
        }
        .animation(reduceMotion ? .easeInOut(duration: 0.2) : .spring(duration: 0.4, bounce: 0.15), value: isExpanded)
        .padding(.bottom, 8)
        .onChange(of: model.current?.id) { expandedID = nil }
    }

    private var decisionWash: some View {
        RoundedRectangle(cornerRadius: 14, style: .continuous)
            .fill(progress > 0 ? Color.brandTint : Color(.systemGray))
            .opacity(min(abs(progress), 1) * 0.12)
            .allowsHitTesting(false)
    }

    private var actionBar: some View {
        let right = max(min(progress, 1), 0)
        let left = max(min(-progress, 1), 0)
        return HStack(spacing: 28) {
            Button { decide(.dismiss(reasons: [])) } label: {
                Image(systemName: "xmark")
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(left > 0.99 ? Color.white : Color.secondary)
                    .frame(width: 64, height: 64)
                    .background(Color(.systemGray).opacity(left), in: .circle)
                    .background(Color(.systemBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator)))
                    .scaleEffect(1 + left * 0.15)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Dismiss")
            Menu {
                Button("Dismiss with reason…", systemImage: "text.badge.xmark") { showReasons = true }
                Button("Add note…", systemImage: "note.text") { showNote = true }
                Button("Exclude company", systemImage: "nosign", role: .destructive) { decide(.excludeCompany) }
            } label: {
                Image(systemName: "ellipsis")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .frame(width: 48, height: 48)
                    .background(Color(.systemBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator)))
            }
            .accessibilityLabel("More")
            .tint(.secondary)
            .opacity(Double(1 - max(left, right) * 0.6))
            Button { decide(.shortlist) } label: {
                Image(systemName: "checkmark")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.white)
                    .frame(width: 64, height: 64)
                    .background(Color.brandTint, in: .circle)
                    .scaleEffect(1 + right * 0.15)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Shortlist")
        }
        .dynamicTypeSize(...DynamicTypeSize.xxxLarge)
        .animation(.interactiveSpring, value: progress)
    }

    private func feedback(old: [UUID], new: [UUID]) -> SensoryFeedback? {
        guard new.count > old.count, let last = model.undoStack.last else { return nil }
        switch last.action {
        case .shortlist: return .success
        case .dismiss: return .impact(weight: .light)
        default: return nil
        }
    }

    private func sync() {
        Task { await model.load() }
    }

    private func undo() {
        _ = withAnimation { model.undo() }
    }

    private func decide(_ action: TriageAction) {
        expandedID = nil
        _ = withAnimation(.snappy) { model.decide(action) }
    }

    private func toggle(_ job: Job) {
        withAnimation(reduceMotion ? .easeInOut(duration: 0.2) : .spring(duration: 0.45, bounce: 0.12)) {
            expandedID = expandedID == job.id ? nil : job.id
        }
    }

    private var insertion: AnyTransition {
        reduceMotion ? .opacity : .scale(scale: 0.94).combined(with: .opacity)
    }

    private var removal: AnyTransition {
        guard !reduceMotion, let last = model.lastRemoved else { return .opacity }
        switch last {
        case .shortlist: return .offset(x: 600).combined(with: .opacity)
        case .skip: return .offset(y: -900).combined(with: .opacity)
        default: return .offset(x: -600).combined(with: .opacity)
        }
    }
}

#Preview {
    NavigationStack { TriageView(client: .mock(jobs: [.sample, .sample]), store: .memory()) }
        .environment(Session(client: .mock()))
}
