#if DEBUG
import SwiftUI

struct PrototypeLogin: View {
    let onSignIn: () -> Void
    @State private var username = ""
    @State private var password = ""
    @State private var signingIn = false
    @FocusState private var focus: Field?

    private enum Field { case username, password }
    private static let indigo = Color(hex: 0x2a1f57)

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            Spacer()
            VStack(alignment: .leading, spacing: 8) {
                (Text("Fast").foregroundStyle(.white) + Text("Track").foregroundStyle(Color(hex: 0xb7a4ff)))
                    .font(.largeTitle.weight(.bold))
                    .tracking(-0.6)
                Text("Triage new jobs in a minute.")
                    .font(.body)
                    .foregroundStyle(.white.opacity(0.7))
            }
            VStack(spacing: 12) {
                TextField("Username", text: $username, prompt: Text("Username").foregroundStyle(.white.opacity(0.5)))
                    .textContentType(.username)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .focused($focus, equals: .username)
                    .submitLabel(.next)
                    .onSubmit { focus = .password }
                    .modifier(LoginField())
                SecureField("Password", text: $password, prompt: Text("Password").foregroundStyle(.white.opacity(0.5)))
                    .textContentType(.password)
                    .focused($focus, equals: .password)
                    .submitLabel(.go)
                    .onSubmit(signIn)
                    .modifier(LoginField())
            }
            Button(action: signIn) {
                ZStack {
                    Text("Sign in").opacity(signingIn ? 0 : 1)
                    if signingIn { ProgressView().tint(.white) }
                }
                .font(.body.weight(.semibold))
                .foregroundStyle(.white)
                .frame(maxWidth: .infinity, minHeight: 52)
                .background(Color.brandTint, in: .rect(cornerRadius: 14, style: .continuous))
            }
            .disabled(signingIn)
            Spacer()
            Spacer()
        }
        .padding(.horizontal, 28)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Self.indigo.ignoresSafeArea())
        .environment(\.colorScheme, .dark)
    }

    private func signIn() {
        signingIn = true
        Task {
            try? await Task.sleep(for: .milliseconds(700))
            onSignIn()
        }
    }
}

private struct LoginField: ViewModifier {
    func body(content: Content) -> some View {
        content
            .foregroundStyle(.white)
            .padding(.horizontal, 16)
            .frame(minHeight: 52)
            .background(.white.opacity(0.08), in: .rect(cornerRadius: 14, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous).strokeBorder(.white.opacity(0.14)))
    }
}
#endif
