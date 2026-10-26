import Foundation

struct HTTPClient {
    let baseURL: URL
    var session = URLSession.shared

    func get<T: Decodable>(_ path: String, query: [URLQueryItem] = []) async throws -> T {
        try JSONDecoder.api.decode(T.self, from: await send(URLRequest(url: baseURL.appending(path: path).appending(queryItems: query))))
    }

    func post<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        try JSONDecoder.api.decode(T.self, from: await send(try request("POST", path, body: body)))
    }

    func post(_ path: String, body: some Encodable) async throws {
        _ = try await send(try request("POST", path, body: body))
    }

    func put<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        try JSONDecoder.api.decode(T.self, from: await send(try request("PUT", path, body: body)))
    }

    func put(_ path: String, body: some Encodable) async throws {
        _ = try await send(try request("PUT", path, body: body))
    }

    func delete(_ path: String) async throws {
        var request = URLRequest(url: baseURL.appending(path: path))
        request.httpMethod = "DELETE"
        _ = try await send(request)
    }

    private func request(_ method: String, _ path: String, body: some Encodable) throws -> URLRequest {
        var request = URLRequest(url: baseURL.appending(path: path))
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        return request
    }

    private func send(_ request: URLRequest) async throws -> Data {
        var request = request
        request.setValue("ios", forHTTPHeaderField: "X-Client")
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch let error as URLError where error.code == .cancelled {
            throw CancellationError()
        }
        switch (response as? HTTPURLResponse)?.statusCode ?? 0 {
        case 200..<300: return data
        case 401: throw APIError.unauthorized
        case let code: throw APIError.status(code)
        }
    }
}

extension JSONDecoder {
    static var api: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let text = try decoder.singleValueContainer().decode(String.self)
            if let date = try? Date.ISO8601FormatStyle(includingFractionalSeconds: true).parse(text) { return date }
            if let date = try? Date.ISO8601FormatStyle().parse(text) { return date }
            if let date = try? Date.dayFormat.parse(text) { return date }
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Bad date \(text)"))
        }
        return decoder
    }
}
