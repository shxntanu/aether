import type { DocumentItem } from "@/lib/vault";
import { fileKind, formatDate } from "@/lib/vault";

import { FileIcon } from "@/components/FileIcon";
import { Status } from "@/components/Status";

/** Renders one selectable document in the library's grid presentation. */
export function DocumentGridCard({
  item,
  selected,
  onSelect,
}: {
  item: DocumentItem;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <div
      className="vault-grid-card"
      role="option"
      tabIndex={0}
      aria-selected={selected}
      onClick={onSelect}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect();
        }
      }}
    >
      <div
        className={`vault-grid-card__preview vault-preview--${fileKind(item.document.mediaType)}`}
      >
        <FileIcon type={item.document.mediaType} />
      </div>
      <span className="vault-doc__title">{item.document.title}</span>
      <span className="vault-doc__filename">
        {item.document.originalFilename}
      </span>
      <div className="vault-grid-card__meta">
        <Status status={item.document.status} />
        <span className="vault-cell vault-cell--muted">
          {formatDate(item.document.updatedAt)}
        </span>
      </div>
    </div>
  );
}
