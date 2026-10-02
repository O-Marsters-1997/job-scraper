export type SaveStatus = "saved" | "saving" | "failed";

export function createSaveLoop(opts: {
	delay: number;
	send: () => Promise<void>;
	onStatus: (status: SaveStatus) => void;
}) {
	let timer: ReturnType<typeof setTimeout> | undefined;
	let inflight: Promise<void> | undefined;
	let again = false;
	let ok = true;

	const start = () => {
		if (inflight) {
			again = true;
			return;
		}
		opts.onStatus("saving");
		inflight = opts.send().then(
			() => finish("saved"),
			() => finish("failed"),
		);
	};
	const finish = (status: SaveStatus) => {
		inflight = undefined;
		ok = status === "saved";
		if (again) {
			again = false;
			start();
			return;
		}
		opts.onStatus(status);
	};
	const cancelTimer = () => {
		clearTimeout(timer);
		timer = undefined;
	};
	const idle = async () => {
		while (inflight) await inflight;
	};

	return {
		schedule() {
			cancelTimer();
			opts.onStatus("saving");
			timer = setTimeout(() => {
				timer = undefined;
				start();
			}, opts.delay);
		},
		retry() {
			cancelTimer();
			start();
		},
		async flush(): Promise<boolean> {
			if (timer !== undefined) {
				cancelTimer();
				start();
			}
			await idle();
			return ok;
		},
		pending: () => timer !== undefined || inflight !== undefined,
	};
}
