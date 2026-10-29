import SwiftUI

struct UndoBanner: View {
    let decision: TriageDecision?
    let onUndo: () -> Void
    let isPaused: Bool
    let onExpire: (UUID) -> Void
    let onDraftCV: (TriageDecision) -> Void
    let onWhy: (TriageDecision) -> Void

    var body: some View {
        if let decision {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(verb(decision.action)).font(.footnote.weight(.semibold))
                    Text(decision.job.title).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                if decision.action == .shortlist(draftCV: false) {
                    Button("Draft CV") { onDraftCV(decision) }
                        .font(.footnote.weight(.semibold))
                        .frame(minHeight: 44)
                }
                if case .dismiss(let reasons) = decision.action, reasons.isEmpty {
                    Button("Why?") { onWhy(decision) }
                        .font(.footnote.weight(.semibold))
                        .frame(minHeight: 44)
                }
                Button("Undo", action: onUndo)
                    .font(.footnote.weight(.semibold))
                    .frame(minHeight: 44)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 2)
            .hairlineCard()
            .padding(.horizontal)
            .transition(.move(edge: .bottom).combined(with: .opacity))
            .task(id: isPaused ? nil : decision.id) {
                guard !isPaused else { return }
                do { try await Task.sleep(for: .seconds(4)) } catch { return }
                withAnimation { onExpire(decision.id) }
            }
        }
    }

    private func verb(_ action: TriageAction) -> String {
        switch action {
        case .shortlist(let draftCV): draftCV ? "Shortlisted, drafting CV" : "Shortlisted"
        case .dismiss: "Dismissed"
        case .skip: "Skipped"
        case .excludeCompany: "Excluded company"
        case .reclassify(let kind): kind.isAgency ? "Marked as agency" : "Marked as direct"
        case .note: "Note added"
        }
    }
}
