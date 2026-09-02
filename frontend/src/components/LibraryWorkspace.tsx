import {
  Archive,
  Filter,
  Grid2X2,
  List,
  Plus,
  RotateCcw,
  Tags,
  X,
} from "lucide-react";
import { useMemo, useState } from "react";

import type { Tag } from "@/lib/api";
import {
  type DocumentItem,
  type RouteName,
  type TagMatch,
  type ViewMode,
} from "@/lib/vault";

import { DocumentGridCard } from "@/components/DocumentGridCard";
import { DocumentRow } from "@/components/DocumentRow";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";

/** Renders the searchable, filterable, and sortable document collection. */
export function LibraryWorkspace({
  route,
  documents,
  availableTags,
  selectedId,
  onSelect,
  query,
  onUpload,
  onRefresh,
  loading,
}: {
  route: RouteName;
  documents: DocumentItem[];
  availableTags: Tag[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  query: string;
  onUpload: () => void;
  onRefresh: () => void;
  loading: boolean;
}) {
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [match, setMatch] = useState<TagMatch>("all");
  const [filterTag, setFilterTag] = useState("");
  const [sort, setSort] = useState("updated");
  const filtered = useMemo(
    () =>
      documents
        .filter((item) => {
          const search = query.trim().toLowerCase();
          const matchesSearch =
            !search ||
            [
              item.document.title,
              item.document.originalFilename,
              ...item.tags.map((tag) => tag.displayName),
            ].some((value) => value.toLowerCase().includes(search));
          const matchesTag =
            !filterTag ||
            (match === "all"
              ? item.tags.some((tag) => tag.normalizedName === filterTag)
              : item.tags.some((tag) => tag.normalizedName === filterTag));
          return matchesSearch && matchesTag;
        })
        .sort((a, b) =>
          sort === "title"
            ? a.document.title.localeCompare(b.document.title)
            : sort === "size"
              ? b.document.sizeBytes - a.document.sizeBytes
              : new Date(b.document.updatedAt).getTime() -
                new Date(a.document.updatedAt).getTime(),
        ),
    [documents, filterTag, match, query, sort],
  );
  const title = route === "recent" ? "Recent" : "Library";
  const subtitle =
    route === "recent"
      ? "Your latest additions and changes, close at hand."
      : "One shared collection, organized with reusable tags.";
  return (
    <section className="vault-main__inner" aria-labelledby="workspace-title">
      <div className="vault-heading">
        <div>
          <h1 id="workspace-title">{title}</h1>
          <p>{subtitle}</p>
        </div>
        <div className="vault-heading__actions">
          <Button
            variant="outline"
            className="vault-button vault-button--quiet"
            type="button"
            onClick={onRefresh}
          >
            <RotateCcw size={14} aria-hidden="true" /> Refresh
          </Button>
          <Button
            variant="default"
            className="vault-button vault-button--primary"
            type="button"
            onClick={onUpload}
          >
            <Plus size={15} aria-hidden="true" /> Add document
          </Button>
        </div>
      </div>
      <Separator className="vault-rule" />
      <div className="vault-toolbar">
        <div className="vault-toolbar__left">
          <span className="vault-toolbar__label">
            {filtered.length} documents
          </span>
          <div className="vault-select">
            <Filter size={14} aria-hidden="true" />
            <Select
              value={filterTag || null}
              onValueChange={(value) => setFilterTag(value ?? "")}
            >
              <SelectTrigger aria-label="Filter by tag" className="vault-select__trigger">
                <SelectValue placeholder="All tags" />
              </SelectTrigger>
              <SelectContent align="start" className="vault-select-content">
                {availableTags.map((tag) => (
                  <SelectItem key={tag.id} value={tag.normalizedName}>
                    {tag.displayName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {filterTag && (
            <Button
              variant="ghost"
              className="vault-filter"
              type="button"
              onClick={() => setFilterTag("")}
            >
              <X size={13} aria-hidden="true" /> Clear
            </Button>
          )}
        </div>
        <div className="vault-toolbar__right">
          <div className="vault-select vault-select--sort">
            <span>Sort by</span>
            <Select
              value={sort}
              onValueChange={(value) => setSort(value ?? "updated")}
            >
              <SelectTrigger aria-label="Sort documents" className="vault-select__trigger">
                <SelectValue />
              </SelectTrigger>
              <SelectContent align="end" className="vault-select-content">
                <SelectItem value="updated">Last modified</SelectItem>
                <SelectItem value="title">Title</SelectItem>
                <SelectItem value="size">File size</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {filterTag && (
            <Button
              variant="ghost"
              className="vault-filter"
              type="button"
              onClick={() =>
                setMatch((current) => (current === "all" ? "any" : "all"))
              }
            >
              <Tags size={14} aria-hidden="true" />{" "}
              {match === "all" ? "All tags" : "Any tags"}
            </Button>
          )}
          <div className="vault-view-toggle" aria-label="View mode">
            <Button
              variant="ghost"
              type="button"
              aria-label="List view"
              aria-pressed={viewMode === "list"}
              onClick={() => setViewMode("list")}
            >
              <List size={15} aria-hidden="true" />
            </Button>
            <Button
              variant="ghost"
              type="button"
              aria-label="Grid view"
              aria-pressed={viewMode === "grid"}
              onClick={() => setViewMode("grid")}
            >
              <Grid2X2 size={15} aria-hidden="true" />
            </Button>
          </div>
        </div>
      </div>
      {loading ? (
        <div className="vault-collection">
          <div className="vault-empty">
            <RotateCcw className="spin" aria-hidden="true" />
            <h2>Reading the archive</h2>
            <p>Fetching the latest catalog records.</p>
          </div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="vault-collection">
          <div className="vault-empty">
            <Archive size={28} aria-hidden="true" />
            <h2>No documents match</h2>
            <p>
              Try removing a filter or searching by a different title, filename,
              or tag.
            </p>
            <Button
              variant="outline"
              className="vault-button vault-button--quiet"
              type="button"
              onClick={() => setFilterTag("")}
            >
              Clear filters
            </Button>
          </div>
        </div>
      ) : (
        <div
          className="vault-collection"
          role="listbox"
          aria-label={`${title} documents`}
        >
          {viewMode === "list" ? (
            <>
              <div className="vault-list-head">
                <span>Document</span>
                <span>Tags</span>
                <span>Contributor / date</span>
                <span>Size</span>
                <span />
              </div>
              {filtered.map((item) => (
                <DocumentRow
                  item={item}
                  key={item.document.id}
                  selected={item.document.id === selectedId}
                  onSelect={() => onSelect(item.document.id)}
                />
              ))}
            </>
          ) : (
            <div className="vault-grid">
              {filtered.map((item) => (
                <DocumentGridCard
                  item={item}
                  key={item.document.id}
                  selected={item.document.id === selectedId}
                  onSelect={() => onSelect(item.document.id)}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
