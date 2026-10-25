import Foundation

enum APIError: LocalizedError, Equatable {
    case unauthorized
    case status(Int)

    var errorDescription: String? {
        switch self {
        case .unauthorized: "You're signed out."
        case .status(let code): "The server returned \(code)."
        }
    }
}
