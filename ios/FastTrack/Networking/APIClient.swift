import Foundation
import SwiftUI

struct APIClient {
    var restoreSession: () async throws -> Void
    var logIn: (_ username: String, _ password: String) async throws -> Void
    var triageQueue: () async throws -> [Job]
    var job: (_ id: String) async throws -> Job
    var shortlist: (_ jobID: String) async throws -> Application
    var deleteApplication: (_ id: String) async throws -> Void
    var dismiss: (_ jobID: String, _ reasons: [GradeReason]) async throws -> Void
    var clearGrade: (_ jobID: String) async throws -> Void
    var setSeen: (_ jobIDs: [String], _ seen: Bool) async throws -> Void
    var recordEvent: (_ event: EventInput) async throws -> Void
    var excludeCompany: (_ companyID: String) async throws -> CompanyExclusion
    var unexcludeCompany: (_ companyID: String, _ removeName: Bool) async throws -> Void
    var note: (_ jobID: String) async throws -> JobNote
    var putNote: (_ jobID: String, _ body: String) async throws -> Void
    var due: () async throws -> [DueJob]
    var setChase: (_ applicationID: String, _ chaseBy: Date) async throws -> Void
    var clearChase: (_ applicationID: String) async throws -> Void
    var logContact: (_ jobID: String, _ channel: ContactChannel, _ outcome: ContactOutcome) async throws -> RecruiterContact
    var setTaskDone: (_ taskID: String, _ done: Bool) async throws -> Void
    var taskTypes: () async throws -> [TaskType]
    var createTask: (_ typeID: String, _ jobID: String, _ notes: String) async throws -> Void
    var recruiterContacts: (_ jobID: String) async throws -> [RecruiterContact]
    var cvTemplates: () async throws -> [CVTemplate]
    var createDraft: (_ jobID: String, _ cv: CVTemplate) async throws -> DraftRef
    var draftPDF: (_ draftID: String) async throws -> URL
    var draftRoles: (_ draftID: String) async throws -> DraftRoles
    var keepDraft: (_ draftID: String) async throws -> Void
    var discardDraft: (_ draftID: String) async throws -> Void
}

extension APIClient {
    static let live = live(baseURL: URL(string: Bundle.main.object(forInfoDictionaryKey: "APIBaseURL") as! String)!)

    static func live(baseURL: URL) -> APIClient {
        let http = HTTPClient(baseURL: baseURL)
        return APIClient(
            restoreSession: {
                let _: Me = try await http.get("auth/me")
            },
            logIn: { username, password in
                let _: Me = try await http.post("auth/login", body: Credentials(username: username, password: password))
            },
            triageQueue: {
                let page: JobPage = try await http.get("jobs/triage")
                return page.items
            },
            job: { id in
                try await http.get("jobs/\(id)")
            },
            shortlist: { jobID in
                try await http.post("applications", body: CreateApplicationInput(jobID: jobID))
            },
            deleteApplication: { id in
                try await http.delete("applications/\(id)")
            },
            dismiss: { jobID, reasons in
                try await http.put("jobs/\(jobID)/grade", body: GradeInput(grade: "no", reasons: reasons))
            },
            clearGrade: { jobID in
                try await http.delete("jobs/\(jobID)/grade")
            },
            setSeen: { jobIDs, seen in
                try await http.post("jobs/seen", body: SeenInput(jobIDs: jobIDs, seen: seen))
            },
            recordEvent: { event in
                try await http.post("events", body: event)
            },
            excludeCompany: { companyID in
                try await http.put("companies/\(companyID)/exclusion", body: EmptyBody())
            },
            unexcludeCompany: { companyID, removeName in
                try await http.post("companies/\(companyID)/exclusion/undo", body: UnexcludeInput(removeName: removeName))
            },
            note: { jobID in
                try await http.get("jobs/\(jobID)/note")
            },
            putNote: { jobID, body in
                try await http.put("jobs/\(jobID)/note", body: NoteInput(body: body))
            },
            due: {
                let list: DueList = try await http.get("due")
                return list.jobs
            },
            setChase: { applicationID, chaseBy in
                try await http.put("applications/\(applicationID)/chase", body: ChaseInput(chaseBy: chaseBy.dayString))
            },
            clearChase: { applicationID in
                try await http.delete("applications/\(applicationID)/chase")
            },
            logContact: { jobID, channel, outcome in
                try await http.post(
                    "jobs/\(jobID)/recruiter-contacts",
                    body: ContactInput(contactedOn: Date.now.dayString, channel: channel, outcome: outcome))
            },
            setTaskDone: { taskID, done in
                try await http.put("tasks/\(taskID)/done", body: TaskDoneInput(done: done))
            },
            taskTypes: {
                try await http.get("task-types")
            },
            createTask: { typeID, jobID, notes in
                try await http.post("tasks", body: CreateTaskInput(typeID: typeID, jobID: jobID, notes: notes))
            },
            recruiterContacts: { jobID in
                try await http.get("jobs/\(jobID)/recruiter-contacts")
            },
            cvTemplates: {
                try await http.get("cv-templates", query: [URLQueryItem(name: "kind", value: "cv")])
            },
            createDraft: { jobID, cv in
                try await http.post("tailoring/drafts", body: DraftInput(jobID: jobID, docID: cv.docID, tabID: cv.tabID))
            },
            draftPDF: { draftID in
                let url = FileManager.default.temporaryDirectory.appending(path: "cv-\(draftID).pdf")
                try await http.data("tailoring/drafts/\(draftID)/pdf").write(to: url)
                return url
            },
            draftRoles: { draftID in
                try await http.get("tailoring/drafts/\(draftID)/roles")
            },
            keepDraft: { draftID in
                try await http.post("tailoring/drafts/\(draftID)/keep", body: EmptyBody())
            },
            discardDraft: { draftID in
                try await http.post("tailoring/drafts/\(draftID)/discard", body: EmptyBody())
            }
        )
    }
}

extension APIClient {
    static func mock(jobs: [Job] = [.sample], due: [DueJob] = [], contacts: [RecruiterContact] = [], failure: APIError? = nil) -> APIClient {
        func respond<T>(_ value: T) throws -> T {
            if let failure { throw failure }
            return value
        }
        return APIClient(
            restoreSession: { try respond(()) },
            logIn: { _, _ in try respond(()) },
            triageQueue: { try respond(jobs) },
            job: { id in try respond(jobs.first { $0.id == id } ?? .sample) },
            shortlist: { jobID in try respond(Application(id: "application-\(jobID)", jobID: jobID)) },
            deleteApplication: { _ in try respond(()) },
            dismiss: { _, _ in try respond(()) },
            clearGrade: { _ in try respond(()) },
            setSeen: { _, _ in try respond(()) },
            recordEvent: { _ in try respond(()) },
            excludeCompany: { _ in try respond(CompanyExclusion(name: "Monzo", added: true)) },
            unexcludeCompany: { _, _ in try respond(()) },
            note: { jobID in try respond(JobNote(jobID: jobID, body: "")) },
            putNote: { _, _ in try respond(()) },
            due: { try respond(due) },
            setChase: { _, _ in try respond(()) },
            clearChase: { _ in try respond(()) },
            logContact: { jobID, channel, outcome in
                try respond(RecruiterContact(id: "contact-\(jobID)", contactedOn: .now, channel: channel, outcome: outcome))
            },
            setTaskDone: { _, _ in try respond(()) },
            taskTypes: { try respond([TaskType(id: "type-finish", name: "Finish application", systemKey: "finish")]) },
            createTask: { _, _, _ in try respond(()) },
            recruiterContacts: { _ in try respond(contacts) },
            cvTemplates: { try respond([CVTemplate(docID: "doc", tabID: "tab", title: "Backend", isDefault: true)]) },
            createDraft: { _, _ in try respond(DraftRef(id: "draft-new")) },
            draftPDF: { _ in try respond(FileManager.default.temporaryDirectory.appending(path: "cv-draft.pdf")) },
            draftRoles: { _ in try respond(.sample) },
            keepDraft: { _ in try respond(()) },
            discardDraft: { _ in try respond(()) }
        )
    }
}

extension EnvironmentValues {
    @Entry var apiClient: APIClient = .live
}

private struct Me: Decodable {
    let username: String
}

private struct Credentials: Encodable {
    let username: String
    let password: String
}
