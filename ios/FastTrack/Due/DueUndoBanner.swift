import SwiftUI

struct DueUndoBanner: View {
    let entry: DueUndo?
    let onUndo: () -> Void
    let onExpire: (UUID) -> Void

    var body: some View {
        if let entry {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(entry.title).font(.footnote.weight(.semibold))
                    Text(entry.jobTitle).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button("Undo", action: onUndo)
                    .font(.footnote.weight(.semibold))
                    .frame(minHeight: 44)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 2)
            .hairlineCard()
            .padding(.horizontal)
            .transition(.move(edge: .bottom).combined(with: .opacity))
            .task(id: entry.id) {
                try? await Task.sleep(for: .seconds(4))
                withAnimation { onExpire(entry.id) }
            }
        }
    }
}
