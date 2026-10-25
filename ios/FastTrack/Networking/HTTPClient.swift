import Foundation

struct HTTPClient {
    let baseURL: URL
    var session = URLSession.shared

    func get<T: Decodable>(_ path: String, query: [URLQueryItem] = []) async throws -> T {
        try await send(URLRequest(url: baseURL.appending(path: path).appending(queryItems: query)))
    }

    func post<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        var request = URLRequest(url: baseURL.appending(path: path))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        return try await send(request)
    }

    private func send<T: Decodable>(_ request: URLRequest) async throws -> T {
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch let error as URLError where error.code == .cancelled {
            throw CancellationError()
        }
        switch (response as? HTTPURLResponse)?.statusCode ?? 0 {
        case 200..<300: return try JSONDecoder().decode(T.self, from: data)
        case 401: throw APIError.unauthorized
        case let code: throw APIError.status(code)
        }
    }
}
