import SwiftUI

struct ShortlistedRow: View {
    let count: Int
    let action: () -> Void

    @ViewBuilder
    var body: some View {
        if count > 0 {
            Button(action: action) {
                HStack(spacing: 10) {
                    Text("Shortlisted").font(.subheadline)
                    Text(count, format: .number).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
                    Spacer(minLength: 4)
                    Image(systemName: "chevron.right").font(.caption.weight(.semibold)).foregroundStyle(.tertiary)
                }
                .padding(.horizontal, 14)
                .frame(minHeight: 44)
                .contentShape(.rect)
                .hairlineCard()
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Shortlisted,\(count)")
        }
    }
}
