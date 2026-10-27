import SwiftUI

struct ShortlistedView: View {
    let model: DueModel
    @State private var query = ""
    @State private var openJobID: String?
    @State private var applying: DueJob?
    @State private var drafting: Set<String> = []
    @ScaledMetric(relativeTo: .subheadline) private var actionSize = 36.0

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
        .searchable(text: $query)
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
                        DueCardHeadline(job: job)
                        Spacer(minLength: 6)
                        Image(systemName: "chevron.right")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.tertiary)
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
            HStack(spacing: 20) {
                Button { draft(job) } label: {
                    action("Draft CV", systemImage: "doc.badge.plus", primary: true, busy: drafting.contains(job.id))
                }
                .disabled(drafting.contains(job.id))
                Button { applying = job } label: {
                    action("Apply", systemImage: "arrow.up.right", primary: false, busy: false)
                }
                Spacer(minLength: 0)
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
        }
        .hairlineCard()
    }

    private func action(_ title: String, systemImage: String, primary: Bool, busy: Bool) -> some View {
        HStack(spacing: 8) {
            Group {
                if busy {
                    ProgressView().controlSize(.small)
                } else {
                    Image(systemName: systemImage)
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(primary ? Color.white : Color.secondary)
                }
            }
            .frame(width: actionSize, height: actionSize)
            .background(primary ? Color.brandTint : Color(.systemBackground), in: .circle)
            .overlay { if !primary { Circle().strokeBorder(Color(.separator)) } }
            .accessibilityHidden(true)
            Text(title).font(.subheadline)
        }
        .frame(minHeight: 44)
        .contentShape(.rect)
    }

    private func draft(_ job: DueJob) {
        drafting.insert(job.id)
        Task {
            await model.retryDraft(job)
            drafting.remove(job.id)
        }
    }
}
