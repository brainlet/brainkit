// Test: MDocument.fromPDF — parses a tiny PDF (Hello from brainkit)
// via pdfjs-dist legacy with workers disabled + DOMMatrix/Path2D
// stubs. Asserts text-content extraction.
import { MDocument } from "agent";
import { output } from "kit";

// Minimal PDF 1.4 containing "Hello from brainkit" in a single
// Helvetica text run. 417 bytes raw.
const pdfB64 =
  "JVBERi0xLjQKJeLjz9MKMSAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZyAvUGFn" +
  "ZXMgMiAwIFIgPj4KZW5kb2JqCjIgMCBvYmoKPDwgL1R5cGUgL1BhZ2VzIC9L" +
  "aWRzIFszIDAgUl0gL0NvdW50IDEgPj4KZW5kb2JqCjMgMCBvYmoKPDwgL1R5" +
  "cGUgL1BhZ2UgL1BhcmVudCAyIDAgUiAvTWVkaWFCb3ggWzAgMCA2MTIgNzky" +
  "XSAvUmVzb3VyY2VzIDw8IC9Gb250IDw8IC9GMSA0IDAgUiA+PiA+PiAvQ29u" +
  "dGVudHMgNSAwIFIgPj4KZW5kb2JqCjQgMCBvYmoKPDwgL1R5cGUgL0ZvbnQg" +
  "L1N1YnR5cGUgL1R5cGUxIC9CYXNlRm9udCAvSGVsdmV0aWNhID4+CmVuZG9i" +
  "ago1IDAgb2JqCjw8IC9MZW5ndGggNDQgPj4Kc3RyZWFtCkJUIC9GMSAyNCBU" +
  "ZiAxMDAgNzAwIFRkIChIZWxsbyBmcm9tIGJyYWlua2l0KSBUaiBFVAplbmRz" +
  "dHJlYW0KZW5kb2JqCnhyZWYKMCA2CjAwMDAwMDAwMDAgNjU1MzUgZiAKMDAw" +
  "MDAwMDAxNSAwMDAwMCBuIAowMDAwMDAwMDY0IDAwMDAwIG4gCjAwMDAwMDAx" +
  "MTUgMDAwMDAgbiAKMDAwMDAwMDIyMiAwMDAwMCBuIAowMDAwMDAwMjgxIDAw" +
  "MDAwIG4gCnRyYWlsZXIgPDwgL1NpemUgNiAvUm9vdCAxIDAgUiA+PgpzdGFy" +
  "dHhyZWYKMzczCiUlRU9GCg==";

let text = "";
let errorMsg = "";
try {
  const doc: any = await (MDocument as any).fromPDF(pdfB64);
  const out = doc.getText();
  text = Array.isArray(out) ? out.join(" ") : String(out || "");
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({
  extractedText: text.length > 0,
  containsHello: text.toLowerCase().includes("hello"),
});
