import Foundation
import Testing
@testable import FastTrack

struct NotificationsModelTests {
    private final class Calls {
        var read: [String] = []
        var readAll = 0
    }

    private func item(_ id: String, read: Bool = false) -> AppNotification {
        AppNotification(
            id: id, jobID: "job-\(id)", title: "Title \(id)", body: "Body", createdAt: .now,
            readAt: read ? .now : nil)
    }

    private func model(_ items: [AppNotification], calls: Calls = Calls(), failMark: Bool = false) async -> NotificationsModel {
        var client = APIClient.mock(notifications: items)
        client.markNotificationRead = { id in
            if failMark { throw APIError.status(500) }
            calls.read.append(id)
        }
        client.markAllNotificationsRead = {
            if failMark { throw APIError.status(500) }
            calls.readAll += 1
        }
        let model = NotificationsModel(client: client)
        await model.load()
        return model
    }

    @Test func badgeCountsUnread() async {
        let model = await model([item("1"), item("2", read: true), item("3")])
        #expect(model.badge == 2)
    }

    @Test func markReadClearsOneAndCallsServer() async {
        let calls = Calls()
        let model = await model([item("1"), item("2")], calls: calls)
        await model.markRead("1")
        #expect(model.badge == 1)
        #expect(calls.read == ["1"])
    }

    @Test func markReadFailureRestoresUnread() async {
        let model = await model([item("1")], failMark: true)
        await model.markRead("1")
        #expect(model.badge == 1)
    }

    @Test func markAllReadClearsBadge() async {
        let calls = Calls()
        let model = await model([item("1"), item("2")], calls: calls)
        await model.markAllRead()
        #expect(model.badge == 0)
        #expect(calls.readAll == 1)
    }

    @Test func markReadForUnlistedIDStillCallsServer() async {
        let calls = Calls()
        let model = await model([], calls: calls)
        await model.markRead("deep")
        #expect(calls.read == ["deep"])
    }

    @Test func decodesWireShape() throws {
        let json = #"{"id":"n1","job_id":"j1","title":"T","body":"B","created_at":"2026-01-02T03:04:05Z","read_at":null}"#
        let decoded = try JSONDecoder.api.decode(AppNotification.self, from: Data(json.utf8))
        #expect(decoded.jobID == "j1")
        #expect(decoded.isUnread)
    }
}
