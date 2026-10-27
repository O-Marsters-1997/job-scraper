import SwiftUI

struct ApplicationRoleGroup: View {
    let role: DraftRoles.Role
    let model: ApplicationModel
    @State private var showBank = false
    @State private var adding: DraftRoles.Bullet?
    @State private var replacing: DraftRoles.Bullet?

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            if role.isLinked {
                ForEach(role.bullets) { bullet in
                    ApplicationBulletRow(
                        bullet: bullet,
                        isFirst: bullet.id == role.bullets.first?.id,
                        isLast: bullet.id == role.bullets.last?.id,
                        isOnlyOne: role.bullets.count == 1,
                        editable: !model.kept,
                        replace: { replacing = bullet },
                        move: { offset in Task { await model.move(bullet, by: offset, in: role) } },
                        remove: { Task { await model.remove(bullet, from: role) } }
                    )
                    Divider().padding(.leading, 42)
                }
                if !model.kept, !role.bank.isEmpty {
                    bankToggle
                    if showBank {
                        ForEach(role.rankedBank) { bullet in
                            Divider().padding(.leading, 42)
                            ApplicationBankRow(bullet: bullet) { adding = bullet }
                        }
                    }
                }
            } else {
                Text("Not linked to your bank, so it stays as it is on your CV.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 14)
                    .padding(.bottom, 12)
            }
        }
        .sheet(item: $adding) { ApplicationInPlaceOfSheet(role: role, candidate: $0, model: model) }
        .sheet(item: $replacing) { ApplicationSwapInSheet(role: role, old: $0, model: model) }
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

    private var bankToggle: some View {
        Button {
            withAnimation(.snappy(duration: 0.22)) { showBank.toggle() }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "plus")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(Color(.tertiaryLabel))
                    .frame(width: 18)
                Text("From your bank").font(.subheadline).foregroundStyle(.secondary)
                Text(role.bank.count, format: .number).font(.footnote.monospacedDigit()).foregroundStyle(.secondary)
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Color(.tertiaryLabel))
                    .rotationEffect(.degrees(showBank ? 90 : 0))
            }
            .padding(.horizontal, 14)
            .frame(minHeight: 44)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .accessibilityValue(showBank ? "Expanded" : "Collapsed")
    }
}

struct ApplicationBulletMeta: View {
    let bullet: DraftRoles.Bullet
    var showsOrigin = false

    var body: some View {
        HStack(spacing: 6) {
            Text("^[\(bullet.lines) line](inflect: true)").font(.caption.monospacedDigit()).foregroundStyle(.secondary)
            if let percent = bullet.fitPercent, let lean = bullet.lean {
                DotLabel(text: "\(percent)% fit", hue: lean > 0 ? .green : lean <= -0.2 ? .red : .orange)
            }
            if showsOrigin, bullet.onSource {
                Text("was on your CV").font(.caption).foregroundStyle(.secondary)
            }
        }
    }
}

struct ApplicationBulletRow: View {
    let bullet: DraftRoles.Bullet
    let isFirst: Bool
    let isLast: Bool
    let isOnlyOne: Bool
    let editable: Bool
    let replace: () -> Void
    let move: (Int) -> Void
    let remove: () -> Void

    var body: some View {
        if editable {
            Menu {
                Button("Replace with…", systemImage: "arrow.left.arrow.right", action: replace)
                if !isFirst { Button("Move up", systemImage: "arrow.up") { move(-1) } }
                if !isLast { Button("Move down", systemImage: "arrow.down") { move(1) } }
                if !isOnlyOne {
                    Divider()
                    Button("Remove", systemImage: "minus.circle", role: .destructive, action: remove)
                }
            } label: {
                content.frame(minHeight: 44).contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityHint("Shows replace, move and remove actions")
        } else {
            content
        }
    }

    private var content: some View {
        HStack(alignment: .top, spacing: 10) {
            Circle()
                .fill(bullet.onSource ? Color.clear : Color.brandTint)
                .frame(width: 6, height: 6)
                .frame(width: 18, height: 18)
                .accessibilityLabel(bullet.onSource ? "" : "New from your bank")
            VStack(alignment: .leading, spacing: 4) {
                Text(bullet.text).font(.subheadline).foregroundStyle(.primary).multilineTextAlignment(.leading)
                ApplicationBulletMeta(bullet: bullet)
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

struct ApplicationBankRow: View {
    let bullet: DraftRoles.Bullet
    let choose: () -> Void

    var body: some View {
        Button(action: choose) {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: "arrow.left.arrow.right")
                    .font(.caption)
                    .foregroundStyle(Color(.tertiaryLabel))
                    .frame(width: 18, height: 18)
                VStack(alignment: .leading, spacing: 4) {
                    Text(bullet.text).font(.subheadline).foregroundStyle(.secondary).multilineTextAlignment(.leading)
                    ApplicationBulletMeta(bullet: bullet, showsOrigin: true)
                }
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .frame(minHeight: 44)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .accessibilityHint("Choose what it replaces")
    }
}
