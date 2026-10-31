import SwiftUI

enum DueRow: Identifiable {
    case chase(DueJob)
    case draft(DueJob.Draft)
    case task(DueJob.Task)

    var id: String {
        switch self {
        case .chase(let job): "chase-\(job.id)"
        case .draft(let draft): "draft-\(draft.id)"
        case .task(let task): "task-\(task.id)"
        }
    }
}

struct DueJobCard: View {
    let job: DueJob
    let model: DueModel
    let isOpen: Bool
    let toggle: () -> Void
    let openJob: (DueJob) -> Void
    let openApplication: (DueJob) -> Void
    @State private var confirmingDelete = false

    private var rows: [DueRow] {
        (model.hasChaseDue(job) ? [.chase(job)] : [])
            + (job.isAtDesk ? [] : job.dueDraft.map { [.draft($0)] } ?? [])
            + job.tasks.map(DueRow.task)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Button(action: toggle) { header }
                .buttonStyle(.plain)
                .accessibilityHint(isOpen ? "Collapse" : "Show \(rows.count) to do")
            if isOpen {
                VStack(spacing: 0) {
                    ForEach(rows) { row in
                        Divider()
                        DueTaskRow(row: row, job: job, model: model, openJob: openJob, openApplication: openApplication)
                    }
                }
                .padding(.bottom, 4)
                .transition(.opacity.animation(.easeOut(duration: 0.12)))
            }
        }
        .hairlineCard()
        .iconSwipeActions([
            .deleteApplication {
                if job.hasCV { confirmingDelete = true } else { delete() }
            }
        ])
        .confirmationDialog("Delete this application?", isPresented: $confirmingDelete, titleVisibility: .visible) {
            Button("Delete application", role: .destructive, action: delete)
        } message: {
            Text("This can't be undone.")
        }
    }

    private func delete() {
        Task { await model.delete(job) }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                DueCardHeadline(job: job)
                Spacer(minLength: 6)
                Text(rows.count, format: .number).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.tertiary)
                    .rotationEffect(.degrees(isOpen ? 90 : 0))
            }
            HStack(spacing: 8) {
                Text(job.title).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
                Spacer(minLength: 4)
                if let urgent { DotLabel(text: urgent.0, hue: urgent.1).opacity(isOpen ? 0 : 1).accessibilityHidden(isOpen) }
            }
        }
        .padding(14)
        .frame(minHeight: 44)
        .contentShape(.rect)
    }

    private var urgent: (String, Color)? {
        if model.hasChaseDue(job) {
            let days = model.overdueDays(job)
            return days > 0 ? ("\(days)d overdue", .red) : ("Chase today", .orange)
        }
        if job.isAtDesk { return ("At desk", .secondary) }
        switch job.dueDraft?.state {
        case .failed: return ("Draft failed", .red)
        case .ready: return ("CV ready", .brandTint)
        default: return nil
        }
    }
}

struct DueCardHeadline: View {
    let job: DueJob

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(job.companyName).font(.system(.title3, design: .serif))
            if job.band != nil || job.score != nil || job.companyKind.badge != nil {
                ViewThatFits(in: .horizontal) {
                    HStack(alignment: .firstTextBaseline, spacing: 6) { chips }
                    VStack(alignment: .leading, spacing: 4) { chips }
                }
            }
        }
    }

    @ViewBuilder private var chips: some View {
        if let band = job.band { DotLabel(text: band.title, hue: band.colour) }
        if let score = job.score {
            Text(score, format: .number).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
        }
        CompanyKindBadge(kind: job.companyKind).lineLimit(1).fixedSize()
    }
}
