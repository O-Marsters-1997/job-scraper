#if DEBUG
import SwiftUI

struct TactileCard<Header: View>: View {
    let job: PrototypeJob
    let isExpanded: Bool
    let toggle: () -> Void
    var radius: CGFloat = 14
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
                        .padding(16)
                    }
                } else {
                    Button(action: toggle) {
                        VStack(alignment: .leading, spacing: 0) {
                            MetaLines(job: job)
                                .padding(.horizontal, 16)
                                .padding(.top, 12)
                                .padding(.bottom, 16)
                            ForEach(job.topRows, id: \.self) { row in
                                Divider().padding(.leading, row == job.topRows.first ? 0 : 46)
                                BreakdownRowView(row: row).padding(.horizontal, 16)
                            }
                            Spacer(minLength: 0)
                        }
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
        .background(Color(.systemBackground))
        .clipShape(.rect(cornerRadius: radius, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: radius, style: .continuous).strokeBorder(Color(.separator)))
    }
}

struct DeckEdges: View {
    let count: Int
    var radius: CGFloat = 14

    var body: some View {
        ZStack(alignment: .bottom) {
            ForEach((0..<count).reversed(), id: \.self) { i in
                RoundedRectangle(cornerRadius: radius, style: .continuous)
                    .fill(Color(.systemBackground))
                    .overlay(RoundedRectangle(cornerRadius: radius, style: .continuous).strokeBorder(Color(.separator)))
                    .frame(height: 40)
                    .scaleEffect(x: 1 - CGFloat(i + 1) * 0.05)
                    .offset(y: CGFloat(i + 1) * 6)
            }
        }
        .animation(.snappy(duration: 0.22), value: count)
        .accessibilityHidden(true)
    }
}
#endif
