import tsParser from "@typescript-eslint/parser";
import solid from "eslint-plugin-solid";

export default [
	{ ignores: ["src/routeTree.gen.ts"] },
	{
		files: ["src/**/*.{ts,tsx}"],
		languageOptions: {
			parser: tsParser,
			parserOptions: {
				ecmaFeatures: { jsx: true },
			},
		},
		plugins: { solid },
		rules: {
			"solid/reactivity": "error",
			"solid/no-destructure": "error",
			"solid/components-return-once": "error",
			"solid/prefer-for": "error",
			"solid/no-react-deps": "error",
			"solid/no-react-specific-props": "error",
			"solid/event-handlers": "error",
		},
	},
];
