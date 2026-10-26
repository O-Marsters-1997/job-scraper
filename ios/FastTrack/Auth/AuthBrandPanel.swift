import SwiftUI

enum AuthPalette {
    static let p1 = Color(hex: 0x2b2a46)
    static let p2 = Color(hex: 0x0c0a1d)
    static let bright = Color(hex: 0xa19ee9)
    static let ink = Color(hex: 0xf5f4fc)
    static let ribbons = [Color(hex: 0x7c73dc), Color(hex: 0x6d5ed5), Color(hex: 0x8d87e2)]
}

struct AuthBrandPanel: View {
    let topInset: CGFloat
    let cloudTop: CGFloat

    var body: some View {
        ZStack(alignment: .topLeading) {
            LinearGradient(colors: [AuthPalette.p1, AuthPalette.p2], startPoint: UnitPoint(x: 0.33, y: 0), endPoint: UnitPoint(x: 0.67, y: 1))
            AuthAurora()
            CompactRoleCloud()
                .padding(.top, cloudTop)
            Wordmark()
                .padding(.leading, 20)
                .padding(.top, topInset)
        }
        .clipped()
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("FastTrack")
    }
}

private struct AuthAurora: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var drift = false

    private struct Ribbon {
        let size: CGSize, origin: CGPoint, to: CGPoint, scale: CGFloat, duration: Double
    }

    private let ribbons = [
        Ribbon(size: CGSize(width: 0.70, height: 0.55), origin: CGPoint(x: -0.10, y: -0.08), to: CGPoint(x: 0.15, y: 0.18), scale: 1.2, duration: 17),
        Ribbon(size: CGSize(width: 0.62, height: 0.60), origin: CGPoint(x: 0.53, y: 0.18), to: CGPoint(x: -0.12, y: 0.12), scale: 1.15, duration: 21),
        Ribbon(size: CGSize(width: 0.55, height: 0.52), origin: CGPoint(x: 0.14, y: 0.66), to: CGPoint(x: 0.10, y: -0.16), scale: 1.25, duration: 19),
    ]

    var body: some View {
        GeometryReader { geo in
            let field = CGSize(width: geo.size.width * 1.4, height: geo.size.height * 1.4)
            ZStack(alignment: .topLeading) {
                ForEach(ribbons.indices, id: \.self) { i in
                    let r = ribbons[i]
                    let w = field.width * r.size.width, h = field.height * r.size.height
                    Ellipse()
                        .fill(RadialGradient(colors: [AuthPalette.ribbons[i], .clear], center: .center, startRadius: 0, endRadius: max(w, h) * 0.65))
                        .frame(width: w, height: h)
                        .scaleEffect(drift ? r.scale : 1)
                        .offset(x: field.width * r.origin.x + (drift ? w * r.to.x : 0),
                                y: field.height * r.origin.y + (drift ? h * r.to.y : 0))
                        .animation(reduceMotion ? nil : .timingCurve(0.16, 1, 0.3, 1, duration: r.duration).repeatForever(autoreverses: true), value: drift)
                }
            }
            .frame(width: field.width, height: field.height, alignment: .topLeading)
            .offset(x: -geo.size.width * 0.2, y: -geo.size.height * 0.2)
            .blur(radius: 52)
            .opacity(0.75)
            .blendMode(.screen)
        }
        .allowsHitTesting(false)
        .onAppear { drift = !reduceMotion }
    }
}
