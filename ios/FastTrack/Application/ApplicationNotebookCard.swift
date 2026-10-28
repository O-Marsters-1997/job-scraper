import SwiftUI

struct ApplicationNotebookCard<Content: View>: View {
    let title: String
    let summary: String
    let done: Bool
    let status: (text: String, hue: Color)?
    let isOpen: Bool
    let toggle: () -> Void
    @ViewBuilder let content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Button(action: toggle) { header }
                .buttonStyle(.plain)
                .accessibilityHint(isOpen ? "Collapse" : "Expand")
            if isOpen {
                Divider()
                content()
                    .transition(.opacity.animation(.easeOut(duration: 0.12)))
            }
        }
        .hairlineCard()
    }

    private var header: some View {
        HStack(alignment: .top, spacing: 12) {
            RoundedRectangle(cornerRadius: 5, style: .continuous)
                .strokeBorder(done ? Color.clear : Color(.tertiaryLabel), lineWidth: 1.5)
                .background(done ? Color(.secondaryLabel) : .clear, in: .rect(cornerRadius: 5, style: .continuous))
                .overlay {
                    if done {
                        Image(systemName: "checkmark").font(.caption2.weight(.bold)).foregroundStyle(.white)
                    }
                }
                .frame(width: 18, height: 18)
                .padding(.top, 2)
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 4) {
                Text(title).font(.headline)
                Text(summary).font(.footnote.monospacedDigit()).foregroundStyle(.secondary).lineLimit(1)
            }
            Spacer(minLength: 8)
            if let status { DotLabel(text: status.text, hue: status.hue) }
            Image(systemName: "chevron.right")
                .font(.footnote.weight(.semibold))
                .foregroundStyle(Color(.tertiaryLabel))
                .rotationEffect(.degrees(isOpen ? 90 : 0))
                .padding(.top, 3)
        }
        .padding(14)
        .frame(minHeight: 44)
        .contentShape(.rect)
    }
}

struct ApplicationActionRow: View {
    let title: String
    let systemImage: String
    var action: (() -> Void)?

    var body: some View {
        if let action {
            Button(action: action) { label }.buttonStyle(.plain)
        } else {
            label
        }
    }

    var label: some View {
        HStack(spacing: 10) {
            Image(systemName: systemImage).font(.subheadline).foregroundStyle(.secondary).frame(width: 18)
            Text(title).font(.subheadline)
            Spacer()
        }
        .padding(.horizontal, 14)
        .frame(minHeight: 44)
        .contentShape(.rect)
    }
}
