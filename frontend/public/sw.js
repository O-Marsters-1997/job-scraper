self.addEventListener("push", (event) => {
	const msg = event.data ? event.data.json() : {};
	event.waitUntil(
		self.registration.showNotification(msg.title || "FastTrack", {
			body: msg.body,
			tag: msg.tag || undefined,
			data: { url: msg.url || "/" },
			icon: "/logo192.png",
		}),
	);
});

self.addEventListener("notificationclick", (event) => {
	event.notification.close();
	const url = new URL(event.notification.data.url, self.location.origin).href;
	event.waitUntil(
		self.clients
			.matchAll({ type: "window", includeUncontrolled: true })
			.then((clients) => {
				const open = clients.find((c) =>
					c.url.startsWith(self.location.origin),
				);
				if (!open) return self.clients.openWindow(url);
				return open
					.focus()
					.then(() => open.navigate(url))
					.catch(() => self.clients.openWindow(url));
			}),
	);
});
