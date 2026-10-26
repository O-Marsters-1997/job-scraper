import SwiftUI

struct Wordmark: View {
    var body: some View {
        HStack(spacing: 10) {
            FastTrackMark()
                .foregroundStyle(AuthPalette.bright)
                .frame(width: 19, height: 19)
                .frame(width: 34, height: 34)
                .background(.white.opacity(0.1), in: .rect(cornerRadius: 10, style: .continuous))
                .overlay(RoundedRectangle(cornerRadius: 10, style: .continuous).strokeBorder(.white.opacity(0.2)))
            (Text("Fast").foregroundStyle(AuthPalette.ink) + Text("Track").foregroundStyle(AuthPalette.bright))
                .font(.headline.weight(.bold))
                .tracking(-0.35)
        }
    }
}

private struct FastTrackMark: View {
    var body: some View {
        Canvas { context, size in
            let unit = min(size.width, size.height) / 32
            let scale = CGAffineTransform(scaleX: unit, y: unit)
            var f = Path()
            f.addLines([CGPoint(x: 9, y: 6), CGPoint(x: 24, y: 6), CGPoint(x: 24, y: 10.5), CGPoint(x: 13.5, y: 10.5),
                        CGPoint(x: 13.5, y: 14), CGPoint(x: 21, y: 14), CGPoint(x: 21, y: 18.5), CGPoint(x: 13.5, y: 18.5),
                        CGPoint(x: 13.5, y: 26), CGPoint(x: 9, y: 26)])
            f.closeSubpath()
            let skew = CGAffineTransform(a: 1, b: 0, c: tan(-13 * .pi / 180), d: 1, tx: 0, ty: 0)
            let fTransform = CGAffineTransform(translationX: 3.4, y: 0).concatenating(skew).concatenating(scale)
            context.fill(f.applying(fTransform), with: .foreground)
            var lines = Path()
            lines.move(to: CGPoint(x: 2.5, y: 9)); lines.addLine(to: CGPoint(x: 7, y: 9))
            lines.move(to: CGPoint(x: 2, y: 17)); lines.addLine(to: CGPoint(x: 6.5, y: 17))
            context.stroke(lines.applying(scale), with: .foreground, style: StrokeStyle(lineWidth: 2.3 * unit, lineCap: .round))
        }
        .accessibilityHidden(true)
    }
}
