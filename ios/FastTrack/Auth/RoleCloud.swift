import SwiftUI

struct CompactRoleCloud: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var drift = false

    private struct Pill {
        let role: String, source: String, hue: Color, x: CGFloat, y: CGFloat, z: CGFloat
    }

    private let pills = [
        Pill(role: "Frontend Engineer", source: "LinkedIn", hue: Color(hex: 0x74a1ff), x: 0.30, y: 0.18, z: 0.85),
        Pill(role: "Product Designer", source: "Greenhouse", hue: Color(hex: 0x65d999), x: 0.74, y: 0.28, z: 0.35),
        Pill(role: "Backend, Go", source: "Lever", hue: Color(hex: 0xff936a), x: 0.28, y: 0.62, z: 0.5),
        Pill(role: "ML Engineer", source: "Indeed", hue: Color(hex: 0xbb91ff), x: 0.70, y: 0.66, z: 0.95),
    ]

    var body: some View {
        GeometryReader { geo in
            ForEach(Array(pills.enumerated()), id: \.element.role) { i, p in
                RolePill(role: p.role, source: p.source, hue: p.hue)
                    .scaleEffect(0.5 + 0.45 * p.z)
                    .opacity(0.55 + 0.45 * p.z)
                    .blur(radius: max(0, 0.55 - p.z) * 5)
                    .offset(x: drift ? (i % 2 == 1 ? -18 : 18) : 0, y: drift ? (i % 3 == 0 ? -12 : 12) : 0)
                    .animation(reduceMotion ? nil : .easeInOut(duration: 11 + Double(i) * 1.7).repeatForever(autoreverses: true), value: drift)
                    .position(x: geo.size.width * p.x, y: geo.size.height * p.y)
                    .zIndex(Double(p.z))
            }
        }
        .allowsHitTesting(false)
        .onAppear { drift = !reduceMotion }
    }
}

private struct RolePill: View {
    let role: String
    let source: String
    let hue: Color

    var body: some View {
        HStack(spacing: 9) {
            RoundedRectangle(cornerRadius: 5, style: .continuous)
                .fill(hue.opacity(0.5))
                .frame(width: 15, height: 15)
            Text(role)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(AuthPalette.ink)
            Text(source.uppercased())
                .font(.system(size: 10, design: .monospaced))
                .tracking(0.5)
                .foregroundStyle(hue)
                .padding(.leading, -2)
        }
        .padding(.horizontal, 12)
        .frame(height: 31)
        .background(.white.opacity(0.08), in: .capsule)
        .overlay(Capsule().strokeBorder(.white.opacity(0.16)))
        .fixedSize()
    }
}
