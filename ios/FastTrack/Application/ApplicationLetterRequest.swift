import SwiftUI

struct ApplicationLetterRequest: View {
    let model: ApplicationModel

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if model.letters.isEmpty {
                Button {
                    Task { await model.requestLetter() }
                } label: {
                    Text(model.requestingLetter ? "Requesting…" : "Request cover letter").frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .buttonBorderShape(.roundedRectangle(radius: 12))
                .controlSize(.large)
                .disabled(model.requestingLetter)
            } else {
                Text("Review the letter at the desk.").font(.footnote).foregroundStyle(.secondary)
            }
            if let error = model.letterError {
                Text(error).font(.footnote).foregroundStyle(.red)
            }
        }
        .padding(14)
    }
}
