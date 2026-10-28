import SwiftUI

struct TriageCard: View {
    let job: Job
    let bandReason: String?
    let isExpanded: Bool
    let toggle: () -> Void
    @Environment(\.dynamicTypeSize) private var typeSize

    private var rows: [ScoreRow] {
        typeSize.isAccessibilitySize ? Array(job.gate.map { [$0] } ?? []) : job.topRows
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            BandRow(job: job)
                .padding(.horizontal, 16)
                .padding(.top, 16)
            if isExpanded {
                ScrollView {
                    VStack(alignment: .leading, spacing: 24) {
                        if let bandReason {
                            Text(bandReason)
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                        }
                        MetaLines(job: job, limitsTitle: false)
                        JobDetailBody(job: job)
                    }
                    .padding(16)
                }
            } else {
                Button(action: toggle) {
                    VStack(alignment: .leading, spacing: 0) {
                        MetaLines(job: job, limitsTitle: !typeSize.isAccessibilitySize)
                            .padding(.horizontal, 16)
                            .padding(.top, 12)
                            .padding(.bottom, 16)
                        ForEach(rows, id: \.key) { row in
                            Divider().padding(.leading, row == rows.first ? 0 : 46)
                            BreakdownRowView(row: row).padding(.horizontal, 16)
                        }
                        Spacer(minLength: 0)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                    .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .accessibilityHidden(true)
            }
            Button(action: toggle) {
                Image(systemName: isExpanded ? "chevron.compact.up" : "chevron.compact.down")
                    .font(.title2)
                    .foregroundStyle(.tertiary)
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityLabel(isExpanded ? "Show summary" : "Show details")
            .accessibilityHidden(!isExpanded)
        }
        .frame(maxHeight: .infinity)
        .hairlineCard()
        .accessibilityElement(children: isExpanded ? .contain : .ignore)
        .accessibilityLabel(isExpanded ? job.title : job.accessibilitySummary)
    }
}

struct BandRow: View {
    let job: Job

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 6) {
            if let band = job.band {
                DotLabel(text: band.title, hue: band.colour)
            }
            if let score = job.score {
                Text(score, format: .number).font(.footnote.weight(.medium).monospacedDigit())
            }
            CompanyKindBadge(kind: job.companyKind)
            Spacer(minLength: 6)
            Text(job.age).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
        }
        .accessibilityElement(children: .combine)
    }
}

struct MetaLines: View {
    let job: Job
    let limitsTitle: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(job.companyName)
                .font(.system(.title, design: .serif))
            Text(job.title)
                .font(.headline)
                .lineLimit(limitsTitle ? 3 : nil)
            Text(job.metaLine)
                .font(.footnote.monospacedDigit())
                .foregroundStyle(.secondary)
                .padding(.top, 2)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct DeckEdges: View {
    let count: Int

    var body: some View {
        ZStack(alignment: .bottom) {
            ForEach((0..<count).reversed(), id: \.self) { i in
                Color.clear
                    .hairlineCard()
                    .frame(height: 40)
                    .scaleEffect(x: 1 - CGFloat(i + 1) * 0.05)
                    .offset(y: CGFloat(i + 1) * 6)
            }
        }
        .animation(.snappy(duration: 0.22), value: count)
        .accessibilityHidden(true)
    }
}
