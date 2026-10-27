import SwiftUI

struct ShortlistedView: View {
    let model: DueModel
    @State private var query = ""
    @State private var openJobID: String?
    @State private var applying: DueJob?
    @State private var drafting: Set<String> = []

    var body: some View {
        let jobs = model.shortlisted(matching: query)
        Group {
            if model.shortlisted.isEmpty {
                ContentUnavailableView(
                    "All drafted", systemImage: "checkmark.circle",
                    description: Text("Shortlisted jobs without a CV draft show up here."))
            } else if jobs.isEmpty {
                ContentUnavailableView.search(text: query)
            } else {
                ScrollView {
                    VStack(spacing: 10) {
                        ForEach(jobs) { job in
                            card(job)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.top, 8)
                    .padding(.bottom, 24)
                    .animation(.snappy(duration: 0.25), value: jobs)
                }
            }
        }
        .background(Color(.systemBackground))
        .navigationTitle("Shortlisted")
        .navigationBarTitleDisplayMode(.inline)
        .clearingSearchable(text: $query)
        .navigationDestination(item: $openJobID) { JobDetailView(id: $0) }
        .sheet(item: $applying) { job in
            ShortlistedApplySheet(job: job) {
                await model.markApplied(job).value
                return model.jobs.first { $0.id == job.id }?.applyTaskID == nil
            }
        }
        .safeAreaInset(edge: .bottom, spacing: 8) {
            DueUndoBanner(entry: model.banner, onUndo: { _ = model.undo() }, onExpire: model.dismissBanner)
                .animation(.snappy, value: model.banner?.id)
        }
        .alert("Couldn't save that", isPresented: Bindable(model).hasFailure) {
            Button("OK") {}
        } message: {
            Text(model.failureMessage)
        }
    }

    private func card(_ job: DueJob) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Button { openJobID = job.id } label: {
                VStack(alignment: .leading, spacing: 6) {
                    HStack(alignment: .firstTextBaseline, spacing: 10) {
                        Text(job.companyName).font(.system(.title3, design: .serif))
                        HStack(alignment: .firstTextBaseline, spacing: 4) {
                            if let band = job.band {
                                Text(band.title).font(.caption.weight(.semibold)).foregroundStyle(band.colour)
                            }
                            if let score = job.score {
                                Text(score, format: .number).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                            }
                        }
                        Spacer(minLength: 6)
                    }
                    Text(job.title).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
                }
                .padding(14)
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityHint("Opens the job")
            Divider()
            HStack(spacing: 8) {
                Button("Draft CV", systemImage: "doc.text") { draft(job) }
                    .disabled(drafting.contains(job.id))
                Button("Apply", systemImage: "arrow.up.right") { applying = job }
            }
            .buttonStyle(.bordered)
            .tint(.brandTint)
            .frame(minHeight: 44)
            .padding(10)
        }
        .clipShape(.rect(cornerRadius: 14, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous).strokeBorder(Color(.separator)))
    }

    private func draft(_ job: DueJob) {
        drafting.insert(job.id)
        Task {
            await model.retryDraft(job)
            drafting.remove(job.id)
        }
    }
}
