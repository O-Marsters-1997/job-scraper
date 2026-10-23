import assert from "node:assert/strict";
import { htmlToText } from "./htmlToText";

assert.equal(
	htmlToText(
		"&lt;p&gt;Hi &amp;amp; bye&lt;/p&gt;&lt;ul&gt;&lt;li&gt;One&lt;/li&gt;&lt;li&gt;Two&lt;/li&gt;&lt;/ul&gt;",
	),
	"Hi & bye\n\n• One\n• Two",
);
assert.equal(htmlToText("plain\ntext"), "plain\ntext");
assert.equal(htmlToText("<p>a &ndash; b &#99999999;</p>"), "a – b &#99999999;");
