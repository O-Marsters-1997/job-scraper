import SwiftUI

struct LoginView: View {
    @Environment(Session.self) private var session
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var username = ""
    @State private var password = ""
    @State private var error: String?
    @State private var isSubmitting = false
    @State private var risen = false
    @FocusState private var focus: Field?

    private enum Field { case username, password }

    private var canSubmit: Bool { !isSubmitting && !username.isEmpty && !password.isEmpty }

    var body: some View {
        GeometryReader { geo in
            let lift = max(0, geo.safeAreaInsets.top - 24)
            VStack(spacing: 0) {
                AuthBrandPanel(topInset: max(24, geo.safeAreaInsets.top), cloudTop: 72 + lift)
                    .frame(height: 254 + lift)
                ScrollView {
                    form
                        .padding(.horizontal, 22)
                        .padding(.top, 28)
                }
                .scrollBounceBehavior(.basedOnSize)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color(.systemBackground), in: UnevenRoundedRectangle(topLeadingRadius: 22, topTrailingRadius: 22, style: .continuous))
                .padding(.top, -20)
                .opacity(risen ? 1 : 0)
                .offset(y: risen ? 0 : 14)
            }
            .ignoresSafeArea(edges: .top)
        }
        .background(Color(.systemBackground))
        .onAppear {
            withAnimation(reduceMotion ? nil : .timingCurve(0.16, 1, 0.3, 1, duration: 0.7)) { risen = true }
        }
    }

    private var form: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Back on track")
                .font(.title.weight(.bold))
                .tracking(-0.7)
            Text("Sign in to pick up where you left off.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .padding(.top, 4)

            VStack(alignment: .leading, spacing: 16) {
                field("Username") {
                    TextField("Username", text: $username, prompt: Text("alice"))
                        .textContentType(.username)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .focused($focus, equals: .username)
                        .submitLabel(.next)
                        .onSubmit { focus = .password }
                }
                field("Password") {
                    SecureField("Password", text: $password, prompt: Text("••••••••"))
                        .textContentType(.password)
                        .focused($focus, equals: .password)
                        .submitLabel(.go)
                        .onSubmit(submit)
                }
                if let error {
                    Text(error)
                        .font(.subheadline)
                        .foregroundStyle(.red)
                }
                Button(action: submit) {
                    ZStack {
                        HStack(spacing: 8) {
                            Text("Sign in")
                            Image(systemName: "arrow.right").font(.subheadline.weight(.bold))
                        }
                        .opacity(isSubmitting ? 0 : 1)
                        if isSubmitting { ProgressView().tint(.white) }
                    }
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity, minHeight: 48)
                    .background(Color.brandTint, in: .rect(cornerRadius: 12, style: .continuous))
                    .opacity(canSubmit || isSubmitting ? 1 : 0.5)
                }
                .buttonStyle(.plain)
                .disabled(!canSubmit)
                .padding(.top, 8)
            }
            .padding(.top, 24)
        }
    }

    private func field(_ label: String, @ViewBuilder input: () -> some View) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label.uppercased())
                .font(.caption.weight(.semibold))
                .tracking(0.6)
                .foregroundStyle(.secondary)
                .accessibilityHidden(true)
            input()
                .padding(.horizontal, 14)
                .frame(minHeight: 44)
                .overlay(RoundedRectangle(cornerRadius: 12, style: .continuous).strokeBorder(Color(.separator)))
        }
    }

    private func submit() {
        guard canSubmit else { return }
        focus = nil
        Task { await logIn() }
    }

    private func logIn() async {
        isSubmitting = true
        error = nil
        defer { isSubmitting = false }
        do {
            try await session.logIn(username: username, password: password)
        } catch APIError.unauthorized {
            error = "Wrong username or password."
        } catch {
            self.error = error.localizedDescription
        }
    }
}

#Preview {
    LoginView().environment(Session(client: .mock()))
}
