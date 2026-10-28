import SwiftUI

extension View {
    func hairlineCard(radius: CGFloat = 14) -> some View {
        background(Color(.systemBackground), in: .rect(cornerRadius: radius, style: .continuous))
            .clipShape(.rect(cornerRadius: radius, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: radius, style: .continuous).strokeBorder(Color(.separator)))
    }
}
