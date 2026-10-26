import SwiftUI

struct JobDetailBody: View {
    let job: Job
    @Environment(\.apiClient) private var client
    @State private var detail: Job?

    private static let order: [ScoreEffect] = [.gated, .blocked, .favourite, .meets, .misses, .level, .unknown]

    private var shown: Job { detail ?? job }

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            if !shown.description.isEmpty {
                section("Description") {
                    Text(shown.description).font(.body).lineSpacing(3)
                }
            }
            section("Why this score") {
                VStack(alignment: .leading, spacing: 16) {
                    ForEach(Self.order, id: \.self) { effect in
                        let rows = shown.breakdown.filter { $0.effect == effect }
                        if !rows.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Text(effect.groupTitle)
                                    .font(.footnote.weight(.medium))
                                    .foregroundStyle(.secondary)
                                ForEach(rows, id: \.key) { BreakdownRowView(row: $0) }
                            }
                        }
                    }
                }
            }
            if let url = URL(string: shown.url) {
                Link(destination: url) {
                    Label("Open original posting", systemImage: "arrow.up.right")
                        .font(.subheadline.weight(.medium))
                        .frame(minHeight: 44)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .task(id: job.id) { detail = try? await client.job(job.id) }
    }

    private func section(_ title: String, @ViewBuilder content: () -> some View) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title).font(.headline)
            content()
        }
    }
}
