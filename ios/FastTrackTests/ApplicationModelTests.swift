import Foundation
import Testing
@testable import FastTrack

struct ApplicationModelTests {
    private final class Calls {
        var log: [String] = []
        var kept = false
    }

    private let calls = Calls()
    private let draft = DueJob.Draft(id: "draft-1", state: .ready, cvTitle: "Backend")

    private func model(roles: DraftRoles = .sample, applyTaskID: String? = "apply-1") -> ApplicationModel {
        var client = APIClient.mock()
        let calls = calls
        client.draftRoles = { _ in DraftRoles(draftID: roles.draftID, kept: calls.kept, spills: roles.spills, roles: roles.roles) }
        client.keepDraft = {
            calls.log.append("keep \($0)")
            calls.kept = true
        }
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
        let bank = DraftRoles.Bullet(
            slotID: "", achievementID: "bank-1", text: "banked", lines: 1, lean: 0.4, onSource: false, replacedFrom: nil)
        let role = DraftRoles.Role(positionID: "p", employer: "Acme", title: "Eng", budget: budget, bullets: bullets, bank: [bank])
        return DraftRoles(draftID: "draft-1", kept: kept, spills: spills, roles: [role])
    }

    private func recordingModel(lines: [Int], budget: Int = 3) -> ApplicationModel {
        var client = APIClient.mock()
        let calls = calls
        let roles = roles(budget: budget, lines: lines)
        client.draftRoles = { _ in roles }
        client.saveRole = { draft, position, bullets in
            calls.log.append("save \(draft) \(position) " + bullets.map { "\($0.text):\($0.achievementIDs.joined(separator: ","))" }.joined(separator: "|"))
        }
        let job = DueJob.sample
        return ApplicationModel(client: client, job: job, draft: draft)
    }

    @Test func replaceTakesTheOldSlotAndSendsTheFullList() async throws {
        let model = recordingModel(lines: [1, 1])
        await model.load()
        let role = try #require(model.roles.value?.roles.first)

        await model.replace(role.bullets[0], with: role.bank[0], in: role)

        #expect(calls.log == ["save draft-1 p banked:bank-1|b:"])
    }

    @Test func addPutsTheBankBulletInTheFreeLines() async throws {
        let model = recordingModel(lines: [1, 1])
        await model.load()
        let role = try #require(model.roles.value?.roles.first)

        await model.add(role.bank[0], to: role)

        #expect(calls.log == ["save draft-1 p b:|b:|banked:bank-1"])
    }

    @Test func removeAndMoveSendTheResultingOrder() async throws {
        let model = recordingModel(lines: [1, 2])
        await model.load()
        let role = try #require(model.roles.value?.roles.first)

        await model.remove(role.bullets[0], from: role)
        await model.move(role.bullets[0], by: 1, in: role)

        #expect(calls.log == ["save draft-1 p b:", "save draft-1 p b:|b:"])
    }

    @Test func removingTheLastBulletSendsNothing() async throws {
        let model = recordingModel(lines: [1])
        await model.load()
        let role = try #require(model.roles.value?.roles.first)

        await model.remove(role.bullets[0], from: role)

        #expect(calls.log.isEmpty)
        #expect(model.hasFailure)
    }

    @Test func sheetBudgetLabelsMatchTheRoleAfterTheEdit() throws {
        let role = try #require(roles(budget: 3, lines: [2, 1]).roles.first)
        let bigger = DraftRoles.Bullet(
            slotID: "", achievementID: "x", text: "x", lines: 3, lean: nil, onSource: false, replacedFrom: nil)

        let after = role.linesAfter(replacing: role.bullets[0], with: bigger)

        #expect(role.budgetLabel(forLines: after) == "1 over")
        #expect(role.budgetLabel(forLines: role.linesAfter(replacing: role.bullets[0], with: role.bank[0])) == "2/3")
        #expect(role.fits(role.bank[0]) == false)
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
        #expect(model.kept)
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

    @Test func requestingALetterRecordsItOnce() async {
        var client = APIClient.mock()
        let calls = calls
        client.requestCoverLetter = {
            calls.log.append("letter \($0)")
            return CoverLetter(id: "l1", createdAt: .now, doneAt: nil)
        }
        let model = ApplicationModel(client: client, job: .sample, draft: draft)

        await model.requestLetter()

        #expect(calls.log == ["letter \(DueJob.sample.id)"])
        #expect(model.letters.count == 1)
        #expect(model.letterError == nil)
    }

    @Test func rejectedLetterRequestShowsServerMessageInline() async {
        var client = APIClient.mock()
        client.requestCoverLetter = { _ in throw APIError.unprocessable("No letter tab") }
        let model = ApplicationModel(client: client, job: .sample, draft: draft)

        await model.requestLetter()

        #expect(model.letters.isEmpty)
        #expect(model.letterError == "No letter tab")
        #expect(model.hasFailure == false)
    }

    @Test func offlineFailureDisablesActions() async {
        var client = APIClient.mock()
        client.keepDraft = { _ in throw URLError(.notConnectedToInternet) }
        let job = DueJob.sample
        let model = ApplicationModel(client: client, job: job, draft: draft)
        await model.load()
        #expect(model.canKeep)

        await model.keep()

        #expect(model.isOffline)
        #expect(model.canKeep == false)
    }
}

private final class BundleToken {}
