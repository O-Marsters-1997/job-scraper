import Foundation

enum DueFocus: Hashable {
    case chase(jobID: String)
    case apply(jobID: String)

    static let dueTabURL = URL(string: "fasttrack://due")!

    init?(url: URL) {
        guard url.scheme == "fasttrack", url.pathComponents.count == 2 else { return nil }
        let id = url.lastPathComponent
        switch url.host() {
        case "due": self = .chase(jobID: id)
        case "apply": self = .apply(jobID: id)
        default: return nil
        }
    }

    var url: URL {
        var components = URLComponents()
        components.scheme = "fasttrack"
        switch self {
        case .chase(let id):
            components.host = "due"
            components.path = "/" + id
        case .apply(let id):
            components.host = "apply"
            components.path = "/" + id
        }
        return components.url!
    }
}
