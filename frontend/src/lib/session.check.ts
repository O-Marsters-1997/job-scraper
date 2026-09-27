import { signOut } from "./session";

function ok(cond: boolean, msg?: string): void {
	if (!cond) throw new Error(msg ?? "assertion failed");
}

{
	let cleanedUp = false;
	await signOut(
		() => Promise.reject(new Error("network down")),
		() => {
			cleanedUp = true;
		},
	).catch(() => {});
	ok(cleanedUp, "afterLogout must run even when logout() rejects");
}

{
	let cleanedUp = false;
	await signOut(
		() => Promise.resolve(),
		() => {
			cleanedUp = true;
		},
	);
	ok(cleanedUp, "afterLogout must run when logout() resolves");
}

console.log("✓ session checks passed");
