import {
  ChevronRight,
  LoaderCircle,
  Pencil,
  Plus,
  Tags,
  Trash2,
} from "lucide-react";
import { useState, type FormEvent } from "react";

import type { Tag } from "@/lib/api";
import type { DocumentItem } from "@/lib/vault";
import { getErrorMessage } from "@/lib/vault";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

/** Renders reusable tags with controls to create, rename, filter, and delete them. */
export function TagsPage({
  tags,
  documents,
  onOpenTag,
  onCreateTag,
  onUpdateTag,
  onDeleteTag,
}: {
  tags: Tag[];
  documents: DocumentItem[];
  onOpenTag: (tag: Tag) => void;
  onCreateTag: (name: string) => Promise<void>;
  onUpdateTag: (tag: Tag, name: string) => Promise<void>;
  onDeleteTag: (tag: Tag) => Promise<void>;
}) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [message, setMessage] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [confirmingDeleteId, setConfirmingDeleteId] = useState<string | null>(
    null,
  );
  const [workingId, setWorkingId] = useState<string | null>(null);
  const create = async (event: FormEvent) => {
    event.preventDefault();
    if (!name.trim()) return;
    try {
      await onCreateTag(name.trim());
      setName("");
      setCreating(false);
      setMessage("Tag created");
    } catch (error) {
      setMessage(getErrorMessage(error));
    }
  };
  const update = async (event: FormEvent, tag: Tag) => {
    event.preventDefault();
    if (!editName.trim()) return;
    setWorkingId(tag.id);
    setMessage("");
    try {
      await onUpdateTag(tag, editName.trim());
      setEditingId(null);
      setEditName("");
      setMessage("Tag renamed");
    } catch (error) {
      setMessage(getErrorMessage(error));
    } finally {
      setWorkingId(null);
    }
  };
  const remove = async (tag: Tag) => {
    setWorkingId(tag.id);
    setMessage("");
    try {
      await onDeleteTag(tag);
      setConfirmingDeleteId(null);
      setMessage("Tag deleted");
    } catch (error) {
      setMessage(getErrorMessage(error));
    } finally {
      setWorkingId(null);
    }
  };
  return (
    <section className="vault-page" aria-labelledby="tags-title">
      <div className="vault-page__header">
        <div>
          <h1 id="tags-title">Tags</h1>
          <p>
            Reusable labels that make one shared library easier to navigate.
          </p>
        </div>
        {creating ? (
          <form
            className="vault-tag-create-form"
            onSubmit={(event) => void create(event)}
          >
            <Input
              className="vault-edit-input"
              aria-label="New tag name"
              autoFocus
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
            <Button
              variant="default"
              className="vault-button vault-button--primary"
              type="submit"
            >
              Create
            </Button>
            <Button
              variant="outline"
              className="vault-button vault-button--quiet"
              type="button"
              onClick={() => setCreating(false)}
            >
              Cancel
            </Button>
          </form>
        ) : (
          <Button
            variant="outline"
            className="vault-button vault-button--quiet"
            type="button"
            onClick={() => setCreating(true)}
          >
            <Plus size={14} aria-hidden="true" /> New tag
          </Button>
        )}
      </div>
      {message && (
        <div className="vault-notice" role="status">
          {message}
        </div>
      )}
      <Separator className="vault-rule" />
      {tags.length === 0 ? (
        <div className="vault-collection">
          <div className="vault-empty">
            <Tags size={28} aria-hidden="true" />
            <h2>No tags yet</h2>
            <p>
              Create a reusable tag from this view or while editing document
              metadata.
            </p>
          </div>
        </div>
      ) : (
        <div className="vault-tag-board">
          {tags.map((tag) => {
            const documentCount = documents.filter((item) =>
              item.tags.some((entry) => entry.id === tag.id),
            ).length;
            const isWorking = workingId === tag.id;
            return (
              <article className="vault-tag-card" key={tag.id}>
                {editingId === tag.id ? (
                  <form
                    className="vault-tag-card__edit"
                    onSubmit={(event) => void update(event, tag)}
                  >
                    <label htmlFor={`tag-name-${tag.id}`}>Tag name</label>
                    <Input
                      className="vault-edit-input"
                      id={`tag-name-${tag.id}`}
                      autoFocus
                      value={editName}
                      onChange={(event) => setEditName(event.target.value)}
                    />
                    <div className="vault-tag-card__edit-actions">
                      <Button
                        variant="default"
                        className="vault-button vault-button--primary"
                        type="submit"
                        disabled={isWorking || !editName.trim()}
                      >
                        {isWorking ? "Saving…" : "Save"}
                      </Button>
                      <Button
                        variant="outline"
                        className="vault-button vault-button--quiet"
                        type="button"
                        disabled={isWorking}
                        onClick={() => setEditingId(null)}
                      >
                        Cancel
                      </Button>
                    </div>
                  </form>
                ) : (
                  <Button
                    variant="ghost"
                    className="vault-tag-card__open"
                    type="button"
                    onClick={() => onOpenTag(tag)}
                  >
                    <span className="vault-tag-card__top">
                      <strong>{tag.displayName}</strong>
                      <Tags size={16} aria-hidden="true" />
                    </span>
                    <span className="vault-tag-card__count">
                      {documentCount} {documentCount === 1 ? "document" : "documents"}
                      <ChevronRight size={12} aria-hidden="true" />
                    </span>
                  </Button>
                )}
                {confirmingDeleteId === tag.id ? (
                  <div className="vault-tag-card__confirm">
                    <p>
                      Delete this tag from {documentCount}{" "}
                      {documentCount === 1 ? "document" : "documents"}?
                    </p>
                    <Button
                      variant="destructive"
                      className="vault-button vault-button--danger"
                      type="button"
                      disabled={isWorking}
                      onClick={() => void remove(tag)}
                    >
                      {isWorking && (
                        <LoaderCircle
                          className="vault-spin"
                          size={14}
                          aria-hidden="true"
                        />
                      )}
                      {isWorking ? "Deleting…" : "Delete tag"}
                    </Button>
                    <Button
                      variant="outline"
                      className="vault-button vault-button--quiet"
                      type="button"
                      disabled={isWorking}
                      onClick={() => setConfirmingDeleteId(null)}
                    >
                      Keep tag
                    </Button>
                  </div>
                ) : (
                  <div className="vault-tag-card__actions">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="vault-button vault-button--icon"
                      type="button"
                      aria-label={`Rename ${tag.displayName}`}
                      disabled={workingId !== null}
                      onClick={() => {
                        setConfirmingDeleteId(null);
                        setEditingId(tag.id);
                        setEditName(tag.displayName);
                      }}
                    >
                      <Pencil size={14} aria-hidden="true" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="vault-button vault-button--icon vault-tag-card__delete"
                      type="button"
                      aria-label={`Delete ${tag.displayName}`}
                      disabled={workingId !== null}
                      onClick={() => {
                        setEditingId(null);
                        setConfirmingDeleteId(tag.id);
                      }}
                    >
                      <Trash2 size={14} aria-hidden="true" />
                    </Button>
                  </div>
                )}
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}
