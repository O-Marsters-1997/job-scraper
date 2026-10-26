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
    var recruiterContacts: (_ jobID: String) async throws -> [RecruiterContact]
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
                let page: JobPage = try await http.get("jobs", query: [
                    URLQueryItem(name: "seen", value: "unseen"),
                    URLQueryItem(name: "scored", value: "1"),
                    URLQueryItem(name: "sort", value: "relevant"),
                ])
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
            recruiterContacts: { jobID in
                try await http.get("jobs/\(jobID)/recruiter-contacts")
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
            recruiterContacts: { _ in try respond(contacts) }
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
