#if DEBUG
import SwiftUI

struct JobDetailBody: View {
    let job: PrototypeJob
    private let order: [PrototypeJob.Effect] = [.gated, .boost, .meets, .misses, .level, .unknown]

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            section("Description") {
                Text(job.description)
                    .font(.body)
                    .lineSpacing(3)
            }
            section("Why this score") {
                VStack(alignment: .leading, spacing: 16) {
                    ForEach(order, id: \.self) { effect in
                        let rows = job.rows.filter { $0.effect == effect }
                        if !rows.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Text(effect.groupTitle)
                                    .font(.footnote.weight(.medium))
                                    .foregroundStyle(.secondary)
                                ForEach(rows, id: \.self) { BreakdownRowView(row: $0) }
                            }
                        }
                    }
                }
            }
            section("Note") {
                Text("No note yet").foregroundStyle(.secondary)
            }
            section("Recruiter contacts") {
                Text("None logged").foregroundStyle(.secondary)
            }
            Link(destination: URL(string: "https://example.com/jobs/\(job.id)")!) {
                Label("Open original posting", systemImage: "arrow.up.right")
                    .font(.subheadline.weight(.medium))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func section(_ title: String, @ViewBuilder content: () -> some View) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title).font(.headline)
            content()
        }
    }
}
#endif
