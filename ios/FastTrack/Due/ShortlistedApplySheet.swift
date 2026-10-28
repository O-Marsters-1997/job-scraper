import SwiftUI

struct ShortlistedApplySheet: View {
    let job: DueJob
    let markApplied: () async -> Bool
    @Environment(\.apiClient) private var client
    @State private var listing: Job?

    var body: some View {
        ApplyOnListingSheet(companyName: job.companyName, listing: listing, pdfURL: nil, pdfName: "", markApplied: markApplied)
            .task { listing = try? await client.job(job.id) }
    }
}
