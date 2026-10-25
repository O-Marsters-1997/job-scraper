import SwiftUI

struct LoadStateOverlay<Value>: View {
    let state: LoadState<Value>
    let failureTitle: String

    var body: some View {
        switch state {
        case .idle, .loading:
            ProgressView()
        case .failed(let error):
            ContentUnavailableView(failureTitle, systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
        case .loaded:
            EmptyView()
        }
    }
}
