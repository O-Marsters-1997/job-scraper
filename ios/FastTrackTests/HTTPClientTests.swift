import Foundation
import Testing
@testable import FastTrack

struct HTTPClientTests {
    private final class Stub: URLProtocol {
        override class func canInit(with request: URLRequest) -> Bool { true }
        override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

        override func startLoading() {
            let url = request.url!
            let status = Int(url.pathComponents[1])!
            let response = HTTPURLResponse(url: url, statusCode: status, httpVersion: nil, headerFields: nil)!
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: Data(#"{"error":"only a ready draft can be edited"}"#.utf8))
            client?.urlProtocolDidFinishLoading(self)
        }

        override func stopLoading() {}
    }

    @Test func conflictShowsTheServerMessage() async {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [Stub.self]
        let http = HTTPClient(baseURL: URL(string: "https://example.test/409")!, session: URLSession(configuration: configuration))

        await #expect(throws: APIError.unprocessable("only a ready draft can be edited")) {
            try await http.put("tailoring/drafts/d/slots", body: ["a": "b"])
        }
    }
}
