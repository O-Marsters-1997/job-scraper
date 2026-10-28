enum LoadState<Value> {
    case idle
    case loading
    case loaded(Value)
    case failed(any Error)

    var isIdle: Bool {
        if case .idle = self { true } else { false }
    }

    var value: Value? {
        if case .loaded(let value) = self { value } else { nil }
    }
}
