import Foundation
import Testing
@testable import FastTrack

struct HTTPClientTests {
    private final class Stub: URLProtocol, @unchecked Sendable {
        nonisolated(unsafe) static var status = 200
        nonisolated(unsafe) static var body = ""

        override class func canInit(with request: URLRequest) -> Bool { true }
        override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

        override func startLoading() {
            let url = request.url!
            let response = HTTPURLResponse(url: url, statusCode: Self.status, httpVersion: nil, headerFields: nil)!
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: Data(Self.body.utf8))
            client?.urlProtocolDidFinishLoading(self)
        }

        override func stopLoading() {}
    }

    @Test func conflictShowsTheServerMessage() async {
        Stub.status = 409
        Stub.body = #"{"error":"only a ready draft can be edited"}"#
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [Stub.self]
        let http = HTTPClient(baseURL: URL(string: "https://example.test")!, session: URLSession(configuration: configuration))

        await #expect(throws: APIError.unprocessable("only a ready draft can be edited")) {
            try await http.put("tailoring/drafts/d/slots", body: ["a": "b"])
        }
    }
}
