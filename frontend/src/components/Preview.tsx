import { FileType2 } from "lucide-react";

import { api } from "@/lib/api";
import type { DocumentItem } from "@/lib/vault";
import { fileKind } from "@/lib/vault";
import fallbackPreview from "@/assets/plates/passport-preview.webp";

/** Renders the inline preview available for a selected document. */
export function Preview({ item }: { item: DocumentItem }) {
  const kind = fileKind(item.document.mediaType);
  if (kind === "image")
    return (
      <div className="vault-preview vault-preview--image">
        <img
          src={api.getDocumentContentUrl(item.document.id)}
          alt={`Preview of ${item.document.title}`}
        />
        <span className="vault-preview__label">
          Image preview · preserved original
        </span>
      </div>
    );
  if (kind === "pdf")
    return (
      <div className="vault-preview vault-preview--document">
        <iframe
          title={`Preview of ${item.document.title}`}
          src={api.getDocumentContentUrl(item.document.id)}
        />
      </div>
    );
  return (
    <div className="vault-preview vault-preview--unsupported">
      <img
        src={fallbackPreview}
        alt=""
        aria-hidden="true"
      />
      <FileType2 size={24} aria-hidden="true" />
      <strong>Preview not available</strong>
      <span>Download the preserved original to open it.</span>
    </div>
  );
}
