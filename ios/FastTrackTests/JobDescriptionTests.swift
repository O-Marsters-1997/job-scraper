import Foundation
import Testing
@testable import FastTrack

struct JobDescriptionTests {
    private func plain(_ html: String) -> String {
        String(JobDescription.attributed(html: html).characters)
    }

    @Test func keepsHeadingsAsStrongText() throws {
        let result = JobDescription.attributed(html: "<h2>About</h2><p>Build things.</p>")
        let heading = try #require(result.runs.first)
        #expect(heading.inlinePresentationIntent == .stronglyEmphasized)
        #expect(String(result.characters) == "About\n\nBuild things.")
    }

    @Test func keepsListItemsAsBullets() {
        #expect(plain("<ul><li>Go</li><li>Postgres</li></ul>") == "• Go\n• Postgres")
        #expect(plain("<ol><li>Apply</li><li>Interview</li></ol>") == "1. Apply\n2. Interview")
    }

    @Test(arguments: ["https://example.com/apply", "http://example.com"])
    func keepsWebLinks(href: String) throws {
        let result = JobDescription.attributed(html: "<a href=\"\(href)\">Apply</a>")
        #expect(result.runs.first?.link == URL(string: href))
    }

    @Test(arguments: ["javascript:alert(1)", "data:text/html,x", "mailto:a@b.co", "/relative"])
    func dropsNonWebLinks(href: String) {
        let result = JobDescription.attributed(html: "<a href=\"\(href)\">Apply</a>")
        #expect(String(result.characters) == "Apply")
        #expect(result.runs.allSatisfy { $0.link == nil })
    }

    @Test func dropsScriptStyleAndImages() {
        let html = "<style>p{color:red}</style>Hi<script>alert('x')</script><img src=\"https://e.com/a.png\" alt=\"pic\"> there"
        #expect(plain(html) == "Hi there")
    }

    @Test func listItemsWrappingParagraphsStayOnOneLine() {
        #expect(plain("<ul><li><p>Go</p></li><li> Rust</li></ul>") == "• Go\n• Rust")
    }

    @Test func dropsCommentsContainingTagEnds() {
        #expect(plain("A<!-- a > b -->B") == "AB")
    }

    @Test func decodesEntities() {
        #expect(plain("it&rsquo;s &pound;5") == "it’s £5")
        #expect(plain("R&amp;D &lt;b&gt; &#169; &#x41;") == "R&D <b> © A")
    }
}
