enum LoadState<Value> {
    case idle
    case loading
    case loaded(Value)
    case failed(any Error)

    var value: Value? {
        if case .loaded(let value) = self { value } else { nil }
    }
}
