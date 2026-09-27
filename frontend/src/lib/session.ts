export async function signOut(
	logout: () => Promise<unknown>,
	afterLogout: () => void,
): Promise<void> {
	try {
		await logout();
	} finally {
		afterLogout();
	}
}
