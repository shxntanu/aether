import {
  CircleAlert,
  Download,
  Trash2,
  X,
} from "lucide-react";
import { useState, type FormEvent } from "react";

import type { DocumentItem } from "@/lib/vault";
import { formatBytes, formatDate, getErrorMessage } from "@/lib/vault";

import { Preview } from "@/components/Preview";
import { Status } from "@/components/Status";

/** Renders and saves editable metadata for the selected document. */
export function Inspector({
  item,
  onClose,
  onUpdate,
  onDelete,
  onDownload,
}: {
  item: DocumentItem;
  onClose: () => void;
  onUpdate: (title: string, tags: string[]) => Promise<void>;
  onDelete: () => Promise<void>;
  onDownload: () => void;
}) {
  const [title, setTitle] = useState(item.document.title);
  const [tags, setTags] = useState(item.tags.map((tag) => tag.displayName));
  const [newTag, setNewTag] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    try {
      await onUpdate(title, tags);
      setMessage("Metadata saved");
    } catch (error) {
      setMessage(getErrorMessage(error));
    } finally {
      setSaving(false);
    }
  };
  return (
    <aside className="vault-inspector" aria-label="Document inspector">
      <div className="vault-inspector__head">
        <h2>Archive inspector</h2>
        <button
          className="vault-button vault-button--icon"
          aria-label="Close inspector"
          type="button"
          onClick={onClose}
        >
          <X size={17} aria-hidden="true" />
        </button>
      </div>
      <form
        className="vault-inspector__body"
        onSubmit={(event) => void submit(event)}
      >
        <Preview item={item} />
        <h3 className="vault-inspector__title">{item.document.title}</h3>
        <p className="vault-inspector__filename">
          {item.document.originalFilename}
        </p>
        <dl className="vault-facts">
          <div className="vault-fact">
            <dt>Format</dt>
            <dd>{item.document.mediaType}</dd>
          </div>
          <div className="vault-fact">
            <dt>File size</dt>
            <dd>{formatBytes(item.document.sizeBytes)}</dd>
          </div>
          <div className="vault-fact">
            <dt>Uploaded</dt>
            <dd>{formatDate(item.document.createdAt)}</dd>
          </div>
          <div className="vault-fact">
            <dt>Last changed</dt>
            <dd>{formatDate(item.document.updatedAt)}</dd>
          </div>
          <div className="vault-fact">
            <dt>Status</dt>
            <dd>
              <Status status={item.document.status} />
            </dd>
          </div>
        </dl>
        <section className="vault-inspector__section">
          <h3>Catalog metadata</h3>
          <label className="vault-toolbar__label" htmlFor="inspector-title">
            Display title
          </label>
          <input
            className="vault-edit-input"
            id="inspector-title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
          <h3 style={{ marginTop: 18 }}>Reusable tags</h3>
          <div className="vault-tag-editor">
            {tags.map((tag) => (
              <span className="vault-tag" key={tag}>
                {tag}
                <button
                  type="button"
                  aria-label={`Remove ${tag} tag`}
                  onClick={() =>
                    setTags((current) =>
                      current.filter((entry) => entry !== tag),
                    )
                  }
                >
                  <X size={11} aria-hidden="true" />
                </button>
              </span>
            ))}
            <input
              className="vault-edit-input vault-add-tag-input"
              aria-label="Add a tag"
              placeholder="Add a tag"
              value={newTag}
              onChange={(event) => setNewTag(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && newTag.trim()) {
                  event.preventDefault();
                  setTags((current) => [...current, newTag.trim()]);
                  setNewTag("");
                }
              }}
            />
          </div>
        </section>
        {message && (
          <div className="vault-notice" role="status">
            <CircleAlert size={14} aria-hidden="true" /> {message}
          </div>
        )}
        <div className="vault-inspector__actions">
          <button
            className="vault-button vault-button--quiet"
            type="button"
            onClick={onDownload}
          >
            <Download size={14} aria-hidden="true" /> Download
          </button>
          <button
            className="vault-button vault-button--primary"
            type="submit"
            disabled={saving}
          >
            {saving ? "Saving…" : "Save changes"}
          </button>
        </div>
      </form>
      <div className="vault-inspector__footer">
        <button
          className="vault-button vault-button--danger"
          type="button"
          onClick={() => void onDelete()}
        >
          <Trash2 size={14} aria-hidden="true" /> Move to Trash
        </button>
      </div>
    </aside>
  );
}
