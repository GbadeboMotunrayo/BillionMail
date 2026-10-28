/**
 * @description Decode an uploaded text file whatever encoding it was saved in.
 * Excel saves CSV in the system code page (GBK on Chinese Windows) rather than UTF-8,
 * and readAsText assumed UTF-8, which turned names and attributes into garbage (#212, #153).
 */
export function decodeTextBuffer(buffer: ArrayBuffer): string {
	const bytes = new Uint8Array(buffer)
	let text: string
	try {
		text = new TextDecoder('utf-8', { fatal: true }).decode(bytes)
	} catch {
		try {
			text = new TextDecoder('gb18030').decode(bytes)
		} catch {
			text = new TextDecoder('windows-1252').decode(bytes)
		}
	}
	// drop a byte-order mark so the first CSV header is matched
	return text.replace(/^\uFEFF/, '')
}

export async function readTextFile(file: Blob): Promise<string> {
	return decodeTextBuffer(await file.arrayBuffer())
}
