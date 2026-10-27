import SwiftUI

struct CompanyKindBadge: View {
    let kind: CompanyKind

    var body: some View {
        if let badge = kind.badge {
            Text(badge)
                .font(.caption.weight(.medium))
                .foregroundStyle(.secondary)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(.quaternary, in: .rect(cornerRadius: 5, style: .continuous))
        }
    }
}
