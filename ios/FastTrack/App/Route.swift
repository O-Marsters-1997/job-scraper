import Foundation

enum Route: Hashable {
    case job(id: String)

    init?(url: URL) {
        guard url.scheme == "fasttrack", url.host() == "jobs", url.pathComponents.count == 2 else { return nil }
        self = .job(id: url.lastPathComponent)
    }
}
