import {
  ArrowDown,
  ArrowUp,
  CircleX,
  CornerDownLeft,
  LoaderCircle,
  Search,
  Tags,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";

import { FileIcon } from "@/components/FileIcon";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  api,
  type DocumentRecord,
  type DocumentSearchResult,
  type Tag,
} from "@/lib/api";

const searchDelay = 120;

/** Renders the keyboard-first document search palette and session query cache. */
export function DocumentSearchPalette({
  open,
  availableTags,
  onOpenChange,
  onSelect,
}: {
  open: boolean;
  availableTags: Tag[];
  onOpenChange: (open: boolean) => void;
  onSelect: (record: DocumentRecord) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const cacheRef = useRef(new Map<string, DocumentSearchResult[]>());
  const abortRef = useRef<AbortController | null>(null);
  const requestRef = useRef(0);
  const listboxID = useId();
  const tagListboxID = useId();
  const [query, setQuery] = useState("");
  const [tagQuery, setTagQuery] = useState("");
  const [selectedTags, setSelectedTags] = useState<Tag[]>([]);
  const [tagMenuOpen, setTagMenuOpen] = useState(false);
  const [match, setMatch] = useState<"all" | "any">("all");
  const [results, setResults] = useState<DocumentSearchResult[]>([]);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const changeOpen = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        abortRef.current?.abort();
        requestRef.current += 1;
        setQuery("");
        setTagQuery("");
        setSelectedTags([]);
        setTagMenuOpen(false);
        setMatch("all");
        setError("");
      }
      onOpenChange(nextOpen);
    },
    [onOpenChange],
  );

  const closeAndSelect = useCallback(
    (result: DocumentSearchResult) => {
      changeOpen(false);
      onSelect({
        document: result.document,
        tags: result.tags,
        uploaderName: result.uploaderName,
      });
    },
    [changeOpen, onSelect],
  );

  useEffect(() => {
    const handleShortcut = (event: globalThis.KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        changeOpen(true);
      }
    };
    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [changeOpen]);

  useEffect(() => {
    if (!open) return;
    const frame = window.requestAnimationFrame(() => inputRef.current?.focus());
    return () => window.cancelAnimationFrame(frame);
  }, [open]);

  const selectedTagNames = useMemo(
    () => selectedTags.map((tag) => tag.normalizedName),
    [selectedTags],
  );
  const searchCacheKey = `${query.trim().toLowerCase()}\u001f${match}\u001f${selectedTagNames.join("\u001f")}`;

  useEffect(() => {
    if (!open) return;
    const normalizedQuery = query.trim().toLowerCase();
    abortRef.current?.abort();
    const request = ++requestRef.current;
    const cached = cacheRef.current.get(searchCacheKey);
    if (cached) {
      setResults(cached);
      setActiveIndex(cached.length > 0 ? 0 : -1);
      setLoading(false);
      setError("");
      return;
    }

    const timer = window.setTimeout(
      () => {
        const controller = new AbortController();
        abortRef.current = controller;
        setLoading(true);
        setError("");
        void api
          .searchDocumentsWithFilters(
            normalizedQuery,
            { tags: selectedTagNames, match },
            10,
            controller.signal,
          )
          .then((response) => {
            if (
              request !== requestRef.current ||
              controller.signal.aborted
            ) return;
            cacheRef.current.set(searchCacheKey, response.data.results);
            setResults(response.data.results);
            setActiveIndex(response.data.results.length > 0 ? 0 : -1);
          })
          .catch(() => {
            if (
              request !== requestRef.current ||
              controller.signal.aborted
            ) return;
            setError(
              "Search is unavailable. Check your connection and try again.",
            );
          })
          .finally(() => {
            if (request === requestRef.current && !controller.signal.aborted) {
              setLoading(false);
            }
          });
      },
      normalizedQuery === "" && selectedTagNames.length === 0 ? 0 : searchDelay,
    );
    return () => window.clearTimeout(timer);
  }, [match, open, query, searchCacheKey, selectedTagNames]);

  const tagSuggestions = useMemo(() => {
    const normalizedTagQuery = tagQuery.trim().toLowerCase();
    const selectedNames = new Set(selectedTagNames);
    return availableTags
      .filter(
        (tag) =>
          !selectedNames.has(tag.normalizedName) &&
          (normalizedTagQuery === "" ||
            tag.displayName.toLowerCase().includes(normalizedTagQuery)),
      )
      .slice(0, 8);
  }, [availableTags, selectedTagNames, tagQuery]);

  const selectTag = (tag: Tag) => {
    setSelectedTags((current) =>
      current.some((entry) => entry.normalizedName === tag.normalizedName)
        ? current
        : [...current, tag],
    );
    setTagQuery("");
    setTagMenuOpen(true);
  };

  const removeTag = (tag: Tag) => {
    setSelectedTags((current) =>
      current.filter((entry) => entry.normalizedName !== tag.normalizedName),
    );
  };

  const moveActive = (direction: 1 | -1) => {
    setActiveIndex((current) => {
      const next =
        direction === 1
          ? (current + 1) % results.length
          : current <= 0
            ? results.length - 1
            : current - 1;
      window.requestAnimationFrame(() => {
        document
          .getElementById(`${listboxID}-option-${next}`)
          ?.scrollIntoView({ block: "nearest" });
      });
      return next;
    });
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (results.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      moveActive(1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      moveActive(-1);
    } else if (event.key === "Enter" && activeIndex >= 0) {
      event.preventDefault();
      closeAndSelect(results[activeIndex]);
    }
  };

  const status = loading
    ? "Searching the archive"
    : `${results.length} ${results.length === 1 ? "document" : "documents"} found`;

  return (
    <Dialog open={open} onOpenChange={changeOpen}>
      <DialogContent
        showCloseButton={false}
        className="vault-command-palette"
      >
        <DialogHeader className="vault-command-palette__header">
          <DialogTitle>Find a document</DialogTitle>
          <DialogDescription>
            Search titles, original filenames, or choose one or more tags.
          </DialogDescription>
        </DialogHeader>
        <div className="vault-command-palette__input-wrap">
          <Search size={18} aria-hidden="true" />
          <input
            ref={inputRef}
            role="combobox"
            aria-autocomplete="list"
            aria-controls={listboxID}
            aria-expanded="true"
            aria-activedescendant={
              activeIndex >= 0 ? `${listboxID}-option-${activeIndex}` : undefined
            }
            maxLength={200}
            placeholder="Search title, filename, date, or tag"
            value={query}
            onFocus={() => setTagMenuOpen(false)}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={handleKeyDown}
          />
          {loading && (
            <LoaderCircle className="vault-spin" size={18} aria-hidden="true" />
          )}
          <kbd>esc</kbd>
        </div>
        <div className="vault-command-palette__tag-filter">
          <div className="vault-command-palette__tag-heading">
            <span>
              <Tags size={14} aria-hidden="true" /> Filter by tags
            </span>
            {selectedTags.length > 1 && (
              <button
                className="vault-command-palette__match"
                type="button"
                aria-pressed={match === "all"}
                onClick={() =>
                  setMatch((current) => (current === "all" ? "any" : "all"))
                }
              >
                Match {match === "all" ? "all" : "any"}
              </button>
            )}
          </div>
          <div className="vault-command-palette__tag-input">
            {selectedTags.map((tag) => (
              <span className="vault-command-palette__selected-tag" key={tag.id}>
                <span>{tag.displayName}</span>
                <button
                  type="button"
                  aria-label={`Remove ${tag.displayName} filter`}
                  onClick={() => removeTag(tag)}
                >
                  <CircleX size={12} aria-hidden="true" />
                </button>
              </span>
            ))}
            <input
              aria-label="Add a tag filter"
              aria-controls={tagListboxID}
              aria-expanded={tagMenuOpen}
              placeholder={
                selectedTags.length > 0 ? "Add another tag" : "Add a tag"
              }
              maxLength={100}
              value={tagQuery}
              onFocus={() => setTagMenuOpen(true)}
              onChange={(event) => {
                setTagQuery(event.target.value);
                setTagMenuOpen(true);
              }}
              onKeyDown={(event) => {
                if (event.key === "Escape") {
                  setTagMenuOpen(false);
                  event.stopPropagation();
                }
                if (event.key === "Enter" && tagSuggestions[0]) {
                  event.preventDefault();
                  selectTag(tagSuggestions[0]);
                }
              }}
            />
          </div>
          {tagMenuOpen && tagSuggestions.length > 0 && (
            <div
              id={tagListboxID}
              className="vault-command-palette__tag-menu"
              role="listbox"
              aria-label="Available tags"
            >
              {tagSuggestions.map((tag) => (
                <button
                  type="button"
                  role="option"
                  aria-selected={false}
                  key={tag.id}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => selectTag(tag)}
                >
                  {tag.displayName}
                  {tag.implicit && <span>date</span>}
                </button>
              ))}
            </div>
          )}
        </div>
        <p className="sr-only" role="status" aria-live="polite">
          {status}
        </p>
        <div
          id={listboxID}
          className="vault-command-palette__results"
          role="listbox"
          aria-label="Document search results"
          aria-busy={loading}
        >
          {results.map((result, index) => (
            <button
              id={`${listboxID}-option-${index}`}
              key={result.document.id}
              className="vault-command-result"
              data-active={index === activeIndex ? "true" : undefined}
              type="button"
              role="option"
              aria-selected={index === activeIndex}
              onMouseEnter={() => setActiveIndex(index)}
              onClick={() => closeAndSelect(result)}
            >
              <span className="vault-command-result__icon">
                <FileIcon type={result.document.mediaType} />
              </span>
              <span className="vault-command-result__body">
                <strong>{result.document.title}</strong>
                <span className="vault-command-result__filename">
                  {result.document.originalFilename}
                </span>
                {result.tags.length > 0 && (
                  <span className="vault-command-result__tags">
                    {result.tags.slice(0, 3).map((tag) => (
                      <span key={tag.id}>{tag.displayName}</span>
                    ))}
                    {result.tags.length > 3 && (
                      <span>+{result.tags.length - 3}</span>
                    )}
                  </span>
                )}
                {result.evidence.length > 0 && (
                  <span className="vault-command-result__evidence">
                    Matched {result.evidence
                      .map((evidence) => `${evidence.field}: ${evidence.value}`)
                      .join(" · ")}
                  </span>
                )}
              </span>
              <CornerDownLeft
                className="vault-command-result__enter"
                size={16}
                aria-hidden="true"
              />
            </button>
          ))}
          {!loading && !error && results.length === 0 && (
            <div className="vault-command-palette__empty">
              <strong>No documents found</strong>
              <span>Try a title, original filename, date, or tag.</span>
            </div>
          )}
          {error && (
            <div className="vault-command-palette__error" role="alert">
              {error}
            </div>
          )}
        </div>
        <footer className="vault-command-palette__footer" aria-hidden="true">
          <span>
            <ArrowUp size={13} />
            <ArrowDown size={13} /> Navigate
          </span>
          <span>
            <CornerDownLeft size={13} /> Open
          </span>
          <span>esc Close</span>
        </footer>
      </DialogContent>
    </Dialog>
  );
}
