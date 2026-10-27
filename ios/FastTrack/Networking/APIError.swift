import Foundation

enum APIError: LocalizedError, Equatable {
    case unauthorized
    case status(Int)
    case unprocessable(String)

    var errorDescription: String? {
        switch self {
        case .unprocessable(let message): message
        case .unauthorized: "You're signed out."
        case .status(let code): "The server returned \(code)."
        }
    }
}
