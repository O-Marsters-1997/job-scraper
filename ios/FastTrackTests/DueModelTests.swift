import Foundation
import Testing
@testable import FastTrack

private actor LoadCounter {
    private(set) var value = 0
    func next() -> Int {
        defer { value += 1 }
        return value
    }
}

struct DueModelTests {
    private final class Calls {
        var log: [String] = []
        var failClear = false
    }

    private let calls = Calls()
    private let today = Calendar.current.startOfDay(for: Date(timeIntervalSince1970: 1_790_000_000))

    private func day(_ offset: Int) -> Date {
        Calendar.current.date(byAdding: .day, value: offset, to: today)!
    }

    private func job(
        _ id: String, chase: Int?, tasks: [String] = [], draft: DueJob.DraftState? = nil, applyTaskID: String? = nil,
        score: Int? = 90, title: String? = nil
    ) -> DueJob {
        DueJob(
            jobID: id, title: title ?? "Role \(id)", companySlug: "co-\(id)", companyKind: .direct, band: .great, score: score, applicationID: "app-\(id)",
            chaseBy: chase.map(day), applyTaskID: applyTaskID,
            tasks: tasks.map { DueJob.Task(id: $0, typeName: "Reach out", systemKey: nil, notes: "") },
            draft: draft.map { DueJob.Draft(id: "d-\(id)", state: $0, cvTitle: "CV") })
    }

    private func model(_ jobs: [DueJob]) async -> DueModel {
        var client = APIClient.mock(due: jobs)
        let calls = calls
        client.clearChase = { id in
            if calls.failClear { throw APIError.status(500) }
            calls.log.append("clearChase \(id)")
        }
        client.setChase = { id, date in calls.log.append("setChase \(id) \(date.dayString)") }
        client.setTaskDone = { id, done in calls.log.append("setTaskDone \(id) \(done)") }
        client.createDraft = { id, cv in
            calls.log.append("createDraft \(id) \(cv.title)")
            return DraftRef(id: "new")
        }
        let model = DueModel(client: client, now: { [today] in today.addingTimeInterval(3_600) })
        await model.load()
        return model
    }

    @Test func badgeCountsDueChasesAndReadyDraftsNotTasks() async {
        let model = await model([
            job("a", chase: -1), job("b", chase: 0, tasks: ["t1", "t2"]), job("c", chase: 3), job("d", chase: nil, draft: .ready),
            job("e", chase: nil, draft: .pending),
        ])

        #expect(model.badge == 3)
        #expect(model.comingUp.map(\.id) == ["c"])
        #expect(model.dueJobs.map(\.id) == ["a", "b", "d", "e"])
    }

    @Test func refreshWhilePendingReloadsUntilDraftSettles() async {
        let responses = [
            [job("a", chase: nil, draft: .pending)], [job("a", chase: nil, draft: .pending)], [job("a", chase: nil, draft: .ready)],
        ]
        let loads = LoadCounter()
        var client = APIClient.mock()
        client.due = { responses[min(await loads.next(), responses.count - 1)] }
        let model = DueModel(client: client)
        await model.load()
        #expect(model.hasPendingDraft)

        await model.refreshWhilePending(every: .milliseconds(1))

        #expect(!model.hasPendingDraft)
        #expect(await loads.value == 3)
    }

    @Test func badgeIsHiddenWhenLoadFails() async {
        let model = DueModel(client: .mock(due: [job("a", chase: 0)], failure: .status(500)))
        await model.load()

        #expect(model.badge == 0)
        #expect(model.loadState.value == nil)
    }

    @Test func completeChaseClearsItAndUndoRestoresTheOldDate() async {
        let model = await model([job("a", chase: -2)])

        await model.completeChase(model.jobs[0]).value
        #expect(model.badge == 0)
        await model.undo().value

        #expect(calls.log == ["clearChase app-a", "setChase app-a \(day(-2).dayString)"])
        #expect(model.badge == 1)
    }

    @Test func snoozeMovesTheChaseAndUndoPutsTheOldDateBack() async {
        let model = await model([job("a", chase: -1)])

        await model.snooze(model.jobs[0], days: 7).value
        #expect(model.dueJobs.isEmpty)
        #expect(model.comingUp.map(\.id) == ["a"])
        await model.undo().value

        #expect(calls.log == ["setChase app-a \(day(7).dayString)", "setChase app-a \(day(-1).dayString)"])
        #expect(model.dueJobs.map(\.id) == ["a"])
    }

    @Test func completingATaskUndoesToNotDone() async {
        let model = await model([job("a", chase: nil, tasks: ["t1", "t2"])])

        await model.complete(model.jobs[0].tasks[0], in: model.jobs[0]).value
        #expect(model.jobs[0].tasks.map(\.id) == ["t2"])
        await model.undo().value

        #expect(calls.log == ["setTaskDone t1 true", "setTaskDone t1 false"])
        #expect(model.jobs[0].tasks.map(\.id) == ["t1", "t2"])
    }

    @Test func markAppliedTicksTheApplyTaskKeepsTheDraftAndUndoUnticks() async {
        let model = await model([job("a", chase: nil, tasks: ["apply"], draft: .ready, applyTaskID: "apply")])

        await model.markApplied(model.jobs[0]).value
        #expect(model.jobs[0].tasks.isEmpty)
        #expect(model.jobs[0].draft?.state == .ready)
        await model.undo().value

        #expect(calls.log == ["setTaskDone apply true", "setTaskDone apply false"])
        #expect(model.jobs[0].tasks.map(\.id) == ["apply"])
    }

    @Test func aJobWithOnlyAnApplyTaskIsShortlistedNotDue() async {
        let model = await model([job("a", chase: nil, applyTaskID: "apply"), job("b", chase: nil, draft: .ready)])
        #expect(model.dueJobs.map(\.id) == ["b"])
        #expect(model.shortlisted.map(\.id) == ["a"])

        await model.markApplied(model.jobs[0]).value
        #expect(model.shortlisted.isEmpty)
        await model.undo().value

        #expect(calls.log == ["setTaskDone apply true", "setTaskDone apply false"])
        #expect(model.shortlisted.map(\.id) == ["a"])
    }

    @Test func shortlistedExcludesDraftedAndAtDeskJobsAndSortsByScoreDescending() async {
        let atDesk = DueJob(
            jobID: "desk", title: "Role desk", companySlug: "co-desk", companyKind: .direct, band: .great, score: 99, applicationID: "app-desk",
            chaseBy: nil, applyTaskID: "apply-desk",
            tasks: [DueJob.Task(id: "f", typeName: "Finish application", systemKey: "finish", notes: "")], draft: nil)
        let model = await model([
            job("low", chase: nil, applyTaskID: "x", score: 60), job("none", chase: nil, applyTaskID: "x", score: nil),
            job("high", chase: nil, applyTaskID: "x", score: 95), job("drafted", chase: nil, draft: .ready, applyTaskID: "x"),
            job("noTask", chase: nil), atDesk,
        ])

        #expect(model.shortlisted.map(\.id) == ["high", "low", "none"])
        #expect(model.dueJobs.map(\.id) == ["drafted", "desk"])
    }

    @Test func readyToApplyHoldsKeptUnappliedJobsAndLeavesNotDrafted() async throws {
        func kept(_ id: String, apply: String?, score: Int) -> DueJob {
            DueJob(
                jobID: id, title: "Role \(id)", companySlug: "co-\(id)", companyKind: .direct, band: .great, score: score, applicationID: "app-\(id)",
                chaseBy: nil, applyTaskID: apply, tasks: [], draft: nil, keptDraftID: "kept-\(id)")
        }
        let model = await model([
            kept("low", apply: "x", score: 60), kept("high", apply: "x", score: 95), kept("applied", apply: nil, score: 99),
            job("plain", chase: nil, applyTaskID: "x"),
        ])

        #expect(model.readyToApply.map(\.id) == ["high", "low"])
        #expect(try #require(model.readyToApply.first).keptDraft?.id == "kept-high")
        #expect(model.shortlisted.map(\.id) == ["plain"])
        #expect(model.shortlistedCount == 3)
    }

    @Test(arguments: [
        ("engineer", ["a"], ["c"]), ("MONZO", ["b"], []), ("  go ", ["a", "b"], ["c"]), ("", ["a", "b"], ["c"]),
        ("zzz", [], []),
    ] as [(String, [String], [String])])
    func searchMatchesTitleAndCompanyOnBothLists(query: String, shortlisted: [String], due: [String]) async {
        let model = await model([
            job("a", chase: nil, applyTaskID: "x", score: 90, title: "Senior Go Engineer"),
            DueJob(
                jobID: "b", title: "Go Developer", companySlug: "monzo", companyKind: .direct, band: .great, score: 80, applicationID: "app-b",
                chaseBy: nil, applyTaskID: "x", tasks: [], draft: nil),
            job("c", chase: nil, draft: .ready, title: "Senior Go Engineer"),
        ])

        #expect(model.shortlisted(matching: query).map(\.id) == shortlisted)
        #expect(model.dueJobs(matching: query).map(\.id) == due)
    }

    @Test func deleteRemovesTheApplicationAndRestoresItOnFailure() async {
        var client = APIClient.mock(due: [job("a", chase: 0), job("b", chase: 0)])
        let calls = calls
        client.deleteApplication = { id in
            calls.log.append("delete \(id)")
            if id == "app-b" { throw APIError.status(500) }
        }
        let model = DueModel(client: client, now: { [today] in today.addingTimeInterval(3_600) })
        await model.load()
        await model.delete(model.jobs[0])
        #expect(model.jobs.map(\.id) == ["b"])
        await model.delete(model.jobs[0])
        #expect(model.jobs.map(\.id) == ["b"])
        #expect(model.hasFailure == true)
        #expect(calls.log == ["delete app-a", "delete app-b"])
    }

    @Test func hasCVCountsDraftsAndKeptDrafts() {
        var kept = job("k", chase: 0)
        kept.keptDraftID = "x"
        #expect(job("d", chase: 0, draft: .ready).hasCV == true)
        #expect(kept.hasCV == true)
        #expect(job("n", chase: 0).hasCV == false)
    }

    @Test func markAppliedWithoutAnApplyTaskDoesNothing() async {
        let model = await model([job("a", chase: nil, draft: .ready)])

        await model.markApplied(model.jobs[0]).value

        #expect(calls.log.isEmpty)
    }

    @Test func retryingAFailedDraftStartsOneFromTheDefaultCV() async {
        let model = await model([job("a", chase: nil, draft: .failed)])

        await model.retryDraft(model.jobs[0])

        #expect(calls.log == ["createDraft a Backend"])
    }

    @Test func undoingAnEarlierTaskKeepsALaterOneDone() async {
        let model = await model([job("a", chase: nil, tasks: ["t1", "t2", "t3"])])

        await model.complete(model.jobs[0].tasks[0], in: model.jobs[0]).value
        await model.complete(model.jobs[0].tasks[0], in: model.jobs[0]).value
        await model.undo().value

        #expect(model.jobs[0].tasks.map(\.id) == ["t2", "t3"])
        await model.undo().value
        #expect(model.jobs[0].tasks.map(\.id) == ["t1", "t2", "t3"])
    }

    @Test func loggingContactOnADueChaseClearsIt() async {
        let model = await model([job("a", chase: -1)])

        let logged = await model.logContact(model.jobs[0], channel: .email, outcome: .noReply)

        #expect(logged)
        #expect(model.badge == 0)
        #expect(calls.log == ["clearChase app-a"])
    }

    @Test func failedActionRestoresTheJob() async {
        let model = await model([job("a", chase: 0)])
        calls.failClear = true

        await model.completeChase(model.jobs[0]).value

        #expect(model.badge == 1)
        #expect(model.undoStack.isEmpty)
        #expect(model.hasFailure)
    }
}
