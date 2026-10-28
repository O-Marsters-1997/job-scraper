import SwiftUI

extension View {
    func clearingSearchable(text: Binding<String>) -> some View {
        modifier(ClearingSearch(text: text))
    }
}

private struct ClearingSearch: ViewModifier {
    @Binding var text: String
    @State private var isPresented = false
    @State private var submitted = false
    @FocusState private var focused: Bool

    func body(content: Content) -> some View {
        if #available(iOS 18, *) {
            content
                .searchable(text: $text, isPresented: $isPresented)
                .searchFocused($focused)
                .onSubmit(of: .search) { submitted = true }
                .onChange(of: focused) { _, isFocused in
                    if isFocused {
                        submitted = false
                    } else if !submitted {
                        text = ""
                        isPresented = false
                    }
                }
        } else {
            content.searchable(text: $text)
        }
    }
}
