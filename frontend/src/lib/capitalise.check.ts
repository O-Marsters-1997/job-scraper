import { capitalise, uniqueCapitalised } from "./capitalise";

function eq(a: unknown, b: unknown): void {
	if (JSON.stringify(a) !== JSON.stringify(b))
		throw new Error(
			`expected ${JSON.stringify(a)} to equal ${JSON.stringify(b)}`,
		);
}

eq(capitalise("  acme corp "), "Acme Corp");
eq(capitalise("openAI"), "OpenAI");
eq(capitalise("são paulo"), "São Paulo");
eq(uniqueCapitalised(["acme", "Acme", " ", "united states"]), [
	"Acme",
	"United States",
]);
