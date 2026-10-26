import SwiftUI

struct OfflineRow: View {
    let isOffline: Bool
    let queuedCount: Int
    let syncNote: String?
    let onSync: () -> Void
    let onClearNote: () -> Void

    var body: some View {
        if isOffline || syncNote != nil {
            HStack(spacing: 8) {
                Image(systemName: isOffline ? "wifi.slash" : "checkmark.circle")
                Text(syncNote ?? "Offline · \(queuedCount) queued")
                    .font(.footnote.weight(.semibold))
                Spacer()
                if isOffline {
                    Button("Sync", action: onSync)
                        .font(.footnote.weight(.semibold))
                        .frame(minHeight: 44)
                }
            }
            .foregroundStyle(.secondary)
            .padding(.horizontal, 16)
            .task(id: syncNote) {
                guard syncNote != nil else { return }
                try? await Task.sleep(for: .seconds(4))
                onClearNote()
            }
        }
    }
}
