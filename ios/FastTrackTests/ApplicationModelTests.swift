import Foundation
import Testing
@testable import FastTrack

struct ApplicationModelTests {
    private final class Calls {
        var log: [String] = []
    }

    private let calls = Calls()
    private let draft = DueJob.Draft(id: "draft-1", state: .ready, cvTitle: "Backend")

    private func model(roles: DraftRoles = .sample, applyTaskID: String? = "apply-1") -> ApplicationModel {
        var client = APIClient.mock()
        let calls = calls
        client.draftRoles = { _ in roles }
        client.keepDraft = { calls.log.append("keep \($0)") }
        client.discardDraft = { calls.log.append("discard \($0)") }
        client.setTaskDone = { calls.log.append("setTaskDone \($0) \($1)") }
        client.createDraft = { job, cv in
            calls.log.append("createDraft \(job) \(cv.title)")
            return DraftRef(id: "new")
        }
        client.createTask = { calls.log.append("createTask \($0) \($1) \($2)") }
        let job = DueJob(
            jobID: "job-1", title: "Engineer", companySlug: "monzo", band: .great, score: 90, applicationID: "app-1",
            chaseBy: nil, applyTaskID: applyTaskID, tasks: [], draft: draft)
        return ApplicationModel(client: client, job: job, draft: draft)
    }

    private func roles(budget: Int, lines: [Int], spills: Bool = false, kept: Bool = false) -> DraftRoles {
        let bullets = lines.enumerated().map {
            DraftRoles.Bullet(slotID: "s\($0.offset)", achievementID: nil, text: "b", lines: $0.element, lean: nil, onSource: true, replacedFrom: nil)
        }
        let role = DraftRoles.Role(positionID: "p", employer: "Acme", title: "Eng", budget: budget, bullets: bullets)
        return DraftRoles(draftID: "draft-1", kept: kept, spills: spills, roles: [role])
    }

    @Test func goldenRolesDecode() throws {
        let url = try #require(Bundle(for: BundleToken.self).url(forResource: "draft_roles", withExtension: "json"))
        let decoded = try JSONDecoder.api.decode(DraftRoles.self, from: Data(contentsOf: url))

        #expect(decoded.draftID == "draft-1")
        #expect(decoded.kept == false)
        let role = try #require(decoded.roles.first)
        #expect(role.positionID == "position-1")
        #expect(role.employer == "Acme")
        #expect(role.budget == 3)
        let bullet = try #require(role.bullets.first)
        #expect(bullet.slotID == "s0")
        #expect(bullet.onSource == false)
        #expect(bullet.replacedFrom == "Built and maintained the public APIs for the platform")
        #expect(bullet.fitPercent == 75)
    }

    @Test(arguments: [
        (3, [1, 1], false, true),
        (3, [2, 2], false, false),
        (3, [1], true, false),
        (3, [1], false, true),
    ])
    func keepIsLockedWhileOverBudgetOrSpilling(budget: Int, lines: [Int], spills: Bool, canKeep: Bool) async {
        let model = model(roles: roles(budget: budget, lines: lines, spills: spills))
        await model.load()

        #expect(model.canKeep == canKeep)
    }

    @Test func keepThenApplyMarksTheApplyTaskDone() async {
        let model = model()
        await model.load()

        await model.keep()
        await model.markApplied()

        #expect(calls.log == ["keep draft-1", "setTaskDone apply-1 true"])
        #expect(model.exit == .applied)
    }

    @Test func finishAtDeskCreatesAFinishTaskWithTheNote() async {
        let model = model()

        await model.finishAtDesk(note: "tidy dates")

        #expect(calls.log == ["createTask type-finish job-1 tidy dates"])
        #expect(model.exit == .atDesk)
    }

    @Test func discardLeavesNoDraftBehind() async {
        let model = model()

        await model.discard()

        #expect(calls.log == ["discard draft-1"])
        #expect(model.exit == .discarded)
    }

    @Test func redraftDiscardsTheOpenDraftBeforeCreatingTheNewOne() async {
        let model = model()
        let cv = CVTemplate(docID: "doc", tabID: "tab", title: "Platform", isDefault: false)

        await model.redraft(from: cv)

        #expect(calls.log == ["discard draft-1", "createDraft job-1 Platform"])
        #expect(model.exit == .redrafted)
    }

    @Test func offlineFailureDisablesActions() async {
        var client = APIClient.mock()
        client.keepDraft = { _ in throw URLError(.notConnectedToInternet) }
        let job = DueJob.sample
        let model = ApplicationModel(client: client, job: job, draft: draft)

        await model.keep()

        #expect(model.isOffline)
        #expect(model.canKeep == false)
    }
}

private final class BundleToken {}
