import SwiftUI

struct ApplicationScreen: View {
    let job: DueJob
    let draft: DueJob.Draft
    var onFinished: () async -> Void = {}
    @Environment(\.apiClient) private var client
    @State private var model: ApplicationModel?

    var body: some View {
        Group {
            if let model {
                ApplicationView(model: model, onFinished: onFinished)
            }
        }
        .onAppear {
            if model == nil { model = ApplicationModel(client: client, job: job, draft: draft) }
        }
    }
}
