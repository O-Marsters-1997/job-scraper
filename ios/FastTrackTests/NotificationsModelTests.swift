import Foundation
import Testing
@testable import FastTrack

struct NotificationsModelTests {
    private final class Calls {
        var read: [String] = []
        var readAll = 0
        var excluded: [String] = []
        var loads = 0
        var deleted: [String] = []
    }

    private let today = Calendar.current.startOfDay(for: Date(timeIntervalSince1970: 1_790_000_000))

    private func item(_ id: String, read: Bool = false, at createdAt: Date = .now) -> AppNotification {
        AppNotification(
            id: id, kind: .newMatch, jobID: "job-\(id)", title: "Title \(id)", body: "Body",
            job: NotificationJob(title: "Role", companyName: "Monzo", location: "", salaryRaw: "", band: nil, score: nil),
            createdAt: createdAt, readAt: read ? .now : nil)
    }

    private func model(
        _ items: [AppNotification], calls: Calls = Calls(), failMark: Bool = false, now: Date = .now
    ) async -> NotificationsModel {
        var client = APIClient.mock(notifications: items)
        client.markNotificationRead = { id in
            if failMark { throw APIError.status(500) }
            calls.read.append(id)
        }
        client.markAllNotificationsRead = {
            if failMark { throw APIError.status(500) }
            calls.readAll += 1
        }
        client.deleteNotification = { id in
            if failMark { throw APIError.status(500) }
            calls.deleted.append(id)
        }
        let model = NotificationsModel(client: client, now: { now })
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

    @Test func jobActionSendsTheActionAndReportsFailure() async {
        var shortlisted: [String] = []
        var client = APIClient.mock()
        client.shortlist = { jobID in
            shortlisted.append(jobID)
            return Application(id: "a", jobID: jobID)
        }
        let ok = NotificationsModel(client: client)
        await ok.perform(.shortlist(draftCV: false), on: .sample)
        #expect(shortlisted == [Job.sample.id])
        #expect(ok.hasFailure == false)

        let failing = NotificationsModel(client: .mock(failure: .status(500)))
        await failing.perform(.skip, on: .sample)
        #expect(failing.hasFailure)
    }

    @Test func deleteRemovesRowAndUnreadBadgeAndCallsServer() async {
        let calls = Calls()
        let model = await model([item("1"), item("2", read: true)], calls: calls)
        await model.delete("1")
        #expect(model.notifications.map(\.id) == ["2"])
        #expect(model.badge == 0)
        #expect(calls.deleted == ["1"])
    }

    @Test func deleteFailureRestoresRowAndReportsIt() async {
        let model = await model([item("1")], failMark: true)
        await model.delete("1")
        #expect(model.notifications.map(\.id) == ["1"])
        #expect(model.failureMessage != nil)
    }

    @Test func markReadForUnlistedIDStillCallsServer() async {
        let calls = Calls()
        let model = await model([], calls: calls)
        await model.markRead("deep")
        #expect(calls.read == ["deep"])
    }

    @Test func groupsSplitAtDayBoundariesNewestFirst() async {
        let items = [
            item("earlier", at: today.addingTimeInterval(-86_401)),
            item("yesterday-start", at: today.addingTimeInterval(-86_400)),
            item("today-start", at: today),
            item("yesterday-end", at: today.addingTimeInterval(-1)),
            item("today-late", at: today.addingTimeInterval(3_600)),
        ]
        let model = await model(items, now: today.addingTimeInterval(7_200))

        #expect(model.groups.map(\.0) == ["Today", "Yesterday", "Earlier"])
        #expect(model.groups.map { $0.1.map(\.id) } == [
            ["today-late", "today-start"], ["yesterday-end", "yesterday-start"], ["earlier"],
        ])
    }

    @Test func decideSendsActionForTheJobAndReloads() async throws {
        let calls = Calls()
        var client = APIClient.mock(notifications: [item("1")])
        client.excludeCompany = { companyID in
            calls.excluded.append(companyID)
            return CompanyExclusion(name: "Monzo", added: true)
        }
        client.notifications = {
            calls.loads += 1
            return []
        }
        let model = NotificationsModel(client: client)
        try await model.decide(.excludeCompany, on: .sample)
        #expect(calls.excluded == [Job.sample.companyID])
        #expect(calls.loads == 1)
    }

    @Test func decideFailureThrows() async {
        let model = NotificationsModel(client: .mock(failure: .status(500)))
        await #expect(throws: APIError.status(500)) { try await model.decide(.skip, on: .sample) }
    }

    @Test func groupsOmitEmptyDays() async {
        let model = await model([item("old", at: today.addingTimeInterval(-10 * 86_400))], now: today)
        #expect(model.groups.map(\.0) == ["Earlier"])
    }

    @Test func decodesWireShape() throws {
        let json = #"""
        {"id":"n1","kind":"new_match","job_id":"j1","title":"T","body":"B",
         "job":{"title":"Senior Go Engineer","company_name":"Monzo","location":"London","salary_raw":"£90k","band":"great","score":92},
         "created_at":"2026-01-02T03:04:05Z","read_at":null}
        """#
        let decoded = try JSONDecoder.api.decode(AppNotification.self, from: Data(json.utf8))
        #expect(decoded.kind == .newMatch)
        #expect(decoded.jobID == "j1")
        #expect(decoded.isUnread)
        #expect(decoded.job.companyName == "Monzo")
        #expect(decoded.job.facts == "London · £90k")
        #expect(decoded.job.band == .great)
        #expect(decoded.job.score == 92)
    }

    @Test func unknownKindAndUnscoredJobStillDecode() throws {
        let json = #"""
        {"id":"n1","kind":"weekly_digest","job_id":"j1","title":"Digest","body":"12 new jobs",
         "job":{"title":"","company_name":"","location":"","salary_raw":"","band":"","score":null},
         "created_at":"2026-01-02T03:04:05Z","read_at":null}
        """#
        let decoded = try JSONDecoder.api.decode(AppNotification.self, from: Data(json.utf8))
        #expect(decoded.kind == .unknown)
        #expect(decoded.title == "Digest")
        #expect(decoded.job.band == nil)
        #expect(decoded.job.score == nil)
        #expect(decoded.job.facts == "")
    }
}
