import Observation

@Observable
final class Session {
    enum State: Equatable {
        case loading
        case signedOut
        case signedIn
        case unreachable(String)
    }

    private(set) var state = State.loading
    private let client: APIClient

    init(client: APIClient) {
        self.client = client
    }

    func restore() async {
        state = .loading
        do {
            try await client.restoreSession()
            state = .signedIn
        } catch APIError.unauthorized {
            state = .signedOut
        } catch {
            state = .unreachable(error.localizedDescription)
        }
    }

    func expire() {
        state = .signedOut
    }

    func logIn(username: String, password: String) async throws {
        try await client.logIn(username, password)
        state = .signedIn
    }
}
