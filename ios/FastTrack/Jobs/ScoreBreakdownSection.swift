import SwiftUI

struct ScoreBreakdownSection: View {
    let rows: [ScoreRow]

    var body: some View {
        Section("Breakdown") {
            ForEach(rows, id: \.key) { row in
                BreakdownRowView(row: row)
            }
        }
    }
}
