#if DEBUG
import SwiftUI

enum SwipeDirection { case left, right, up }

struct SwipeToDecide: ViewModifier {
    var allowsUp = false
    var isEnabled = true
    let onCommit: (SwipeDirection) -> Void

    @State private var offset: CGSize = .zero
    @State private var armed: SwipeDirection?
    @State private var lockedVertical = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func body(content: Content) -> some View {
        GeometryReader { proxy in
            let threshold = proxy.size.width * 0.35
            content
                .overlay(alignment: .top) { stamp }
                .offset(x: offset.width, y: allowsUp ? min(offset.height, 0) : 0)
                .rotationEffect(.degrees(reduceMotion ? 0 : Double(offset.width / proxy.size.width) * 8))
                .simultaneousGesture(drag(threshold: threshold), including: isEnabled ? .all : .subviews)
                .sensoryFeedback(.selection, trigger: armed) { _, new in new != nil }
        }
    }

    private var stamp: some View {
        let isRight = offset.width > 0
        let progress = min(abs(offset.width) / 120, 1)
        return Label(isRight ? "Shortlist" : "Dismiss", systemImage: isRight ? "checkmark" : "xmark")
            .font(.headline)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .foregroundStyle(.white)
            .background(isRight ? Color.brandTint : Color(.systemGray), in: .capsule)
            .opacity(armed == .left || armed == .right ? 1 : progress * 0.6)
            .padding(.top, 20)
            .frame(maxWidth: .infinity, alignment: isRight ? .leading : .trailing)
            .padding(.horizontal, 20)
            .opacity(offset.width == 0 ? 0 : 1)
            .allowsHitTesting(false)
    }

    private func drag(threshold: CGFloat) -> some Gesture {
        DragGesture(minimumDistance: 10)
            .onChanged { value in
                if lockedVertical { return }
                if !allowsUp, offset == .zero, abs(value.translation.height) > abs(value.translation.width) {
                    lockedVertical = true
                    return
                }
                offset = value.translation
                armed = direction(for: value.translation, predicted: value.translation, threshold: threshold)
            }
            .onEnded { value in
                if lockedVertical {
                    lockedVertical = false
                    return
                }
                if let dir = direction(for: value.translation, predicted: value.predictedEndTranslation, threshold: threshold) {
                    withAnimation(.easeOut(duration: 0.25)) {
                        offset = flyOff(dir)
                    }
                    Task {
                        try? await Task.sleep(for: .milliseconds(200))
                        onCommit(dir)
                        offset = .zero
                        armed = nil
                    }
                } else {
                    withAnimation(.spring(duration: 0.3, bounce: reduceMotion ? 0 : 0.15)) { offset = .zero }
                    armed = nil
                }
            }
    }

    private func direction(for t: CGSize, predicted: CGSize, threshold: CGFloat) -> SwipeDirection? {
        if allowsUp, -t.height > threshold || -predicted.height > threshold * 2.5, abs(t.height) > abs(t.width) { return .up }
        guard abs(t.width) > abs(t.height) else { return nil }
        if t.width > threshold || predicted.width > threshold * 2.5 { return .right }
        if t.width < -threshold || predicted.width < -threshold * 2.5 { return .left }
        return nil
    }

    private func flyOff(_ dir: SwipeDirection) -> CGSize {
        if reduceMotion { return offset }
        switch dir {
        case .right: return CGSize(width: 600, height: offset.height)
        case .left: return CGSize(width: -600, height: offset.height)
        case .up: return CGSize(width: offset.width, height: -900)
        }
    }
}

extension View {
    func swipeToDecide(allowsUp: Bool = false, isEnabled: Bool = true, onCommit: @escaping (SwipeDirection) -> Void) -> some View {
        modifier(SwipeToDecide(allowsUp: allowsUp, isEnabled: isEnabled, onCommit: onCommit))
    }
}
#endif
