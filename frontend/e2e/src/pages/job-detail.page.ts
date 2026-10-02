import type { Locator, Page } from "@playwright/test";

export class JobDetailPage {
	readonly actionBar: Locator;
	readonly applied: Locator;
	readonly dialog: Locator;

	constructor(private readonly page: Page) {
		this.actionBar = page.locator("div.fixed.bottom-0");
		this.applied = this.actionBar.getByRole("button", { name: "I applied" });
		this.dialog = page.getByRole("dialog");
	}

	async goto(jobId: string) {
		await this.page.goto(`/jobs/${jobId}`);
		await this.page.getByRole("heading", { level: 1 }).waitFor();
	}

	// The dev-only router devtools button floats over the bar's right edge.
	async tapApplied() {
		await this.applied.dispatchEvent("click");
	}

	async hasHorizontalScroll() {
		return this.page.evaluate(
			() =>
				document.documentElement.scrollWidth >
				document.documentElement.clientWidth,
		);
	}
}
