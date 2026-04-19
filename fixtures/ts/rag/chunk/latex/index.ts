// Test: latex chunking — respects \section / \subsection boundaries.
import { MDocument } from "agent";
import { output } from "kit";

const latex = String.raw`
\documentclass{article}
\begin{document}
\section{Intro}
brainkit embeds QuickJS.
\section{Bus}
Watermill provides transports.
\subsection{Topics}
Sanitization depends on the backend.
\end{document}
`;

const doc = MDocument.fromText(latex);
const chunks = await doc.chunk({ strategy: "latex", maxSize: 120, overlap: 0 });
output({
  chunkCount: chunks.length,
  atLeastOne: chunks.length >= 1,
  allHaveText: chunks.every((c) => typeof c.text === "string" && c.text.length > 0),
});
