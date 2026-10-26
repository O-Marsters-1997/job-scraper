import SwiftUI

enum SwipeDirection { case left, right, up }

struct SwipeToDecide: ViewModifier {
    var allowsUp = false
    var onProgress: ((CGFloat) -> Void)?
    let onCommit: (SwipeDirection) -> Void

    @State private var offset: CGSize = .zero
    @State private var armed: SwipeDirection?
    @State private var lockedVertical = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func body(content: Content) -> some View {
        GeometryReader { proxy in
            let threshold = proxy.size.width * 0.35
            content
                .offset(x: offset.width, y: allowsUp ? min(offset.height, 0) : 0)
                .rotationEffect(.degrees(reduceMotion ? 0 : Double(offset.width / proxy.size.width) * 8))
                .simultaneousGesture(drag(threshold: threshold))
                .sensoryFeedback(.selection, trigger: armed) { _, new in new != nil }
        }
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
                onProgress?(value.translation.width / threshold)
            }
            .onEnded { value in
                if lockedVertical {
                    lockedVertical = false
                    return
                }
                if let dir = direction(for: value.translation, predicted: value.predictedEndTranslation, threshold: threshold) {
                    withAnimation(.easeOut(duration: 0.25)) {
                        offset = flyOff(dir)
                    } completion: {
                        onProgress?(0)
                        onCommit(dir)
                    }
                } else {
                    withAnimation(.spring(duration: 0.3, bounce: reduceMotion ? 0 : 0.15)) { offset = .zero }
                    armed = nil
                    onProgress?(0)
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
    func swipeToDecide(
        allowsUp: Bool = false,
        onProgress: ((CGFloat) -> Void)? = nil,
        onCommit: @escaping (SwipeDirection) -> Void
    ) -> some View {
        modifier(SwipeToDecide(allowsUp: allowsUp, onProgress: onProgress, onCommit: onCommit))
    }
}
