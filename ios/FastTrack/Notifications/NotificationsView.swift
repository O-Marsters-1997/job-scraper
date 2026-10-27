import SwiftUI

struct NotificationsView: View {
    let model: NotificationsModel
    @State private var openJobID: String?

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Notifications")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button("Mark all read") { Task { await model.markAllRead() } }
                            .disabled(model.badge == 0)
                    }
                }
                .navigationDestination(item: $openJobID) { JobDetailView(id: $0) }
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
            if model.notifications.isEmpty {
                ContentUnavailableView(
                    "No notifications", systemImage: "bell",
                    description: Text("New matching jobs show up here."))
            } else {
                List(model.notifications) { notification in
                    Button { open(notification) } label: { NotificationRow(notification: notification) }
                        .buttonStyle(.plain)
                }
                .refreshable { await model.load() }
            }
        }
    }

    private func open(_ notification: AppNotification) {
        openJobID = notification.jobID
        Task { await model.markRead(notification.id) }
    }
}

private struct NotificationRow: View {
    let notification: AppNotification

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(notification.title)
                .font(.body.weight(notification.isUnread ? .bold : .regular))
            Text(notification.body)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            Text(notification.createdAt, style: .relative)
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
        .accessibilityValue(notification.isUnread ? "Unread" : "")
    }
}

#Preview {
    NotificationsView(model: NotificationsModel(client: .mock()))
}
