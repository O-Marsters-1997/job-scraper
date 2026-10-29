import { download } from "./sharePdf";

type Mode = { mode: "readwrite" };

interface FolderHandle extends FileSystemDirectoryHandle {
	queryPermission(mode: Mode): Promise<PermissionState>;
	requestPermission(mode: Mode): Promise<PermissionState>;
}

type PickerWindow = Window & {
	showDirectoryPicker?: (options: Mode) => Promise<FolderHandle>;
};

const DB_NAME = "cv-downloads";
const STORE = "handles";
const KEY = "folder";
const READWRITE: Mode = { mode: "readwrite" };

export const folderPickerSupported = () =>
	typeof window !== "undefined" &&
	typeof (window as PickerWindow).showDirectoryPicker === "function" &&
	typeof indexedDB !== "undefined";

function openDb(): Promise<IDBDatabase> {
	return new Promise((resolve, reject) => {
		const req = indexedDB.open(DB_NAME, 1);
		req.onupgradeneeded = () => req.result.createObjectStore(STORE);
		req.onsuccess = () => resolve(req.result);
		req.onerror = () => reject(req.error);
	});
}

async function withStore<T>(
	mode: IDBTransactionMode,
	run: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
	const db = await openDb();
	try {
		return await new Promise<T>((resolve, reject) => {
			const req = run(db.transaction(STORE, mode).objectStore(STORE));
			req.onsuccess = () => resolve(req.result);
			req.onerror = () => reject(req.error);
		});
	} finally {
		db.close();
	}
}

export async function getDownloadFolder(): Promise<FolderHandle | undefined> {
	if (!folderPickerSupported()) return undefined;
	return withStore<FolderHandle | undefined>("readonly", (s) =>
		s.get(KEY),
	).catch(() => undefined);
}

export async function chooseDownloadFolder(): Promise<
	FolderHandle | undefined
> {
	try {
		const handle = await (window as PickerWindow).showDirectoryPicker?.(
			READWRITE,
		);
		if (!handle) return undefined;
		await withStore("readwrite", (s) => s.put(handle, KEY));
		return handle;
	} catch (err) {
		if (err instanceof DOMException && err.name === "AbortError")
			return undefined;
		throw err;
	}
}

export async function clearDownloadFolder(): Promise<void> {
	await withStore("readwrite", (s) => s.delete(KEY));
}

export async function writableDownloadFolder(): Promise<
	FolderHandle | undefined
> {
	const folder = await getDownloadFolder();
	if (!folder) return undefined;
	const granted = await folder
		.queryPermission(READWRITE)
		.then((state) =>
			state === "granted" ? state : folder.requestPermission(READWRITE),
		)
		.catch(() => "denied");
	return granted === "granted" ? folder : undefined;
}

async function writeToFolder(
	folder: FolderHandle,
	file: File,
): Promise<boolean> {
	try {
		const handle = await folder.getFileHandle(file.name, { create: true });
		const out = await handle.createWritable();
		await out.write(file);
		await out.close();
		return true;
	} catch {
		return false;
	}
}

export async function savePdf(
	bytes: ArrayBuffer,
	filename: string,
	folder: FolderHandle | undefined,
): Promise<void> {
	const file = new File([bytes], filename, { type: "application/pdf" });
	if (folder && (await writeToFolder(folder, file))) return;
	download(file);
}
