import Foundation

enum JobDescription {
    static func attributed(html: String) -> AttributedString {
        var builder = Builder()
        var rest = html[...]
        while let open = rest.firstIndex(of: "<") {
            builder.text(String(rest[..<open]))
            if rest[open...].hasPrefix("<!--") {
                rest = rest[open...].range(of: "-->").map { rest[$0.upperBound...] } ?? ""
                continue
            }
            guard let close = rest[open...].firstIndex(of: ">") else {
                rest = rest[open...]
                break
            }
            let tag = Tag(String(rest[rest.index(after: open)..<close]))
            rest = rest[rest.index(after: close)...]
            if !tag.isClosing, skipped.contains(tag.name) {
                rest = skip(past: tag.name, in: rest)
            } else {
                builder.apply(tag)
            }
        }
        builder.text(String(rest))
        return builder.result
    }

    private static let skipped: Set<String> = ["script", "style"]

    private static func skip(past name: String, in rest: Substring) -> Substring {
        guard let end = rest.range(of: "</\(name)", options: .caseInsensitive),
              let close = rest[end.upperBound...].firstIndex(of: ">")
        else { return "" }
        return rest[rest.index(after: close)...]
    }

    private struct Tag {
        let name: String
        let isClosing: Bool
        let href: URL?

        init(_ raw: String) {
            let body = raw.trimmingCharacters(in: .whitespacesAndNewlines)
            isClosing = body.hasPrefix("/")
            name = String(body.drop { $0 == "/" }.prefix { !$0.isWhitespace && $0 != "/" }).lowercased()
            let value = body.firstMatch(of: /(?:^|\s)href\s*=\s*(?:"([^"]*)"|'([^']*)')/.ignoresCase())
                .map { String($0.1 ?? $0.2 ?? "") }
            href = value.flatMap { URL(string: decodeEntities($0)) }
                .flatMap { ["http", "https"].contains($0.scheme?.lowercased()) ? $0 : nil }
        }
    }

    private struct Builder {
        var result = AttributedString()
        private var pendingBreaks = 0
        private var afterMarker = false
        private var strong = 0
        private var emphasis = 0
        private var links: [URL?] = []
        private var lists: [Int?] = []

        mutating func text(_ raw: String) {
            let collapsed = decodeEntities(raw).replacing(/\s+/, with: " ")
            let atLineStart = result.characters.isEmpty || pendingBreaks > 0 || afterMarker
            let trimmed = atLineStart ? String(collapsed.drop { $0 == " " }) : collapsed
            guard !trimmed.isEmpty else { return }
            emit(trimmed)
        }

        mutating func apply(_ tag: Tag) {
            let delta = tag.isClosing ? -1 : 1
            switch tag.name {
            case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
                if !afterMarker { pendingBreaks = 2 }
                if tag.name.hasPrefix("h") { strong = max(strong + delta, 0) }
            case "br":
                pendingBreaks = max(pendingBreaks, 1)
            case "strong", "b":
                strong = max(strong + delta, 0)
            case "em", "i":
                emphasis = max(emphasis + delta, 0)
            case "ul", "ol":
                pendingBreaks = 2
                if tag.isClosing { _ = lists.popLast() } else { lists.append(tag.name == "ol" ? 0 : nil) }
            case "li":
                guard !tag.isClosing else { return }
                pendingBreaks = 1
                var marker = "• "
                if let number = lists.last ?? nil {
                    lists[lists.count - 1] = number + 1
                    marker = "\(number + 1). "
                }
                emit(marker)
                afterMarker = true
            case "a":
                if tag.isClosing { _ = links.popLast() } else { links.append(tag.href) }
            default:
                break
            }
        }

        private mutating func emit(_ string: String) {
            if !result.characters.isEmpty, pendingBreaks > 0 {
                result += AttributedString(String(repeating: "\n", count: pendingBreaks))
            }
            pendingBreaks = 0
            afterMarker = false
            var piece = AttributedString(string)
            var intent = InlinePresentationIntent()
            if strong > 0 { intent.insert(.stronglyEmphasized) }
            if emphasis > 0 { intent.insert(.emphasized) }
            if !intent.isEmpty { piece.inlinePresentationIntent = intent }
            if let url = links.last ?? nil { piece.link = url }
            result += piece
        }
    }

    private static let namedEntities = [
        "amp": "&", "lt": "<", "gt": ">", "quot": "\"", "apos": "'", "nbsp": " ",
        "rsquo": "’", "lsquo": "‘", "rdquo": "”", "ldquo": "“", "ndash": "–", "mdash": "—",
        "hellip": "…", "bull": "•", "pound": "£", "euro": "€", "copy": "©",
    ]

    private static func decodeEntities(_ string: String) -> String {
        string.replacing(/&(#[xX][0-9a-fA-F]+|#[0-9]+|[a-zA-Z]+);/) { match in
            let name = String(match.1)
            if let value = namedEntities[name] { return value }
            let code = name.hasPrefix("#x") || name.hasPrefix("#X")
                ? UInt32(name.dropFirst(2), radix: 16) : UInt32(name.dropFirst())
            return code.flatMap { $0 >= 32 ? Unicode.Scalar($0) : nil }.map { String(Character($0)) } ?? String(match.0)
        }
    }
}
