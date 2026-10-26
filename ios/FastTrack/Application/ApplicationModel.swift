import Observation
import SwiftUI

@Observable
final class ApplicationModel {
    enum Exit { case applied, discarded, redrafted, atDesk }

    let job: DueJob
    let draft: DueJob.Draft
    private(set) var roles: LoadState<DraftRoles> = .idle
    private(set) var listing: Job?
    private(set) var pdfURL: URL?
    private(set) var templates: [CVTemplate] = []
    private(set) var letters: [CoverLetter] = []
    private(set) var requestingLetter = false
    private(set) var letterError: String?
    private(set) var exit: Exit?
    private(set) var isOffline = false
    private(set) var failureMessage = ""
    var hasFailure = false

    private let client: APIClient

    init(client: APIClient, job: DueJob, draft: DueJob.Draft) {
        self.client = client
        self.job = job
        self.draft = draft
    }

    var kept: Bool { roles.value?.kept ?? false }
    var canKeep: Bool { !isOffline && (roles.value?.canKeep ?? false) }
    var pdfName: String { "olly-marsters-\(job.companySlug).pdf" }

    func load() async {
        do {
            roles = .loaded(try await client.draftRoles(draft.id))
            listing = try? await client.job(job.id)
            templates = (try? await client.cvTemplates()) ?? []
            letters = (try? await client.coverLetters(job.id)) ?? []
            await fetchPDF()
            isOffline = false
        } catch is CancellationError {
            return
        } catch {
            noteOffline(error)
            if roles.value == nil { roles = .failed(error) }
        }
    }

    func keep() async {
        await attempt {
            try await client.keepDraft(draft.id)
            roles = .loaded(try await client.draftRoles(draft.id))
        }
    }

    func markApplied() async {
        guard let taskID = job.applyTaskID else { return fail("This application has no apply task.") }
        await attempt {
            try await client.setTaskDone(taskID, true)
            exit = .applied
        }
    }

    func requestLetter() async {
        guard !requestingLetter else { return }
        requestingLetter = true
        letterError = nil
        defer { requestingLetter = false }
        do {
            letters.append(try await client.requestCoverLetter(job.id))
            isOffline = false
        } catch is CancellationError {
            return
        } catch {
            noteOffline(error)
            letterError = error.localizedDescription
        }
    }

    func discard() async {
        await attempt {
            try await client.discardDraft(draft.id)
            exit = .discarded
        }
    }

    func redraft(from cv: CVTemplate) async {
        await attempt {
            try await client.discardDraft(draft.id)
            _ = try await client.createDraft(job.id, cv)
            exit = .redrafted
        }
    }

    func finishAtDesk(note: String) async {
        await attempt {
            guard let type = try await client.taskTypes().first(where: { $0.systemKey == "finish" }) else {
                return fail("There's no Finish task type.")
            }
            try await client.createTask(type.id, job.id, note)
            exit = .atDesk
        }
    }

    private func fetchPDF() async {
        guard let source = try? await client.draftPDF(draft.id) else { return }
        let named = source.deletingLastPathComponent().appending(path: pdfName)
        try? FileManager.default.removeItem(at: named)
        pdfURL = (try? FileManager.default.copyItem(at: source, to: named)) == nil ? source : named
    }

    private func attempt(_ work: () async throws -> Void) async {
        do {
            try await work()
            isOffline = false
        } catch is CancellationError {
            return
        } catch {
            noteOffline(error)
            fail(error.localizedDescription)
        }
    }

    private func noteOffline(_ error: any Error) {
        if let error = error as? URLError, [.notConnectedToInternet, .networkConnectionLost, .timedOut].contains(error.code) {
            isOffline = true
        }
    }

    private func fail(_ message: String) {
        failureMessage = message
        hasFailure = true
    }
}
