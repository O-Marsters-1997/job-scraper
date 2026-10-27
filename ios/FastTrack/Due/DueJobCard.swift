import SwiftUI

enum DueRow: Identifiable {
    case chase(DueJob)
    case draft(DueJob.Draft)
    case task(DueJob.Task)
    case apply(DueJob)

    var id: String {
        switch self {
        case .apply(let job): "apply-\(job.id)"
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
    let logContact: (DueJob) -> Void
    let openJob: (DueJob) -> Void
    let openApplication: (DueJob) -> Void

    private var rows: [DueRow] {
        (model.hasChaseDue(job) ? [.chase(job)] : [])
            + (job.isAtDesk ? [] : job.draft.map { [.draft($0)] } ?? [])
            + (job.showsApplyRow ? [.apply(job)] : [])
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
                        DueTaskRow(row: row, job: job, model: model, logContact: logContact, openJob: openJob, openApplication: openApplication)
                    }
                }
                .padding(.bottom, 4)
                .transition(.opacity.animation(.easeOut(duration: 0.12)))
            }
        }
        .clipShape(.rect(cornerRadius: 14, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous).strokeBorder(Color(.separator)))
    }

    private var header: some View {
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
        switch job.draft?.state {
        case .failed: return ("Draft failed", .red)
        case .ready: return ("CV ready", .brandTint)
        default: return nil
        }
    }
}
