#if DEBUG
import SwiftUI

struct VariantTactile: View {
    let store: TriagePrototypeStore

    private var shortlisted: Int { store.log.filter { $0.removed && $0.action.isPositive }.count }
    private var behind: Int { min(max(store.queue.count - 1, 0), 3) }

    var body: some View {
        VStack(spacing: 20) {
            ZStack(alignment: .bottom) {
                ForEach((0..<behind).reversed(), id: \.self) { i in
                    RoundedRectangle(cornerRadius: 20, style: .continuous)
                        .fill(Color(.secondarySystemGroupedBackground))
                        .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
                        .frame(height: 40)
                        .scaleEffect(x: 1 - CGFloat(i + 1) * 0.05)
                        .offset(y: CGFloat(i + 1) * 7)
                }
                .animation(.snappy, value: behind)
                TriageCardHost(store: store) { job, isExpanded, toggle in
                    card(job, isExpanded: isExpanded, toggle: toggle)
                }
            }
            .padding(.horizontal)
            .padding(.bottom, CGFloat(behind) * 7)
            if let job = store.current {
                buttons(job)
            }
        }
        .padding(.bottom, 8)
        .navigationTitle("Triage")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func card(_ job: PrototypeJob, isExpanded: Bool, toggle: @escaping () -> Void) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Text(job.band.rawValue)
                    .font(.title3.weight(.bold))
                    .foregroundStyle(job.band.colour)
                Text(job.score, format: .number)
                    .font(.title3.monospacedDigit())
                    .foregroundStyle(.secondary)
                Spacer()
                Text(job.age).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
            }
            .padding(.horizontal, 22)
            .padding(.vertical, 16)
            .background(job.band.colour.opacity(0.1))
            Group {
                if isExpanded {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            MetaLines(job: job)
                            JobDetailBody(job: job)
                        }
                        .padding(22)
                    }
                } else {
                    VStack(alignment: .leading, spacing: 20) {
                        MetaLines(job: job)
                        VStack(alignment: .leading, spacing: 10) {
                            ForEach(job.topRows, id: \.self) { BreakdownRowView(row: $0) }
                        }
                        Spacer(minLength: 0)
                    }
                    .padding(22)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                    .contentShape(.rect)
                    .onTapGesture(perform: toggle)
                }
            }
            Button(action: toggle) {
                Image(systemName: "chevron.compact.\(isExpanded ? "up" : "down")")
                    .font(.title2)
                    .foregroundStyle(.tertiary)
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityLabel(isExpanded ? "Show summary" : "Show details")
        }
        .frame(maxHeight: .infinity)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(.rect(cornerRadius: 20, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
    }

    private func buttons(_ job: PrototypeJob) -> some View {
        HStack(spacing: 28) {
            Button { act(.dismiss, job) } label: {
                Image(systemName: "xmark")
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .frame(width: 64, height: 64)
                    .background(Color(.secondarySystemGroupedBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator)))
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Dismiss")
            TriageMoreMenu(job: job, store: store) {
                Image(systemName: "ellipsis")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .frame(width: 48, height: 48)
                    .background(Color(.secondarySystemGroupedBackground), in: .circle)
                    .overlay(Circle().strokeBorder(Color(.separator)))
                    .accessibilityLabel("More")
            }
            .tint(.secondary)
            Button { act(.shortlist, job) } label: {
                Image(systemName: "checkmark")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.white)
                    .symbolEffect(.bounce, value: shortlisted)
                    .frame(width: 64, height: 64)
                    .background(Color.brandTint, in: .circle)
            }
            .accessibilityLabel("Shortlist")
        }
    }

    private func act(_ action: TriagePrototypeStore.Action, _ job: PrototypeJob) {
        withAnimation(.snappy) { store.act(action, on: job) }
    }
}
#endif
