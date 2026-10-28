import SwiftUI

struct BreakdownRowView: View {
    let row: ScoreRow

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: row.effect.symbol)
                .foregroundStyle(row.effect.colour)
                .frame(width: 20)
                .accessibilityHidden(true)
            Text(row.effect.isGate ? "Gate: \(row.displayLabel)" : row.displayLabel)
        }
        .font(.subheadline)
        .frame(maxWidth: .infinity, minHeight: 40, alignment: .leading)
    }
}
