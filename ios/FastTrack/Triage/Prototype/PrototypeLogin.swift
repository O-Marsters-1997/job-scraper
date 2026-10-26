#if DEBUG
import SwiftUI

struct PrototypeLogin: View {
    let onSignIn: () -> Void
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var username = ""
    @State private var password = ""
    @State private var signingIn = false
    @State private var risen = false
    @FocusState private var focus: Field?

    private enum Field { case username, password }

    var body: some View {
        GeometryReader { geo in
            let lift = max(0, geo.safeAreaInsets.top - 24)
            VStack(spacing: 0) {
                AuthBrandPanel(topInset: max(24, geo.safeAreaInsets.top), cloudTop: 72 + lift)
                    .frame(height: 254 + lift)
                form
                    .padding(.horizontal, 22)
                    .padding(.top, 28)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
                    .background(Color(.systemBackground), in: UnevenRoundedRectangle(topLeadingRadius: 22, topTrailingRadius: 22, style: .continuous))
                    .padding(.top, -20)
                    .opacity(risen ? 1 : 0)
                    .offset(y: risen ? 0 : 14)
            }
            .ignoresSafeArea(edges: .top)
        }
        .background(Color(.systemBackground))
        .onAppear {
            withAnimation(reduceMotion ? nil : .timingCurve(0.16, 1, 0.3, 1, duration: 0.7)) { risen = true }
        }
    }

    private var form: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Back on track")
                .font(.title.weight(.bold))
                .tracking(-0.7)
            Text("Sign in to pick up where you left off.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .padding(.top, 4)

            VStack(alignment: .leading, spacing: 16) {
                field("Username") {
                    TextField("Username", text: $username, prompt: Text("alice"))
                        .textContentType(.username)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .focused($focus, equals: .username)
                        .submitLabel(.next)
                        .onSubmit { focus = .password }
                }
                field("Password") {
                    SecureField("Password", text: $password, prompt: Text("••••••••"))
                        .textContentType(.password)
                        .focused($focus, equals: .password)
                        .submitLabel(.go)
                        .onSubmit(signIn)
                }
                Button(action: signIn) {
                    ZStack {
                        HStack(spacing: 8) {
                            Text("Sign in")
                            Image(systemName: "arrow.right").font(.subheadline.weight(.bold))
                        }
                        .opacity(signingIn ? 0 : 1)
                        if signingIn { ProgressView().tint(.white) }
                    }
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity, minHeight: 48)
                    .background(Color.brandTint, in: .rect(cornerRadius: 12, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(signingIn)
                .padding(.top, 8)
            }
            .padding(.top, 24)
        }
    }

    private func field(_ label: String, @ViewBuilder input: () -> some View) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label.uppercased())
                .font(.caption.weight(.semibold))
                .tracking(0.6)
                .foregroundStyle(.secondary)
                .accessibilityHidden(true)
            input()
                .padding(.horizontal, 14)
                .frame(minHeight: 44)
                .overlay(RoundedRectangle(cornerRadius: 12, style: .continuous).strokeBorder(Color(.separator)))
        }
    }

    private func signIn() {
        focus = nil
        signingIn = true
        Task {
            try? await Task.sleep(for: .milliseconds(700))
            onSignIn()
        }
    }
}

private enum AuthPalette {
    static let p1 = Color(hex: 0x2b2a46)
    static let p2 = Color(hex: 0x0c0a1d)
    static let bright = Color(hex: 0xa19ee9)
    static let ink = Color(hex: 0xf5f4fc)
    static let ribbons = [Color(hex: 0x7c73dc), Color(hex: 0x6d5ed5), Color(hex: 0x8d87e2)]
}

private struct AuthBrandPanel: View {
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
        .onAppear { drift = true }
    }
}

private struct CompactRoleCloud: View {
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
            ForEach(pills.indices, id: \.self) { i in
                let p = pills[i]
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
        .onAppear { drift = true }
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

private struct Wordmark: View {
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
#endif
