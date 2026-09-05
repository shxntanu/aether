import { ExternalLink } from "lucide-react";

import { api } from "@/lib/api";
import type { DocumentItem } from "@/lib/vault";
import { fileKind } from "@/lib/vault";
import fallbackPreview from "@/assets/plates/passport-preview.webp";
import { FileIcon } from "@/components/FileIcon";

/** Renders a provider-backed link for viewing a selected document. */
export function Preview({ item }: { item: DocumentItem }) {
  const kind = fileKind(item.document.mediaType);
  if (kind === "image" || kind === "pdf")
    return <ProviderLinkPreview item={item} kind={kind} />;
  return (
    <div className="vault-preview vault-preview--unsupported">
      <a
        className="vault-preview__link"
        href={api.getDocumentContentUrl(item.document.id)}
        target="_blank"
        rel="noreferrer"
      >
        <img src={fallbackPreview} alt="" aria-hidden="true" />
        <FileIcon type={item.document.mediaType} />
        <strong>Open file</strong>
        <span>View the preserved original in storage.</span>
      </a>
    </div>
  );
}

function ProviderLinkPreview({
  item,
  kind,
}: {
  item: DocumentItem;
  kind: "image" | "pdf";
}) {
  return (
    <div className={`vault-preview vault-preview--${kind}`}>
      <a
        className="vault-preview__link"
        href={api.getDocumentContentUrl(item.document.id)}
        target="_blank"
        rel="noreferrer"
      >
        <FileIcon type={item.document.mediaType} />
        <strong>Open file</strong>
        <span>View the preserved original in storage.</span>
        <ExternalLink
          className="vault-preview__external"
          size={14}
          aria-hidden="true"
        />
      </a>
    </div>
  );
}
