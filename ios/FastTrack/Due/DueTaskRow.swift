import QuickLook
import SwiftUI

struct DueTaskRow: View {
    let row: DueRow
    let job: DueJob
    let model: DueModel
    let openJob: (DueJob) -> Void
    let openApplication: (DueJob) -> Void
    @State private var struck = false
    @State private var previewURL: URL?

    var body: some View {
        HStack(spacing: 0) {
            leading
                .frame(width: 44, height: 44)
            Menu {
                menuItems
            } label: {
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline)
                        .strikethrough(struck)
                        .foregroundStyle(struck ? .secondary : .primary)
                        .multilineTextAlignment(.leading)
                    meta
                }
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                .padding(.vertical, 8)
                .contentShape(.rect)
            } primaryAction: {
                if opensApplication { openApplication(job) }
            }
            .buttonStyle(.plain)
        }
        .padding(.leading, 1)
        .padding(.trailing, 14)
        .quickLookPreview($previewURL)
    }

    private var opensApplication: Bool {
        if case .draft(let draft) = row { draft.state == .ready } else { false }
    }

    private var title: String {
        switch row {
        case .chase: "Chase the application"
        case .draft(let draft): draft.state == .ready ? "Apply with tailored CV" : "Tailored CV"
        case .task(let task): task.typeName
        }
    }

    @ViewBuilder
    private var meta: some View {
        switch row {
        case .chase:
            let days = model.overdueDays(job)
            HStack(spacing: 6) {
                DotLabel(text: days > 0 ? "Overdue \(days)d" : "Today", hue: days > 0 ? .red : .orange)
                if let chaseBy = job.chaseBy {
                    Text(chaseBy.formatted(.dateTime.day().month(.abbreviated).locale(Locale(identifier: "en_GB"))))
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(.secondary)
                }
            }
        case .draft(let draft):
            switch draft.state {
            case .ready:
                HStack(spacing: 6) {
                    DotLabel(text: "CV ready", hue: .brandTint)
                    Text(draft.cvTitle).font(.caption).foregroundStyle(.secondary)
                }
            case .failed: DotLabel(text: "Couldn't draft", hue: .red)
            case .pending: Text("Drafting…").font(.caption).foregroundStyle(.secondary)
            }
        case .task(let task):
            if !task.notes.isEmpty { Text(task.notes).font(.caption).foregroundStyle(.secondary).lineLimit(2) }
        }
    }

    @ViewBuilder
    private var leading: some View {
        switch row {
        case .draft(let draft):
            switch draft.state {
            case .pending:
                ProgressView().controlSize(.small).accessibilityLabel("Drafting")
            case .failed:
                Button { Task { await model.retryDraft(job) } } label: {
                    RoundedRectangle(cornerRadius: 5, style: .continuous)
                        .strokeBorder(.red, lineWidth: 1.5)
                        .overlay { Image(systemName: "arrow.clockwise").font(.caption2.weight(.bold)).foregroundStyle(.red) }
                        .frame(width: 18, height: 18)
                        .frame(width: 44, height: 44)
                        .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Retry draft")
            case .ready:
                if job.applyTaskID != nil {
                    checkbox.accessibilityLabel("Mark applied")
                } else {
                    Image(systemName: "doc.text").font(.caption).foregroundStyle(Color.brandTint).accessibilityHidden(true)
                }
            }
        case .chase, .task:
            checkbox.accessibilityLabel("Complete: \(title)")
        }
    }

    private var checkbox: some View {
        Button(action: complete) {
            RoundedRectangle(cornerRadius: 5, style: .continuous)
                .strokeBorder(struck ? Color.brandTint : Color(.tertiaryLabel), lineWidth: 1.5)
                .background(struck ? Color.brandTint : .clear, in: .rect(cornerRadius: 5, style: .continuous))
                .overlay {
                    if struck { Image(systemName: "checkmark").font(.caption2.weight(.bold)).foregroundStyle(.white) }
                }
                .frame(width: 18, height: 18)
                .frame(width: 44, height: 44)
                .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .sensoryFeedback(.success, trigger: struck) { _, new in new }
    }

    @ViewBuilder
    private var menuItems: some View {
        switch row {
        case .chase:
            Button("Mark chased", systemImage: "checkmark", action: complete)
            Menu("Snooze", systemImage: "clock") {
                ForEach([(2, "2 days"), (7, "1 week"), (14, "2 weeks")], id: \.0) { days, label in
                    Button(label) { _ = withAnimation(.snappy) { model.snooze(job, days: days) } }
                }
            }
        case .task:
            Button("Mark done", systemImage: "checkmark", action: complete)
        case .draft(let draft):
            switch draft.state {
            case .ready:
                Button("Open application", systemImage: "doc.text") { openApplication(job) }
                Button("View PDF", systemImage: "doc.richtext") { Task { previewURL = await model.pdf(draft) } }
                if job.applyTaskID != nil { Button("Mark applied", systemImage: "checkmark", action: complete) }
            case .failed:
                Button("Retry draft", systemImage: "arrow.clockwise") { Task { await model.retryDraft(job) } }
            case .pending:
                EmptyView()
            }
        }
        Divider()
        Button("Open job", systemImage: "arrow.up.right") { openJob(job) }
    }

    private func complete() {
        guard !struck else { return }
        struck = true
        Task {
            try? await Task.sleep(for: .milliseconds(500))
            withAnimation(.snappy(duration: 0.25)) {
                switch row {
                case .chase: _ = model.completeChase(job)
                case .task(let task): _ = model.complete(task, in: job)
                case .draft: _ = model.markApplied(job)
                }
            }
            struck = false
        }
    }
}
