import SwiftUI

struct ApplyOnListingSheet: View {
    let companyName: String
    let listing: Job?
    let pdfURL: URL?
    let pdfName: String
    let markApplied: () async -> Bool
    @Environment(\.openURL) private var openURL
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.dismiss) private var dismiss
    @State private var openedListing = false
    @State private var askApplied = false

    private var listingURL: URL? { listing.flatMap { URL(string: $0.url) } }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    if let pdfURL {
                        ShareLink(item: pdfURL) { step(1, "Send your CV", detail: pdfName, systemImage: "square.and.arrow.up") }
                    }
                    Button {
                        guard let listingURL else { return }
                        openedListing = true
                        openURL(listingURL)
                    } label: {
                        step(2, "Open the listing", detail: listingURL?.host() ?? "Listing unavailable", systemImage: "safari")
                    }
                    .disabled(listingURL == nil)
                } footer: {
                    Text("Save the PDF to Files or hand it to the form, then apply on the listing. We'll ask when you're back.")
                }
            }
            .navigationTitle("Apply to \(companyName)")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Close") { dismiss() } }
            }
            .onChange(of: scenePhase) { _, phase in
                if phase == .active, openedListing { askApplied = true }
            }
            .alert("Did you apply to \(companyName)?", isPresented: $askApplied) {
                Button("Yes, applied") {
                    Task {
                        if await markApplied() { dismiss() }
                    }
                }
                Button("Not yet", role: .cancel) { openedListing = false }
            } message: {
                Text("Marks the application Applied.")
            }
        }
        .presentationDetents([.medium])
    }

    private func step(_ number: Int, _ title: String, detail: String, systemImage: String) -> some View {
        HStack(spacing: 12) {
            Text(number, format: .number)
                .font(.caption.weight(.semibold).monospacedDigit())
                .frame(width: 22, height: 22)
                .background(Color(.tertiarySystemFill), in: .circle)
                .foregroundStyle(.secondary)
            VStack(alignment: .leading, spacing: 2) {
                Text(title).foregroundStyle(.primary)
                Text(detail).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            Image(systemName: systemImage).foregroundStyle(.secondary)
        }
        .frame(minHeight: 44)
    }
}
