import { z } from "zod";

function ok(condition: boolean, msg: string) {
	if (!condition) throw new Error(`FAIL: ${msg}`);
}

// schema.parse succeeds on valid data
const schema = z.object({ id: z.string(), score: z.number() });
const result = schema.parse({ id: "abc", score: 42 });
ok(result.id === "abc" && result.score === 42, "parse returns typed data");

// schema.parse throws on invalid data (trust boundary rejects bad shapes)
let threw = false;
try {
	schema.parse({ id: 123 });
} catch {
	threw = true;
}
ok(threw, "parse throws on invalid shape");

// schema.parse strips extra fields (no unexpected data leaks through)
const strict = z.object({ name: z.string() }).strict();
let strictThrew = false;
try {
	strict.parse({ name: "ok", extra: "bad" });
} catch {
	strictThrew = true;
}
ok(strictThrew, "strict schema rejects extra fields");
