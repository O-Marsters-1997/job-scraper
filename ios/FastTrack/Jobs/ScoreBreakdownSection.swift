import SwiftUI

struct ScoreBreakdownSection: View {
    let rows: [ScoreRow]

    var body: some View {
        Section("Breakdown") {
            ForEach(rows) { row in
                Label(row.label, systemImage: row.meets ? "checkmark.circle" : "minus.circle")
                    .foregroundStyle(row.meets ? .primary : .secondary)
            }
        }
    }
}
