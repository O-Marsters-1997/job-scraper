export async function sharePdf(bytes: ArrayBuffer, filename: string) {
	const file = new File([bytes], filename, { type: "application/pdf" });
	if (!navigator.canShare?.({ files: [file] })) {
		download(file);
		return;
	}
	try {
		await navigator.share({ files: [file] });
	} catch (err) {
		if (!(err instanceof DOMException)) throw err;
		if (err.name === "NotAllowedError") download(file);
		else if (err.name !== "AbortError") throw err;
	}
}

function download(file: File) {
	const url = URL.createObjectURL(file);
	const a = document.createElement("a");
	a.href = url;
	a.download = file.name;
	a.click();
	setTimeout(() => URL.revokeObjectURL(url), 0);
}
