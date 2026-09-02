import { ChevronRight, Plus, Tags } from "lucide-react";
import { useState, type FormEvent } from "react";

import type { Tag } from "@/lib/api";
import type { DocumentItem } from "@/lib/vault";
import { getErrorMessage } from "@/lib/vault";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

/** Renders reusable tags and the lightweight tag-creation form. */
export function TagsPage({
  tags,
  documents,
  onOpenTag,
  onCreateTag,
}: {
  tags: Tag[];
  documents: DocumentItem[];
  onOpenTag: (tag: Tag) => void;
  onCreateTag: (name: string) => Promise<void>;
}) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [message, setMessage] = useState("");
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
            onSubmit={(event) => void create(event)}
            style={{ alignItems: "center", display: "flex", gap: 8 }}
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
          {tags.map((tag) => (
            <Button
              variant="ghost"
              className="vault-tag-card"
              key={tag.id}
              type="button"
              onClick={() => onOpenTag(tag)}
            >
              <div className="vault-tag-card__top">
                <strong>{tag.displayName}</strong>
                <Tags size={16} aria-hidden="true" />
              </div>
              <p>
                {
                  documents.filter((item) =>
                    item.tags.some(
                      (entry) =>
                        entry.normalizedName === tag.normalizedName,
                    ),
                  ).length
                } documents in this scope <ChevronRight size={12} aria-hidden="true" />
              </p>
            </Button>
          ))}
        </div>
      )}
    </section>
  );
}
