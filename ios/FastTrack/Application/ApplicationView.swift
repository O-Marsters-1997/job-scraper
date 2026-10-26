import QuickLook
import SwiftUI

struct ApplicationView: View {
    private enum Card: Hashable { case cv, letter, apply }

    let model: ApplicationModel
    var onFinished: () async -> Void = {}
    @Environment(\.dismiss) private var dismiss
    @State private var open: Set<Card> = [.cv]
    @State private var applying = false
    @State private var toDesk = false
    @State private var showingJob = false
    @State private var confirmingDiscard = false
    @State private var previewURL: URL?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                masthead
                if model.isOffline {
                    Label("Needs a connection. Triage still works offline.", systemImage: "wifi.slash")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                        .padding(10)
                        .frame(maxWidth: .infinity)
                        .background(Color(.secondarySystemBackground), in: .rect(cornerRadius: 12, style: .continuous))
                }
                if model.roles.value == nil {
                    loading
                } else {
                    VStack(spacing: 10) {
                        cvCard
                        letterCard
                        applyCard
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 8)
            .padding(.bottom, 24)
            .animation(.snappy(duration: 0.22), value: open)
        }
        .background(Color(.systemBackground))
        .disabled(model.isOffline)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("Job description", systemImage: "doc.text") { showingJob = true }
            }
            ToolbarItem(placement: .primaryAction) { moreMenu }
        }
        .onChange(of: model.kept) { _, kept in
            guard kept else { return }
            open.remove(.cv)
            open.insert(.apply)
        }
        .onChange(of: model.exit) { _, exit in
            guard exit != nil else { return }
            Task {
                await onFinished()
                dismiss()
            }
        }
        .sensoryFeedback(.success, trigger: model.kept) { _, now in now }
        .quickLookPreview($previewURL)
        .sheet(isPresented: $showingJob) { JobDescriptionSheet(job: model.listing) }
        .sheet(isPresented: $applying) { ApplyOnListingSheet(model: model) }
        .sheet(isPresented: $toDesk) { FinishAtDeskSheet(model: model) }
        .confirmationDialog("Discard this draft?", isPresented: $confirmingDiscard, titleVisibility: .visible) {
            Button("Discard draft", role: .destructive) { Task { await model.discard() } }
        }
        .alert("Couldn't do that", isPresented: Bindable(model).hasFailure) {
            Button("OK") {}
        } message: {
            Text(model.failureMessage)
        }
        .task { await model.load() }
    }

    @ViewBuilder
    private var loading: some View {
        if case .failed(let error) = model.roles {
            ContentUnavailableView(
                "Couldn't load the draft", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
        } else {
            ProgressView().frame(maxWidth: .infinity, minHeight: 120)
        }
    }

    private var masthead: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(model.job.companyName).font(.system(.title, design: .serif)).accessibilityAddTraits(.isHeader)
            Text(model.job.title).font(.headline)
            HStack(spacing: 8) {
                if let band = model.job.band { DotLabel(text: band.title, hue: band.colour) }
                if let score = model.job.score {
                    Text(score, format: .number).font(.footnote.weight(.medium).monospacedDigit())
                }
            }
            .padding(.top, 4)
        }
    }

    private var moreMenu: some View {
        Menu("More", systemImage: "ellipsis") {
            if let pdf = model.pdfURL {
                Button("Preview PDF", systemImage: "eye") { previewURL = pdf }
                ShareLink(item: pdf) { Label("Download PDF", systemImage: "arrow.down.doc") }
            }
            if !model.templates.isEmpty {
                Menu("Redraft from…", systemImage: "arrow.triangle.2.circlepath") {
                    ForEach(model.templates, id: \.docID) { template in
                        Button(template.title) { Task { await model.redraft(from: template) } }
                    }
                }
            }
            Divider()
            Button("Finish at desk…", systemImage: "desktopcomputer") { toDesk = true }
            Button("Discard draft", systemImage: "trash", role: .destructive) { confirmingDiscard = true }
        }
        .disabled(model.isOffline)
    }

    private var cvStatus: (text: String, hue: Color)? {
        guard let roles = model.roles.value else { return nil }
        if roles.linesOver > 0 { return ("\(roles.linesOver) over", .red) }
        if roles.spills { return ("Page 2", .red) }
        return roles.kept ? ("Kept", .green) : nil
    }

    private var cvCard: some View {
        ApplicationNotebookCard(
            title: "CV", summary: model.draft.cvTitle, done: model.kept, status: cvStatus,
            isOpen: open.contains(.cv), toggle: { toggle(.cv) }
        ) {
            VStack(alignment: .leading, spacing: 0) {
                ForEach(model.roles.value?.roles ?? []) { ApplicationRoleGroup(role: $0, model: model) }
                if !model.kept { keepFooter }
            }
        }
    }

    private var keepFooter: some View {
        VStack(spacing: 8) {
            Button {
                Task { await model.keep() }
            } label: {
                Text("Keep CV").frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .buttonBorderShape(.roundedRectangle(radius: 12))
            .controlSize(.large)
            .disabled(!model.canKeep)
            if let roles = model.roles.value {
                if roles.linesOver > 0 {
                    Text("Remove ^[\(roles.linesOver) line](inflect: true) to fit.").font(.footnote).foregroundStyle(.secondary)
                } else if roles.spills {
                    Text("Runs onto page 2: remove a line").font(.footnote).foregroundStyle(.secondary)
                }
            }
        }
        .padding(14)
    }

    private var letterCard: some View {
        ApplicationNotebookCard(
            title: "Cover letter", summary: "Write it at the desk", done: false, status: nil,
            isOpen: open.contains(.letter), toggle: { toggle(.letter) }
        ) {
            Text("Cover letters are drafted and reviewed at the desk.")
                .font(.footnote)
                .foregroundStyle(.secondary)
                .padding(14)
        }
    }

    private var applyCard: some View {
        ApplicationNotebookCard(
            title: "Apply", summary: model.kept ? model.pdfName : "Keep the CV first", done: false, status: nil,
            isOpen: open.contains(.apply), toggle: { toggle(.apply) }
        ) {
            VStack(spacing: 0) {
                if let pdf = model.pdfURL {
                    ApplicationActionRow(title: "Preview PDF", systemImage: "eye") { previewURL = pdf }
                    Divider().padding(.leading, 42)
                    ShareLink(item: pdf) { ApplicationActionRow(title: "Download PDF", systemImage: "arrow.down.doc") }
                        .buttonStyle(.plain)
                }
                Button {
                    applying = true
                } label: {
                    Label("Apply on the listing", systemImage: "paperplane").frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .buttonBorderShape(.roundedRectangle(radius: 12))
                .controlSize(.large)
                .disabled(!model.kept)
                .padding(14)
            }
        }
    }

    private func toggle(_ card: Card) {
        if open.contains(card) { open.remove(card) } else { open.insert(card) }
    }
}

#Preview {
    let job = DueJob.sample
    NavigationStack { ApplicationView(model: ApplicationModel(client: .mock(), job: job, draft: job.draft!)) }
}
