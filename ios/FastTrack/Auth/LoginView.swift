import SwiftUI

struct LoginView: View {
    @Environment(Session.self) private var session
    @State private var username = ""
    @State private var password = ""
    @State private var error: String?
    @State private var isSubmitting = false

    var body: some View {
        Form {
            Section("FastTrack") {
                TextField("Username", text: $username)
                    .textContentType(.username)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                SecureField("Password", text: $password)
                    .textContentType(.password)
            }
            if let error {
                Text(error).foregroundStyle(.red)
            }
            Button("Log in", action: submit)
                .disabled(isSubmitting || username.isEmpty || password.isEmpty)
        }
    }

    private func submit() {
        Task { await logIn() }
    }

    private func logIn() async {
        isSubmitting = true
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
