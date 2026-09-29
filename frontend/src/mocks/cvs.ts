import { faker } from "@faker-js/faker";
import type { CV } from "@/types/cv";
import { failIfRequested } from "./helpers";
import { seed } from "./seed";

let cvs: CV[] = seed.cvs;

export function getCVs(): CV[] {
	return cvs;
}

export function addTrackedDoc(url: string): void {
	failIfRequested("addTrackedDoc");
	const docId = `doc-${faker.string.uuid().slice(0, 8)}`;
	cvs = [
		...cvs,
		{
			DocID: docId,
			TabID: "t.0",
			Title: "New Tracked Doc",
			SourceDoc: "New Tracked Doc",
			ModifiedAt: new Date().toISOString(),
			DocURL: url,
			Visible: true,
		},
	];
}

export function removeTrackedDoc(docId: string): void {
	cvs = cvs.filter((cv) => cv.DocID !== docId);
}

export function setTabVisibility(
	docId: string,
	tabId: string,
	visible: boolean,
): void {
	cvs = cvs.map((cv) =>
		cv.DocID === docId && cv.TabID === tabId ? { ...cv, Visible: visible } : cv,
	);
}

const MOCK_PDF = `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 58 >>
stream
BT /F1 24 Tf 72 700 Td (Demo CV preview) Tj ET
endstream
endobj
5 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
trailer
<< /Size 6 /Root 1 0 R >>
%%EOF
`;

export function getMockPdfBytes(): ArrayBuffer {
	return new TextEncoder().encode(MOCK_PDF).buffer;
}
