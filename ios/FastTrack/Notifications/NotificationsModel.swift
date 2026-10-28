import Observation
import SwiftUI

@Observable
final class NotificationsModel {
    private(set) var notifications: [AppNotification] = []
    private(set) var loadState: LoadState<Void> = .idle
    private(set) var failureMessage: String?

    private let client: APIClient
    private let now: () -> Date

    init(client: APIClient, now: @escaping () -> Date = { .now }) {
        self.client = client
        self.now = now
    }

    var badge: Int { notifications.filter(\.isUnread).count }

    var groups: [(String, [AppNotification])] {
        let calendar = Calendar.current
        let today = calendar.startOfDay(for: now())
        let yesterday = calendar.date(byAdding: .day, value: -1, to: today) ?? today
        let newest = notifications.sorted { $0.createdAt > $1.createdAt }
        return [
            ("Today", newest.filter { $0.createdAt >= today }),
            ("Yesterday", newest.filter { $0.createdAt >= yesterday && $0.createdAt < today }),
            ("Earlier", newest.filter { $0.createdAt < yesterday }),
        ].filter { !$0.1.isEmpty }
    }

    func load() async {
        if notifications.isEmpty { loadState = .loading }
        do {
            notifications = try await client.notifications()
            loadState = .loaded(())
        } catch is CancellationError {
            return
        } catch {
            if notifications.isEmpty { loadState = .failed(error) }
        }
    }

    func markRead(_ id: String) async {
        let previous = notifications
        guard let index = notifications.firstIndex(where: { $0.id == id }), notifications[index].isUnread else {
            try? await client.markNotificationRead(id)
            return
        }
        notifications[index] = notifications[index].markedRead()
        do {
            try await client.markNotificationRead(id)
        } catch {
            notifications = previous
        }
    }

    func decide(_ action: TriageAction, on job: Job) async throws {
        _ = try await client.perform(action.kind(on: job), jobID: job.id)
        if case .shortlist(draftCV: true) = action,
            let cv = try? await client.cvTemplates().first(where: \.isDefault)
        {
            _ = try? await client.createDraft(job.id, cv)
        }
        await load()
    }

    func note(for job: Job) async throws -> String {
        try await client.note(job.id).body
    }

    func delete(_ id: String) async {
        let previous = notifications
        notifications.removeAll { $0.id == id }
        do {
            try await client.deleteNotification(id)
        } catch {
            notifications = previous
            failureMessage = "The notification couldn't be deleted: \(error.localizedDescription)"
        }
    }

    func perform(_ action: TriageAction, on job: Job) async {
        do {
            _ = try await client.perform(action.kind(on: job), jobID: job.id)
        } catch {
            failureMessage = error.localizedDescription
            return
        }
        guard case .shortlist(draftCV: true) = action else { return }
        do {
            guard let cv = try await client.cvTemplates().first(where: \.isDefault) else {
                failureMessage = "There's no default CV, so the job was shortlisted without a draft."
                return
            }
            _ = try await client.createDraft(job.id, cv)
        } catch {
            failureMessage = "The job was shortlisted, but the CV draft didn't start: \(error.localizedDescription)"
        }
    }

    func job(_ id: String) async -> Job? {
        do {
            return try await client.job(id)
        } catch {
            failureMessage = "The job couldn't be loaded: \(error.localizedDescription)"
            return nil
        }
    }

    var hasFailure: Bool {
        get { failureMessage != nil }
        set { if !newValue { failureMessage = nil } }
    }

    func markAllRead() async {
        let previous = notifications
        notifications = notifications.map { $0.markedRead() }
        do {
            try await client.markAllNotificationsRead()
        } catch {
            notifications = previous
        }
    }
}

private extension AppNotification {
    func markedRead() -> AppNotification {
        AppNotification(
            id: id, kind: kind, jobID: jobID, title: title, body: body, job: job, createdAt: createdAt,
            readAt: readAt ?? .now)
    }
}
