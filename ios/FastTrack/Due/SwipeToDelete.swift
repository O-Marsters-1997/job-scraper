import SwiftUI

struct SwipeToDelete: ViewModifier {
    let onDelete: () -> Void

    @State private var offset: CGFloat = 0
    @State private var isRevealed = false
    private let revealWidth: CGFloat = 96

    func body(content: Content) -> some View {
        content
            .offset(x: offset)
            .background(alignment: .trailing) {
                Button(role: .destructive) {
                    withAnimation(.snappy(duration: 0.22)) {
                        offset = 0
                        isRevealed = false
                    }
                    onDelete()
                } label: {
                    Label("Delete", systemImage: "trash")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.white)
                        .frame(width: revealWidth)
                        .frame(maxHeight: .infinity)
                }
                .background(.red, in: .rect(cornerRadius: 14, style: .continuous))
                .opacity(offset < 0 ? 1 : 0)
                .accessibilityHidden(true)
            }
            .simultaneousGesture(drag)
            .accessibilityAction(named: "Delete application", onDelete)
    }

    private var drag: some Gesture {
        DragGesture(minimumDistance: 20)
            .onChanged { value in
                guard abs(value.translation.width) > abs(value.translation.height) else { return }
                let start = isRevealed ? -revealWidth : 0
                offset = min(0, max(-revealWidth * 1.3, start + value.translation.width))
            }
            .onEnded { _ in
                withAnimation(.snappy(duration: 0.22)) {
                    isRevealed = offset < -revealWidth / 2
                    offset = isRevealed ? -revealWidth : 0
                }
            }
    }
}

extension View {
    func swipeToDelete(perform onDelete: @escaping () -> Void) -> some View {
        modifier(SwipeToDelete(onDelete: onDelete))
    }
}
