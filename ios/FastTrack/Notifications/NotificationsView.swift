import SwiftUI

struct NotificationsView: View {
    let model: NotificationsModel
    var onDecided: () -> Void = {}
    @State private var openJobID: String?
    @State private var preview: AppNotification?
    @State private var suppressedTap: String?
    @State private var actionsJob: Job?

    var body: some View {
        NavigationStack {
            content
                .navigationBarTitleDisplayMode(.inline)
                .navigationDestination(item: $openJobID) { JobDetailView(id: $0) }
                .sheet(item: $preview, onDismiss: { suppressedTap = nil }) {
                    JobPreviewSheet(id: $0.jobID, model: model, onDecided: onDecided)
                }
                .sensoryFeedback(.impact(weight: .medium), trigger: preview) { _, new in new != nil }
                .notificationJobActions(job: $actionsJob, model: model)
                .alert("Couldn't do that", isPresented: Bindable(model).hasFailure) {
                    Button("OK") {}
                } message: {
                    Text(model.failureMessage ?? "")
                }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch model.loadState {
        case .idle, .loading:
            ProgressView()
        case .failed:
            ContentUnavailableView {
                Label("Notifications need a connection", systemImage: "wifi.slash")
            } actions: {
                Button("Try again") { Task { await model.load() } }
            }
        case .loaded:
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    header
                    ForEach(model.groups, id: \.0) { group in
                        Text(group.0)
                            .font(.headline)
                            .accessibilityAddTraits(.isHeader)
                            .padding(.horizontal, 16)
                            .padding(.top, 20)
                            .padding(.bottom, 4)
                        ForEach(group.1) { row($0) }
                    }
                }
                .padding(.bottom, 24)
            }
            .refreshable { await model.load() }
            .overlay {
                if model.notifications.isEmpty {
                    ContentUnavailableView(
                        "No notifications", systemImage: "bell",
                        description: Text("New matching jobs show up here."))
                }
            }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("Notifications")
                .font(.system(.title, design: .serif))
                .accessibilityAddTraits(.isHeader)
            HStack(alignment: .firstTextBaseline) {
                Text(model.badge > 0 ? "Unread (\(model.badge))" : "All caught up")
                    .font(.subheadline.weight(.medium).monospacedDigit())
                    .contentTransition(.numericText())
                Spacer()
                Button("Mark all as read") { Task { await model.markAllRead() } }
                    .font(.subheadline)
                    .disabled(model.badge == 0)
            }
        }
        .padding(.horizontal, 16)
        .padding(.top, 8)
    }

    private func row(_ notification: AppNotification) -> some View {
        Button { open(notification) } label: { NotificationRow(notification: notification) }
            .buttonStyle(.plain)
            .simultaneousGesture(
                LongPressGesture(minimumDuration: 0.35).onEnded { _ in
                    suppressedTap = notification.id
                    preview = notification
                }
            )
            .accessibilityAction(named: "Preview job") { preview = notification }
            .iconSwipeActions([
                SwipeAction(title: "More", systemImage: "ellipsis") {
                    Task { actionsJob = await model.job(notification.jobID) }
                },
                SwipeAction(title: "Delete notification", systemImage: "trash", isDestructive: true) {
                    Task { await model.delete(notification.id) }
                },
            ])
    }

    private func open(_ notification: AppNotification) {
        if suppressedTap == notification.id {
            suppressedTap = nil
            return
        }
        openJobID = notification.jobID
        Task { await model.markRead(notification.id) }
    }
}

private struct NotificationRow: View {
    let notification: AppNotification
    @ScaledMetric(relativeTo: .caption2) private var badgeSize = 17.0

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            HStack(alignment: .top, spacing: 8) {
                Circle()
                    .fill(notification.isUnread ? Color.brandTint : .clear)
                    .frame(width: 7, height: 7)
                    .padding(.top, 16)
                avatar
            }
            VStack(alignment: .leading, spacing: 5) {
                sentence
                if let facts {
                    Text(facts)
                        .font(.subheadline.monospacedDigit())
                        .foregroundStyle(.secondary)
                }
                if let band = notification.job.band, let score = notification.job.score {
                    DotLabel(text: "\(band.title) · \(score)", hue: band.colour)
                        .padding(.top, 2)
                }
                Text(age)
                    .font(.footnote)
                    .foregroundStyle(.tertiary)
                    .padding(.top, 2)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
        .accessibilityValue(notification.isUnread ? "Unread" : "")
    }

    private var facts: String? {
        let text = notification.kind == .unknown ? notification.body : notification.job.facts
        return text.isEmpty ? nil : text
    }

    private var age: String {
        let text = notification.createdAt.formatted(.relative(presentation: .named))
        return text.prefix(1).uppercased() + text.dropFirst()
    }

    private var sentence: Text {
        guard let verb = notification.kind.verb else { return Text(notification.title) }
        let company = Text(notification.job.companyName).fontWeight(.semibold)
        let role = Text(notification.job.title).fontWeight(.semibold)
        return Text("\(company) \(Text(verb).foregroundStyle(.secondary)) \(role)")
    }

    private var avatar: some View {
        let name = notification.job.companyName.isEmpty ? notification.title : notification.job.companyName
        return Text(String(name.prefix(1)))
            .font(.system(.headline, design: .serif))
            .frame(width: 40, height: 40)
            .background(Color(.secondarySystemBackground), in: .circle)
            .overlay(Circle().strokeBorder(Color(.separator)))
            .overlay(alignment: .bottomTrailing) {
                Image(systemName: notification.kind.symbol)
                    .font(.caption2.bold())
                    .imageScale(.small)
                    .foregroundStyle(.white)
                    .frame(width: badgeSize, height: badgeSize)
                    .background(Color.brandTint, in: .circle)
                    .overlay(Circle().strokeBorder(Color(.systemBackground), lineWidth: 2))
                    .offset(x: 3, y: 3)
            }
            .accessibilityHidden(true)
    }
}

private struct JobPreviewSheet: View {
    let id: String
    let model: NotificationsModel
    let onDecided: () -> Void
    @Environment(\.apiClient) private var client
    @Environment(\.dismiss) private var dismiss
    @State private var job: Job?
    @State private var loaded = false
    @State private var showReasons = false
    @State private var showNote = false
    @State private var failed = false

    var body: some View {
        Group {
            if loaded {
                JobDescriptionSheet(job: job, actions: job.map { menu(for: $0) })
            } else {
                ProgressView()
                    .presentationDetents([.fraction(0.45), .large])
            }
        }
        .sheet(isPresented: $showReasons) {
            DismissReasonSheet { decide(.dismiss(reasons: $0)) }
        }
        .sheet(isPresented: $showNote) {
            if let job {
                NoteSheet(load: { try await model.note(for: job) }) { decide(.note($0)) }
            }
        }
        .alert("Couldn't save that", isPresented: $failed) { Button("OK") {} }
        .task {
            job = try? await client.job(id)
            loaded = true
        }
    }

    private func menu(for job: Job) -> some View {
        JobActionsMenu(
            job: job, onDecide: decide, onDismissWithReason: { showReasons = true }, onAddNote: { showNote = true })
    }

    private func decide(_ action: TriageAction) {
        guard let job else { return }
        Task {
            do {
                try await model.decide(action, on: job)
                onDecided()
                if !action.keepsCard {
                    dismiss()
                } else if case .reclassify = action {
                    self.job = try? await client.job(id)
                }
            } catch {
                failed = true
            }
        }
    }
}

#Preview {
    NotificationsView(model: NotificationsModel(client: .mock(notifications: AppNotification.samples)))
        .environment(\.apiClient, .mock())
}
