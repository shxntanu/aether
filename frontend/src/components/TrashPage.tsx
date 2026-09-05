import { Trash2 } from "lucide-react";
import { useState } from "react";

import type { DocumentItem } from "@/lib/vault";
import { formatDate, getErrorMessage } from "@/lib/vault";

import { FileIcon } from "@/components/FileIcon";
import { Button } from "@/components/ui/button";

/** Renders recoverable documents and their restore or purge actions. */
export function TrashPage({
  documents,
  canManage,
  onRestore,
  onPurge,
}: {
  documents: DocumentItem[];
  canManage: boolean;
  onRestore: (id: string) => Promise<void>;
  onPurge: (id: string) => Promise<void>;
}) {
  const [notice, setNotice] = useState("");
  const [workingId, setWorkingId] = useState<string | null>(null);
  const softDeleted = documents.filter(
    (item) => item.document.status === "deleted",
  );
  const runAction = async (
    id: string,
    action: (documentId: string) => Promise<void>,
    success: string,
  ) => {
    setWorkingId(id);
    setNotice("");
    try {
      await action(id);
      setNotice(success);
    } catch (error) {
      setNotice(getErrorMessage(error));
    } finally {
      setWorkingId(null);
    }
  };
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
            <h2>Trash is empty</h2>
            <p>Documents moved here remain recoverable during retention.</p>
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
                {item.document.deletionStatus !== "complete"
                  ? item.document.deletionStatus === "failed"
                    ? "Storage move will retry"
                    : "Moving storage objects…"
                  : `Deleted ${
                      item.document.deletedAt
                        ? formatDate(item.document.deletedAt)
                        : "time unavailable"
                    }`}
              </div>
              <div className="vault-cell">
                {canManage && (
                  <>
                    <Button
                      variant="outline"
                      className="vault-member-action"
                      type="button"
                      disabled={
                        item.document.deletionStatus !== "complete" ||
                        workingId !== null
                      }
                      onClick={() =>
                        void runAction(
                          item.document.id,
                          onRestore,
                          "Document restored to the Library.",
                        )
                      }
                    >
                      {workingId === item.document.id ? "Working…" : "Restore"}
                    </Button>
                    <Button
                      variant="outline"
                      className="vault-member-action"
                      type="button"
                      disabled={
                        item.document.deletionStatus !== "complete" ||
                        workingId !== null
                      }
                      onClick={() =>
                        void runAction(
                          item.document.id,
                          onPurge,
                          "Document permanently removed.",
                        )
                      }
                    >
                      Purge
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
