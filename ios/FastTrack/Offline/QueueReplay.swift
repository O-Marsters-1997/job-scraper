import Foundation

extension APIClient {
    func perform(_ kind: QueuedAction.Kind, jobID: String) async throws -> UndoReceipt? {
        switch kind {
        case .shortlist:
            return .application(id: try await shortlist(jobID).id)
        case .dismiss(let reasons):
            try await dismiss(jobID, reasons)
            try? await recordEvent(EventInput(type: .jobDismissed, subjectID: jobID, reason: "", undone: false))
            return .grade
        case .skip:
            try await setSeen([jobID], true)
            return .seen
        case .excludeCompany(let companyID):
            let exclusion = try await excludeCompany(companyID)
            return .exclusion(companyID: companyID, removeName: exclusion.added)
        case .note(let body):
            let previous = try await note(jobID).body
            try await putNote(jobID, body)
            return .note(previous: previous)
        case .undo:
            return nil
        }
    }

    func revert(_ receipt: UndoReceipt?, jobID: String) async throws {
        switch receipt {
        case .application(let id):
            try await deleteApplication(id)
        case .grade:
            try await clearGrade(jobID)
            try? await recordEvent(EventInput(type: .jobDismissed, subjectID: jobID, reason: "", undone: true))
        case .seen:
            try await setSeen([jobID], false)
        case .exclusion(let companyID, let removeName):
            try await unexcludeCompany(companyID, removeName)
        case .note(let previous):
            try await putNote(jobID, previous)
        case nil:
            break
        }
    }
}

func replay(_ queue: [QueuedAction], client: APIClient, receipts: inout [UUID: UndoReceipt]) async -> ReplayOutcome {
    let ordered = queue.enumerated().sorted { ($0.element.queuedAt, $0.offset) < ($1.element.queuedAt, $1.offset) }.map(\.element)
    var synced = 0
    var skipped = 0
    for (index, action) in ordered.enumerated() {
        do {
            if case .undo(let original) = action.kind {
                guard let receipt = receipts[original] else {
                    skipped += 1
                    continue
                }
                try await client.revert(receipt, jobID: action.jobID)
                receipts[original] = nil
            } else {
                receipts[action.id] = try await client.perform(action.kind, jobID: action.jobID)
                if case .shortlist(draftCV: true) = action.kind,
                   let cv = try? await client.cvTemplates().first(where: \.isDefault) {
                    _ = try? await client.createDraft(action.jobID, cv)
                }
            }
            synced += 1
        } catch APIError.status(409) {
            synced += 1
        } catch APIError.status(404) {
            skipped += 1
        } catch let error where error is URLError || error as? APIError == .unauthorized || error.isServerFailure {
            let result = ReplayResult(synced: synced, skipped: skipped)
            return ReplayOutcome(result: result, remaining: Array(ordered[index...]), stoppedBy: error)
        } catch {
            skipped += 1
        }
    }
    return ReplayOutcome(result: ReplayResult(synced: synced, skipped: skipped), remaining: [], stoppedBy: nil)
}

private extension Error {
    var isServerFailure: Bool {
        if case .status(let code)? = self as? APIError { code >= 500 } else { false }
    }
}
