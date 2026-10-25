import Foundation
import SwiftUI

struct APIClient {
    var restoreSession: () async throws -> Void
    var logIn: (_ username: String, _ password: String) async throws -> Void
    var jobs: () async throws -> [Job]
    var job: (_ id: String) async throws -> Job
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
            jobs: {
                let page: JobPage = try await http.get("jobs", query: [
                    URLQueryItem(name: "sort", value: "best"),
                    URLQueryItem(name: "scored", value: "1"),
                ])
                return page.items
            },
            job: { id in
                try await http.get("jobs/\(id)")
            }
        )
    }
}

extension APIClient {
    static func mock(jobs: [Job] = [.sample], failure: APIError? = nil) -> APIClient {
        func respond<T>(_ value: T) throws -> T {
            if let failure { throw failure }
            return value
        }
        return APIClient(
            restoreSession: { try respond(()) },
            logIn: { _, _ in try respond(()) },
            jobs: { try respond(jobs) },
            job: { id in try respond(jobs.first { $0.id == id } ?? .sample) }
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

private struct JobPage: Decodable {
    let items: [Job]
}
