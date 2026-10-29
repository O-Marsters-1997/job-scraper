import SwiftUI

struct SwipeAction: Identifiable {
    let title: String
    let systemImage: String
    var isDestructive = false
    let perform: () -> Void

    var id: String { title }

    static func deleteApplication(perform: @escaping () -> Void) -> SwipeAction {
        SwipeAction(title: "Delete application", systemImage: "trash", isDestructive: true, perform: perform)
    }
}

struct SwipeActions: ViewModifier {
    let actions: [SwipeAction]

    @State private var offset: CGFloat = 0
    @State private var isRevealed = false
    @ScaledMetric(relativeTo: .body) private var buttonSize = 44.0
    private let gap: CGFloat = 8

    private var revealWidth: CGFloat { CGFloat(actions.count) * (buttonSize + gap) + gap }

    func body(content: Content) -> some View {
        content
            .offset(x: offset)
            .background(alignment: .trailing) {
                HStack(spacing: gap) {
                    ForEach(actions) { action in
                        Button(action.title, systemImage: action.systemImage, role: action.isDestructive ? .destructive : nil) {
                            close()
                            action.perform()
                        }
                        .labelStyle(.iconOnly)
                        .font(.body.weight(.semibold))
                        .foregroundStyle(action.isDestructive ? Color.red : Color.secondary)
                        .frame(width: buttonSize, height: buttonSize)
                        .background(Color(.secondarySystemBackground), in: .circle)
                        .overlay(Circle().strokeBorder(Color(.separator)))
                        .contentShape(.circle)
                    }
                }
                .padding(.trailing, gap)
                .opacity(offset < 0 ? 1 : 0)
                .accessibilityHidden(true)
            }
            .simultaneousGesture(drag)
            .accessibilityActions {
                ForEach(actions) { action in
                    Button(action.title, action: action.perform)
                }
            }
    }

    private func close() {
        withAnimation(.snappy(duration: 0.22)) {
            offset = 0
            isRevealed = false
        }
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
    func iconSwipeActions(_ actions: [SwipeAction]) -> some View {
        modifier(SwipeActions(actions: actions))
    }
}
