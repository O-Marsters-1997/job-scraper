#if DEBUG
import QuickLook
import SwiftUI

struct DueListView: View {
    @Environment(TriagePrototypeStore.self) private var store
    @State private var pdfURL: URL?
    @State private var contactFor: PrototypeJob?
    @State private var struck: Set<PrototypeTask.ID> = []

    private var dueChases: [PrototypeChase] { store.chases.filter { $0.chaseBy <= .now }.sorted { $0.chaseBy < $1.chaseBy } }

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
            } else if dueChases.isEmpty && store.drafts.isEmpty && store.tasks.isEmpty {
                ContentUnavailableView("Nothing due", systemImage: "checkmark.circle",
                                       description: Text("Chases, open Tasks and ready CV drafts show up here."))
            } else {
                list
            }
        }
        .navigationTitle("Due")
        .navigationDestination(for: PrototypeJob.self) { PrototypeJobPage(job: $0) }
        .quickLookPreview($pdfURL)
        .sheet(item: $contactFor) { job in
            ContactSheet { store.logContact($0, for: job) }
        }
    }

    private var list: some View {
        List {
            if !dueChases.isEmpty {
                Section("Chases due") {
                    ForEach(dueChases) { chase in
                        NavigationLink(value: chase.job) { chaseRow(chase) }
                            .swipeActions(edge: .leading) {
                                Button("Done", systemImage: "checkmark") { withAnimation { store.completeChase(chase) } }
                                    .tint(Color.brandTint)
                            }
                            .swipeActions(edge: .trailing) {
                                Button("Log contact", systemImage: "person.badge.plus") { contactFor = chase.job }
                                    .tint(.blue)
                                Menu {
                                    ForEach([(2, "2 days"), (7, "1 week"), (14, "2 weeks")], id: \.0) { days, label in
                                        Button(label) { withAnimation { store.snooze(chase, days: days) } }
                                    }
                                } label: {
                                    Label("Snooze", systemImage: "moon.zzz")
                                }
                                .tint(.orange)
                            }
                    }
                }
            }
            if !store.drafts.isEmpty {
                Section("CV drafts") {
                    ForEach(store.drafts) { draft in draftRow(draft) }
                }
            }
            if !store.tasks.isEmpty {
                Section("Open tasks") {
                    ForEach(store.tasks) { task in taskRow(task) }
                }
            }
        }
        .listStyle(.insetGrouped)
        .refreshable { try? await Task.sleep(for: .milliseconds(600)) }
    }

    private func chaseRow(_ chase: PrototypeChase) -> some View {
        HStack(spacing: 12) {
            Monogram(job: chase.job, size: 36)
            VStack(alignment: .leading, spacing: 2) {
                Text(chase.job.title).font(.headline).lineLimit(2)
                Text(chase.job.company).font(.subheadline).foregroundStyle(.secondary)
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 2) {
                Text(chase.chaseBy.shortGB).font(.subheadline.monospacedDigit())
                if chase.overdueDays > 0 {
                    Text("overdue by \(chase.overdueDays) \(chase.overdueDays == 1 ? "day" : "days")")
                        .font(.footnote)
                        .foregroundStyle(.red)
                }
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func draftRow(_ draft: PrototypeDraft) -> some View {
        HStack(spacing: 12) {
            Button {
                if draft.state == .ready { pdfURL = DraftPDF.render(for: draft) }
            } label: {
                HStack(spacing: 12) {
                    Image(systemName: "doc.richtext")
                        .font(.title3)
                        .foregroundStyle(draft.state == .failed ? Color.red : Color.brandTint)
                        .frame(width: 36, height: 36)
                        .background((draft.state == .failed ? Color.red : Color.brandTint).opacity(0.12), in: .rect(cornerRadius: 9, style: .continuous))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(draft.job.title).font(.headline).lineLimit(1)
                        switch draft.state {
                        case .pending: Text("Drafting…").font(.subheadline).foregroundStyle(.secondary)
                        case .ready: Text(draft.template).font(.subheadline).foregroundStyle(.secondary)
                        case .failed: Text("Couldn't draft").font(.subheadline).foregroundStyle(.red)
                        }
                    }
                    Spacer(minLength: 0)
                }
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .disabled(draft.state == .pending)
            switch draft.state {
            case .pending:
                ProgressView()
            case .ready:
                Button("Mark applied") { withAnimation { store.markApplied(draft) } }
                    .buttonStyle(.bordered)
                    .tint(Color.brandTint)
                    .font(.subheadline.weight(.semibold))
            case .failed:
                Button("Try again") { withAnimation { store.retry(draft) } }
                    .buttonStyle(.bordered)
                    .font(.subheadline.weight(.semibold))
            }
        }
    }

    private func taskRow(_ task: PrototypeTask) -> some View {
        let isStruck = struck.contains(task.id)
        return HStack(spacing: 12) {
            Button {
                struck.insert(task.id)
                Task {
                    try? await Task.sleep(for: .seconds(1))
                    withAnimation { store.completeTask(task) }
                    struck.remove(task.id)
                }
            } label: {
                Image(systemName: isStruck ? "checkmark.circle.fill" : "circle")
                    .font(.title2)
                    .foregroundStyle(isStruck ? Color.brandTint : Color.secondary)
                    .contentTransition(.symbolEffect(.replace))
                    .frame(width: 44, height: 44)
            }
            .buttonStyle(.plain)
            .sensoryFeedback(.success, trigger: isStruck) { _, new in new }
            .accessibilityLabel("Mark \(task.type) done")
            NavigationLink(value: task.job) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(task.type).font(.headline).strikethrough(isStruck)
                    Text("\(task.job.title) · \(task.job.company)")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                .opacity(isStruck ? 0.5 : 1)
            }
        }
    }
}

struct Monogram: View {
    let job: PrototypeJob
    var size: CGFloat = 48

    var body: some View {
        Text(String(job.company.prefix(1)))
            .font(size > 40 ? .title2.weight(.bold) : .headline.weight(.bold))
            .fontDesign(.rounded)
            .foregroundStyle(job.band.colour)
            .frame(width: size, height: size)
            .background(job.band.colour.opacity(0.14), in: .rect(cornerRadius: size / 4, style: .continuous))
            .accessibilityHidden(true)
    }
}

struct PrototypeJobPage: View {
    let job: PrototypeJob

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                HStack(spacing: 14) {
                    Monogram(job: job)
                    VStack(alignment: .leading, spacing: 2) {
                        HStack(alignment: .firstTextBaseline, spacing: 6) {
                            Text(job.band.rawValue).font(.headline).foregroundStyle(job.band.colour)
                            Text(job.score, format: .number).font(.headline.monospacedDigit()).foregroundStyle(.secondary)
                        }
                        Text("\(job.source.capitalized) · \(job.age) ago").font(.footnote).foregroundStyle(.secondary)
                    }
                }
                MetaLines(job: job)
                JobDetailBody(job: job)
            }
            .padding(22)
        }
        .background(Color(.systemGroupedBackground))
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
