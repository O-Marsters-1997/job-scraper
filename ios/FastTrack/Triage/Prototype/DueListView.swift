#if DEBUG
import QuickLook
import SwiftUI

struct DueListView: View {
    @Environment(TriagePrototypeStore.self) private var store
    @State private var pdfURL: URL?
    @State private var contactFor: PrototypeJob?
    @State private var openJob: PrototypeJob?

    var body: some View {
        Group {
            if store.offline {
                ContentUnavailableView {
                    Label("Due list needs a connection", systemImage: "wifi.slash")
                } description: {
                    Text("Triage keeps working offline.")
                } actions: {
                    Button("Try again") { store.offline = false }
                }
            } else if store.dueItems.isEmpty {
                ContentUnavailableView("Nothing due", systemImage: "checkmark.circle",
                                       description: Text("Chases, open Tasks and ready CV drafts show up here."))
            } else {
                DueNotebook(actions: actions)
            }
        }
        .navigationTitle("Due")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $openJob) { PrototypeJobPage(job: $0) }
        .quickLookPreview($pdfURL)
        .sheet(item: $contactFor) { job in
            ContactSheet { store.logContact($0, for: job) }
        }
    }

    private var actions: DueActions {
        DueActions(openPDF: { pdfURL = DraftPDF.render(for: $0) }, logContact: { contactFor = $0 }, openJob: { openJob = $0 })
    }
}

struct PrototypeJobPage: View {
    let job: PrototypeJob

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                VStack(alignment: .leading, spacing: 12) {
                    BandRow(job: job)
                    MetaLines(job: job)
                }
                JobDetailBody(job: job)
            }
            .padding(16)
        }
        .background(Color(.systemBackground))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ShareLink(item: URL(string: "https://example.com/jobs/\(job.id)")!)
        }
    }
}

enum DraftPDF {
    static func render(for draft: PrototypeDraft) -> URL {
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("CV – \(draft.job.company).pdf")
        let page = CGRect(x: 0, y: 0, width: 595, height: 842)
        let data = UIGraphicsPDFRenderer(bounds: page).pdfData { context in
            context.beginPage()
            let heading: [NSAttributedString.Key: Any] = [.font: UIFont.systemFont(ofSize: 24, weight: .bold)]
            let body: [NSAttributedString.Key: Any] = [.font: UIFont.systemFont(ofSize: 11), .foregroundColor: UIColor.darkGray]
            "Olly Marsters".draw(at: CGPoint(x: 56, y: 56), withAttributes: heading)
            "Tailored for \(draft.job.title), \(draft.job.company)".draw(at: CGPoint(x: 56, y: 92), withAttributes: body)
            for i in 0..<18 {
                "Prototype CV line \(i + 1). Real drafts come from GET /tailoring/drafts/{id}/pdf."
                    .draw(at: CGPoint(x: 56, y: 140 + i * 22), withAttributes: body)
            }
        }
        try? data.write(to: url)
        return url
    }
}
#endif
