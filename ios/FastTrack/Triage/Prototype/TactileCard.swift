#if DEBUG
import SwiftUI

struct TactileCard<Header: View>: View {
    let job: PrototypeJob
    let isExpanded: Bool
    let toggle: () -> Void
    var radius: CGFloat = 20
    @ViewBuilder let header: () -> Header

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header()
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
                    Button(action: toggle) {
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
                    }
                    .buttonStyle(.plain)
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
        .clipShape(.rect(cornerRadius: radius, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: radius, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
    }
}

struct DeckEdges: View {
    let colours: [Color]
    var radius: CGFloat = 20

    var body: some View {
        ZStack(alignment: .bottom) {
            ForEach(Array(colours.enumerated()).reversed(), id: \.offset) { i, colour in
                RoundedRectangle(cornerRadius: radius, style: .continuous)
                    .fill(Color(.secondarySystemGroupedBackground))
                    .overlay(RoundedRectangle(cornerRadius: radius, style: .continuous).fill(colour))
                    .overlay(RoundedRectangle(cornerRadius: radius, style: .continuous).strokeBorder(Color(.separator).opacity(0.6)))
                    .frame(height: 40)
                    .scaleEffect(x: 1 - CGFloat(i + 1) * 0.05)
                    .offset(y: CGFloat(i + 1) * 7)
            }
        }
        .animation(.snappy, value: colours.count)
        .accessibilityHidden(true)
    }
}
#endif
