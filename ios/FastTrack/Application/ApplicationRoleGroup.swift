import SwiftUI

struct ApplicationRoleGroup: View {
    let role: DraftRoles.Role

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            if role.isLinked {
                ForEach(role.bullets) { bullet in
                    ApplicationBulletRow(bullet: bullet)
                    Divider().padding(.leading, 42)
                }
            } else {
                Text("Not linked to your bank, so it stays as it is on your CV.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 14)
                    .padding(.bottom, 12)
            }
        }
    }

    private var header: some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(role.employer).font(.system(.body, design: .serif))
            Text(role.title).font(.footnote).foregroundStyle(.secondary).lineLimit(1)
            Spacer(minLength: 8)
            if role.isLinked {
                if role.over > 0 {
                    DotLabel(text: "\(role.over) over", hue: .red)
                } else {
                    Text("\(role.used)/\(role.budget) lines").font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                }
            }
        }
        .padding(.horizontal, 14)
        .padding(.top, 16)
        .padding(.bottom, 6)
        .accessibilityElement(children: .combine)
    }
}

struct ApplicationBulletRow: View {
    let bullet: DraftRoles.Bullet

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Circle()
                .fill(bullet.onSource ? Color.clear : Color.brandTint)
                .frame(width: 6, height: 6)
                .frame(width: 18, height: 18)
                .accessibilityLabel(bullet.onSource ? "" : "New from your bank")
            VStack(alignment: .leading, spacing: 4) {
                Text(bullet.text).font(.subheadline).multilineTextAlignment(.leading)
                HStack(spacing: 6) {
                    Text("^[\(bullet.lines) line](inflect: true)").font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                    if let percent = bullet.fitPercent, let lean = bullet.lean {
                        DotLabel(text: "\(percent)% fit", hue: lean > 0 ? .green : lean <= -0.2 ? .red : .orange)
                    }
                }
                if let replaced = bullet.replacedFrom {
                    Text("In place of \u{201C}\(replaced)\u{201D}").font(.caption).foregroundStyle(.secondary).lineLimit(1)
                }
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
    }
}
