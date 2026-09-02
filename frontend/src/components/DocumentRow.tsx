import { MoreHorizontal } from "lucide-react";

import type { DocumentItem } from "@/lib/vault";
import { fileKind, formatBytes, formatDate } from "@/lib/vault";

import { FileIcon } from "@/components/FileIcon";

/** Renders one selectable document in the library's list presentation. */
export function DocumentRow({
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
      className="vault-row"
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
      <div className="vault-doc">
        <span
          className={`vault-doc__icon vault-doc__icon--${fileKind(item.document.mediaType)}`}
        >
          <FileIcon type={item.document.mediaType} />
        </span>
        <span className="vault-doc__copy">
          <span className="vault-doc__title">{item.document.title}</span>
          <span className="vault-doc__filename">
            {item.document.originalFilename}
          </span>
        </span>
      </div>
      <div className="vault-cell">
        <div className="vault-tags">
          {item.tags.length > 0 ? (
            item.tags.map((tag) => (
              <span className="vault-tag" key={tag.id}>
                {tag.displayName}
              </span>
            ))
          ) : (
            <span className="vault-cell--muted">No tags</span>
          )}
        </div>
      </div>
      <div className="vault-cell vault-cell--muted">
        {item.document.uploaderId}
        <br />
        {formatDate(item.document.updatedAt)}
      </div>
      <div className="vault-cell vault-cell--size">
        {formatBytes(item.document.sizeBytes)}
      </div>
      <button
        className="vault-button vault-button--icon vault-more"
        aria-label={`More actions for ${item.document.title}`}
        type="button"
        onClick={(event) => event.stopPropagation()}
      >
        <MoreHorizontal size={17} aria-hidden="true" />
      </button>
    </div>
  );
}
