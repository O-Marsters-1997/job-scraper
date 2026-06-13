import type { Locator, Page } from "@playwright/test";

export class AppShellComponent {
  readonly overviewLink: Locator;
  readonly jobsLink: Locator;
  readonly applicationsLink: Locator;
  readonly statusesLink: Locator;
  readonly signOutButton: Locator;

  constructor(
    private readonly page: Page,
    readonly root: Locator,
  ) {
    this.overviewLink = root.getByTitle("Overview");
    this.jobsLink = root.getByTitle("Jobs");
    this.applicationsLink = root.getByTitle("Applications");
    this.statusesLink = root.getByTitle("Status settings");
    this.signOutButton = root.getByTitle("Sign out");
  }
  async navigateTo(link: Locator) {
    await link.click();
  }

  async signOut() {
    await this.signOutButton.click();
    await this.page.waitForURL(/\/login/);
  }
}
