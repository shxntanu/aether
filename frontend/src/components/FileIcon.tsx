import { FileImage, FileText, FileType2 } from "lucide-react";

import { fileKind } from "@/lib/vault";

/** Renders the file-type icon used by document rows and previews. */
export function FileIcon({ type }: { type: string }) {
  const kind = fileKind(type);
  if (kind === "image") return <FileImage aria-hidden="true" />;
  if (kind === "pdf") return <FileText aria-hidden="true" />;
  return <FileType2 aria-hidden="true" />;
}
