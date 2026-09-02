import { Trash2 } from "lucide-react";
import { useState } from "react";

import type { DocumentItem } from "@/lib/vault";
import { formatDate } from "@/lib/vault";

import { FileIcon } from "@/components/FileIcon";

/** Renders recoverable documents and their restore or purge actions. */
export function TrashPage({
  documents,
  onRestore,
  onPurge,
}: {
  documents: DocumentItem[];
  onRestore: (id: string) => Promise<void>;
  onPurge: (id: string) => Promise<void>;
}) {
  const [notice, setNotice] = useState("");
  const softDeleted = documents.filter(
    (item) => item.document.status === "deleted",
  );
  return (
    <section className="vault-page" aria-labelledby="trash-title">
      <div className="vault-page__header">
        <div>
          <h1 id="trash-title">Trash</h1>
          <p>
            Deleted files remain recoverable while their retention window is
            active.
          </p>
        </div>
      </div>
      <div className="vault-notice">
        <Trash2 size={14} aria-hidden="true" /> Items in Trash are hidden from
        the Library until restored.
      </div>
      {notice && (
        <div className="vault-notice" role="status">
          {notice}
        </div>
      )}
      <div className="vault-collection">
        {softDeleted.length === 0 ? (
          <div className="vault-empty">
            <Trash2 size={28} aria-hidden="true" />
            <h2>No deleted documents available</h2>
            <p>
              The current documents endpoint returns ready records only. The
              backend must expose deleted records before Trash can list them.
            </p>
          </div>
        ) : (
          softDeleted.map((item) => (
            <div className="vault-row" key={item.document.id}>
              <div className="vault-doc">
                <span className="vault-doc__icon vault-doc__icon--pdf">
                  <FileIcon type={item.document.mediaType} />
                </span>
                <span className="vault-doc__copy">
                  <span className="vault-doc__title">
                    {item.document.title}
                  </span>
                  <span className="vault-doc__filename">
                    {item.document.originalFilename}
                  </span>
                </span>
              </div>
              <div className="vault-cell vault-cell--muted">
                Deleted{" "}
                {item.document.deletedAt
                  ? formatDate(item.document.deletedAt)
                  : "Deletion time unavailable"}
              </div>
              <div className="vault-cell">
                <button
                  className="vault-member-action"
                  type="button"
                  onClick={() =>
                    void onRestore(item.document.id).then(() =>
                      setNotice("Document restored to the Library."),
                    )
                  }
                >
                  Restore
                </button>
                <button
                  className="vault-member-action"
                  type="button"
                  onClick={() =>
                    void onPurge(item.document.id).then(() =>
                      setNotice("Document permanently removed."),
                    )
                  }
                >
                  Purge
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
