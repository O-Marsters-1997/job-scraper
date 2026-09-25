# HTML parsing

Once you've copied `internal/sources/wis/wis.go` for an HTML source, load the
`goquery-parsing` skill before writing any selectors or `Find`/`Attr`/`Each`
calls — it covers stable selector strategy, typed structs, and parsers that
survive minor markup changes. Don't duplicate that guidance here; load it at
the point you're about to write the parser, not after.
