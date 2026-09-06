import {
  ArrowDown,
  ArrowUp,
  CornerDownLeft,
  LoaderCircle,
  Search,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
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
} from "@/lib/api";

const searchDelay = 120;

/** Renders the keyboard-first document search palette and session query cache. */
export function DocumentSearchPalette({
  open,
  onOpenChange,
  onSelect,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (record: DocumentRecord) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const cacheRef = useRef(new Map<string, DocumentSearchResult[]>());
  const abortRef = useRef<AbortController | null>(null);
  const requestRef = useRef(0);
  const listboxID = useId();
  const [query, setQuery] = useState("");
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
        setError("");
      }
      onOpenChange(nextOpen);
    },
    [onOpenChange],
  );

  const closeAndSelect = useCallback(
    (result: DocumentSearchResult) => {
      changeOpen(false);
      onSelect({ document: result.document, tags: result.tags });
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

  useEffect(() => {
    if (!open) return;
    const normalizedQuery = query.trim().toLowerCase();
    abortRef.current?.abort();
    const request = ++requestRef.current;
    const cached = cacheRef.current.get(normalizedQuery);
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
          .searchDocuments(normalizedQuery, 10, controller.signal)
          .then((response) => {
            if (
              request !== requestRef.current ||
              controller.signal.aborted
            ) return;
            cacheRef.current.set(normalizedQuery, response.data.results);
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
      normalizedQuery === "" ? 0 : searchDelay,
    );
    return () => window.clearTimeout(timer);
  }, [open, query]);

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
            Search titles, original filenames, and reusable tags.
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
            placeholder="Search title, filename, or tag"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={handleKeyDown}
          />
          {loading && (
            <LoaderCircle className="vault-spin" size={18} aria-hidden="true" />
          )}
          <kbd>esc</kbd>
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
              <span>Try a title, original filename, or tag.</span>
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
