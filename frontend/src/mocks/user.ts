import { clearApplications } from "./applications";

export const mockUser = { id: "user-1", username: "demo" };

export function switchMockUser(username: string): void {
	if (username === mockUser.username) return;
	mockUser.id = `user-${username}`;
	mockUser.username = username;
	clearApplications();
}
