import SwiftUI

struct DotLabel: View {
    let text: String
    let hue: Color

    var body: some View {
        HStack(spacing: 6) {
            Circle().fill(hue).frame(width: 6, height: 6)
            Text(text).font(.caption.weight(.medium).monospacedDigit())
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 2)
        .background(hue.opacity(0.1), in: .rect(cornerRadius: 5, style: .continuous))
        .accessibilityElement(children: .combine)
    }
}
