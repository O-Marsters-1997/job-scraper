import Foundation
import Testing
@testable import FastTrack

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
        _ id: String, chase: Int?, tasks: [String] = [], draft: DueJob.DraftState? = nil, applyTaskID: String? = nil
    ) -> DueJob {
        DueJob(
            jobID: id, title: "Role \(id)", companySlug: "co-\(id)", band: .great, score: 90, applicationID: "app-\(id)",
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
