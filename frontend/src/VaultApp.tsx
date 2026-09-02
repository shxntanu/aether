import {
  Archive,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  Clock3,
  Download,
  FileImage,
  FileText,
  FileType2,
  Filter,
  Grid2X2,
  List,
  LogOut,
  Menu,
  MoreHorizontal,
  Plus,
  RotateCcw,
  Search,
  Shield,
  Tags,
  Trash2,
  Upload,
  UserRound,
  X,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type FormEvent,
} from "react";
import {
  api,
  type DocumentRecord,
  type Member,
  type Session,
  type Tag,
} from "@/lib/api";
import "./VaultApp.css";

type RouteName = "library" | "recent" | "tags" | "trash" | "members";
type ViewMode = "list" | "grid";
type TagMatch = "all" | "any";

type DocumentItem = DocumentRecord;

function initials(name: string): string {
  return name
    .split(" ")
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(bytes > 5 * 1024 * 1024 ? 1 : 2)} MB`;
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat("en-IN", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(new Date(value));
}

function fileKind(mediaType: string): "pdf" | "image" | "file" {
  if (mediaType === "application/pdf") return "pdf";
  if (mediaType.startsWith("image/")) return "image";
  return "file";
}

function getRoute(pathname: string): RouteName {
  if (pathname === "/library/recent") return "recent";
  if (pathname.startsWith("/tags")) return "tags";
  if (pathname.startsWith("/trash")) return "trash";
  if (pathname.startsWith("/admin/members")) return "members";
  return "library";
}

function getDocumentIdFromPath(pathname: string): string | null {
  const match = pathname.match(/^\/documents\/([^/]+)$/);
  return match ? decodeURIComponent(match[1]) : null;
}

function navigate(pathname: string): void {
  window.history.pushState({}, "", pathname);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

function getErrorMessage(error: unknown): string {
  if (
    error &&
    typeof error === "object" &&
    "message" in error &&
    typeof error.message === "string"
  )
    return error.message;
  return "The vault could not complete that request.";
}

function FileIcon({ type }: { type: string }) {
  const kind = fileKind(type);
  if (kind === "image") return <FileImage aria-hidden="true" />;
  if (kind === "pdf") return <FileText aria-hidden="true" />;
  return <FileType2 aria-hidden="true" />;
}

function Brand() {
  return (
    <a
      className="vault-brand"
      href="/library"
      onClick={(event) => {
        event.preventDefault();
        navigate("/library");
      }}
    >
      <span className="vault-brand__mark">A</span>
      <span className="vault-brand__name">AETHER</span>
    </a>
  );
}

function NavItem({
  href,
  route,
  icon: Icon,
  label,
  count,
}: {
  href: string;
  route: RouteName;
  icon: typeof Archive;
  label: string;
  count?: string;
}) {
  return (
    <a
      href={href}
      aria-current={
        getRoute(window.location.pathname) === route ? "page" : undefined
      }
      onClick={(event) => {
        event.preventDefault();
        navigate(href);
      }}
    >
      <Icon aria-hidden="true" />
      <span>{label}</span>
      {count && <span className="vault-nav__count">{count}</span>}
    </a>
  );
}

function AccountMenu({
  onLogout,
  onClose,
}: {
  onLogout: () => void;
  onClose: () => void;
}) {
  return (
    <div className="vault-account-menu" role="menu">
      <button
        type="button"
        onClick={() => {
          onClose();
          onLogout();
        }}
      >
        <LogOut aria-hidden="true" /> Sign out
      </button>
    </div>
  );
}

function Navigation({
  session,
  documents,
  tags,
  open,
  onClose,
}: {
  session: Session;
  documents: DocumentItem[];
  tags: Tag[];
  open: boolean;
  onClose: () => void;
}) {
  const isAdmin = session.member.role === "admin";
  return (
    <>
      {open && (
        <button
          className="vault-overlay"
          aria-label="Close navigation"
          type="button"
          onClick={onClose}
        />
      )}
      <aside className={`vault-rail ${open ? "is-open" : ""}`}>
        <span className="vault-rail__label">Your archive</span>
        <nav className="vault-nav" aria-label="Primary navigation">
          <NavItem
            href="/library"
            route="library"
            icon={Archive}
            label="Library"
            count={String(documents.filter((item) => item.document.status !== "deleted").length)}
          />
          <NavItem
            href="/library/recent"
            route="recent"
            icon={Clock3}
            label="Recent"
          />
          <NavItem
            href="/tags"
            route="tags"
            icon={Tags}
            label="Tags"
            count={String(tags.length)}
          />
          <NavItem
            href="/trash"
            route="trash"
            icon={Trash2}
            label="Trash"
            count={String(documents.filter((item) => item.document.status === "deleted").length)}
          />
        </nav>
        {isAdmin && (
          <>
            <div className="vault-rail__rule" />
            <span className="vault-rail__label">Administration</span>
            <nav className="vault-nav">
              <NavItem
                href="/admin/members"
                route="members"
                icon={Shield}
                label="Members"
              />
            </nav>
          </>
        )}
        <div className="vault-rail__note">
          <strong>Private by design.</strong>
          <p>
            Original files stay preserved. Metadata makes them easier to find.
          </p>
          <span>Session authenticated</span>
        </div>
      </aside>
    </>
  );
}

function TopBar({
  session,
  onMenu,
  onSearch,
  onUpload,
  onLogout,
}: {
  session: Session;
  onMenu: () => void;
  onSearch: (value: string) => void;
  onUpload: () => void;
  onLogout: () => void;
}) {
  const [accountOpen, setAccountOpen] = useState(false);
  return (
    <header className="vault-topbar">
      <button
        className="vault-mobile-menu"
        aria-label="Open navigation"
        type="button"
        onClick={onMenu}
      >
        <Menu aria-hidden="true" />
      </button>
      <Brand />
      <label className="vault-search">
        <Search size={16} aria-hidden="true" />
        <input
          aria-label="Search your archive"
          placeholder="Search title, filename, or tag"
          onChange={(event) => onSearch(event.target.value)}
        />
        <span className="vault-search__hint">⌘ K</span>
      </label>
      <div className="vault-topbar__actions">
        <button
          className="vault-button vault-button--primary"
          type="button"
          onClick={onUpload}
        >
          <Upload size={15} aria-hidden="true" /> Upload
        </button>
        <button
          className="vault-account"
          type="button"
          aria-expanded={accountOpen}
          onClick={() => setAccountOpen((current) => !current)}
        >
          <span className="vault-account__avatar">
            {initials(session.member.displayName)}
          </span>
          <span className="vault-account__copy">
            <span className="vault-account__name">
              {session.member.displayName}
            </span>
            <span className="vault-account__role">
              {session.member.role === "admin" ? "Administrator" : "Member"}
            </span>
          </span>
          <ChevronDown size={14} aria-hidden="true" />
        </button>
      </div>
      {accountOpen && (
        <AccountMenu
          onLogout={onLogout}
          onClose={() => setAccountOpen(false)}
        />
      )}
    </header>
  );
}

function Status({ status }: { status: string }) {
  if (status === "ready")
    return <span className="vault-status vault-status--ready">Ready</span>;
  if (status === "failed")
    return (
      <span className="vault-status vault-status--failed">Needs attention</span>
    );
  if (status === "deleted")
    return <span className="vault-status vault-status--failed">Deleted</span>;
  return (
    <span className="vault-status vault-status--processing">Processing</span>
  );
}

function DocumentRow({
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

function DocumentGridCard({
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

function LibraryWorkspace({
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
          <button
            className="vault-button vault-button--quiet"
            type="button"
            onClick={onRefresh}
          >
            <RotateCcw size={14} aria-hidden="true" /> Refresh
          </button>
          <button
            className="vault-button vault-button--primary"
            type="button"
            onClick={onUpload}
          >
            <Plus size={15} aria-hidden="true" /> Add document
          </button>
        </div>
      </div>
      <div className="vault-rule" />
      <div className="vault-toolbar">
        <div className="vault-toolbar__left">
          <span className="vault-toolbar__label">
            {filtered.length} documents
          </span>
          <label className="vault-select">
            <Filter size={14} aria-hidden="true" />
            <select aria-label="Filter by tag" value={filterTag} onChange={(event) => setFilterTag(event.target.value)}>
              <option value="">All tags</option>
              {availableTags.map((tag) => <option key={tag.id} value={tag.normalizedName}>{tag.displayName}</option>)}
            </select>
            <ChevronDown size={12} aria-hidden="true" />
          </label>
          {filterTag && (
            <button
              className="vault-filter"
              type="button"
              onClick={() => setFilterTag("")}
            >
              <X size={13} aria-hidden="true" /> Clear
            </button>
          )}
        </div>
        <div className="vault-toolbar__right">
          <label className="vault-select">
            Sort by{" "}
            <select
              aria-label="Sort documents"
              value={sort}
              onChange={(event) => setSort(event.target.value)}
            >
              <option value="updated">Last modified</option>
              <option value="title">Title</option>
              <option value="size">File size</option>
            </select>
            <ChevronDown size={12} aria-hidden="true" />
          </label>
          {filterTag && <button className="vault-filter" type="button" onClick={() => setMatch((current) => current === "all" ? "any" : "all")}><Tags size={14} aria-hidden="true" /> {match === "all" ? "All tags" : "Any tags"}</button>}
          <div className="vault-view-toggle" aria-label="View mode">
            <button
              type="button"
              aria-label="List view"
              aria-pressed={viewMode === "list"}
              onClick={() => setViewMode("list")}
            >
              <List size={15} aria-hidden="true" />
            </button>
            <button
              type="button"
              aria-label="Grid view"
              aria-pressed={viewMode === "grid"}
              onClick={() => setViewMode("grid")}
            >
              <Grid2X2 size={15} aria-hidden="true" />
            </button>
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
            <button
              className="vault-button vault-button--quiet"
              type="button"
              onClick={() => setFilterTag("")}
            >
              Clear filters
            </button>
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

function Preview({ item }: { item: DocumentItem }) {
  const kind = fileKind(item.document.mediaType);
  if (kind === "image")
    return (
      <div className="vault-preview vault-preview--image">
        <img
          src={api.getDocumentContentUrl(item.document.id)}
          alt={`Preview of ${item.document.title}`}
        />
        <span className="vault-preview__label">
          Image preview · preserved original
        </span>
      </div>
    );
  if (kind === "pdf")
    return (
      <div className="vault-preview vault-preview--document">
        <iframe
          title={`Preview of ${item.document.title}`}
          src={api.getDocumentContentUrl(item.document.id)}
        />
      </div>
    );
  return (
    <div className="vault-preview vault-preview--unsupported">
      <FileType2 size={24} aria-hidden="true" />
      <strong>Preview not available</strong>
      <span>Download the preserved original to open it.</span>
    </div>
  );
}

function Inspector({
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

function TagsPage({
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
          <form onSubmit={(event) => void create(event)} style={{ alignItems: "center", display: "flex", gap: 8 }}>
            <input className="vault-edit-input" aria-label="New tag name" autoFocus value={name} onChange={(event) => setName(event.target.value)} />
            <button className="vault-button vault-button--primary" type="submit">Create</button>
            <button className="vault-button vault-button--quiet" type="button" onClick={() => setCreating(false)}>Cancel</button>
          </form>
        ) : (
          <button className="vault-button vault-button--quiet" type="button" onClick={() => setCreating(true)}>
            <Plus size={14} aria-hidden="true" /> New tag
          </button>
        )}
      </div>
      {message && <div className="vault-notice" role="status">{message}</div>}
      <div className="vault-rule" />
      {tags.length === 0 ? (
        <div className="vault-collection">
          <div className="vault-empty">
            <Tags size={28} aria-hidden="true" />
            <h2>No tags yet</h2>
            <p>Create a reusable tag from this view or while editing document metadata.</p>
          </div>
        </div>
      ) : <div className="vault-tag-board">
        {tags.map((tag) => (
          <button
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
                    (entry) => entry.normalizedName === tag.normalizedName,
                  ),
                ).length
              }{" "}
              documents in this scope{" "}
              <ChevronRight size={12} aria-hidden="true" />
            </p>
          </button>
        ))}
      </div>}
    </section>
  );
}

function TrashPage({
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
              The current documents endpoint returns ready records only. The backend must expose deleted records before Trash can list them.
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

function MembersPage({
  members,
  onAdd,
  onUpdate,
}: {
  members: Member[];
  onAdd: (email: string, role: "member" | "admin") => Promise<void>;
  onUpdate: (member: Member) => Promise<void>;
}) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"member" | "admin">("member");
  const [notice, setNotice] = useState("");
  const add = async (event: FormEvent) => {
    event.preventDefault();
    if (!email.trim()) return;
    try {
      await onAdd(email.trim(), role);
      setEmail("");
      setNotice("Member added to the allowlist.");
    } catch (error) {
      setNotice(getErrorMessage(error));
    }
  };
  return (
    <section className="vault-page" aria-labelledby="members-title">
      <div className="vault-page__header">
        <div>
          <h1 id="members-title">Members</h1>
          <p>Maintain the allowlist and access level for your private vault.</p>
        </div>
      </div>
      <div className="vault-rule" />
      <form className="vault-notice" onSubmit={(event) => void add(event)}>
        <UserRound size={14} aria-hidden="true" />
        <label htmlFor="member-email" style={{ flex: 1 }}>
          <span className="vault-toolbar__label">Add email</span>
          <input
            className="vault-edit-input"
            id="member-email"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="name@example.com"
          />
        </label>
        <select
          className="vault-edit-input"
          aria-label="New member role"
          value={role}
          onChange={(event) =>
            setRole(event.target.value as "member" | "admin")
          }
          style={{ width: 120 }}
        >
          <option value="member">Member</option>
          <option value="admin">Admin</option>
        </select>
        <button className="vault-button vault-button--primary" type="submit">
          <Plus size={14} aria-hidden="true" /> Add
        </button>
      </form>
      {notice && (
        <div className="vault-notice" role="status">
          {notice}
        </div>
      )}
      <div className="vault-members">
        <div className="vault-member-row vault-member-row--head">
          <span>Person</span>
          <span>Role</span>
          <span>Status</span>
          <span />
        </div>
        {members.map((member) => (
          <div className="vault-member-row" key={member.id}>
            <div className="vault-member">
              <span className="vault-member__avatar">
                {initials(member.displayName)}
              </span>
              <span>
                <strong>{member.displayName}</strong>
                <span>{member.email}</span>
              </span>
            </div>
            <span className="vault-role">
              {member.role === "admin" ? "Administrator" : "Member"}
            </span>
            <span>
              <Status
                status={member.status === "active" ? "ready" : "failed"}
              />
            </span>
            <button
              className="vault-member-action"
              type="button"
              onClick={() =>
                void onUpdate({
                  ...member,
                  status: member.status === "active" ? "disabled" : "active",
                }).then(() =>
                  setNotice(
                    `${member.displayName} is now ${member.status === "active" ? "disabled" : "active"}.`,
                  ),
                )
              }
            >
              {member.status === "active" ? "Disable" : "Reactivate"}
            </button>
          </div>
        ))}
      </div>
    </section>
  );
}

const supportedUploadTypes = new Set([
  "application/pdf",
  "image/jpeg",
  "image/png",
  "image/webp",
]);

function UploadQueue({
  files,
  onClose,
  onComplete,
}: {
  files: File[];
  onClose: () => void;
  onComplete: (item: DocumentItem, finished: boolean) => void;
}) {
  const [index, setIndex] = useState(0);
  const [progress, setProgress] = useState<number | null>(null);
  const [status, setStatus] = useState("Uploading original…");
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const file = files[index];
  useEffect(() => {
    if (!file) return;
    let active = true;
    const upload = async () => {
      setFailed(false);
      setProgress(null);
      setStatus("Uploading original…");
      try {
        const result = await api.uploadDocument(
          file,
          {
            onProgress: (state) => {
              if (!active) return;
              if (state.state === "complete") setProgress(100);
              else if (state.percent !== null) setProgress(state.percent);
            },
          },
          `aether-${file.name}-${file.size}-${file.lastModified}`,
        );
        if (!active) return;
        setProgress(100);
        setStatus("Upload complete");
        onComplete(
          result.data,
          index === files.length - 1,
        );
        if (index < files.length - 1) setIndex((current) => current + 1);
      } catch (error) {
        if (!active) return;
        setFailed(true);
        setStatus(getErrorMessage(error));
      }
    };
    void upload();
    return () => {
      active = false;
    };
  }, [attempt, file, files.length, index, onComplete]);
  if (!file) return null;
  return (
    <div className="vault-upload-queue" role="status" aria-live="polite">
      <div className="vault-upload-queue__head">
        <strong>Upload queue</strong>
        <span>
          {failed ? `${index + 1} failed` : `${index + 1} of ${files.length}`}
        </span>
        <button
          className="vault-upload-queue__close"
          type="button"
          aria-label="Close upload queue"
          onClick={onClose}
        >
          <X size={15} aria-hidden="true" />
        </button>
      </div>
      <div className="vault-upload-item">
        <Upload size={15} aria-hidden="true" />
        <span className="vault-upload-item__copy">
          <strong>{file.name}</strong>
          <span>
            {progress === null || failed ? status : `${status} ${progress}%`}
          </span>
          <span className="vault-upload-progress">
            <span style={{ width: progress === null ? "38%" : `${progress}%` }} />
          </span>
        </span>
        {failed && (
          <button
            className="vault-member-action"
            type="button"
            onClick={() => setAttempt((current) => current + 1)}
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}

function UploadDialog({
  onClose,
  onUpload,
}: {
  onClose: () => void;
  onUpload: (files: File[]) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [validationError, setValidationError] = useState("");
  const acceptFiles = (files: File[]) => {
    const invalid = files.find(
      (file) =>
        file.size > 50 * 1024 * 1024 || !supportedUploadTypes.has(file.type),
    );
    if (invalid) {
      setValidationError(
        `${invalid.name} is not supported. Choose a PDF, JPEG, PNG, or WebP up to 50 MB.`,
      );
      return;
    }
    setValidationError("");
    onUpload(files);
  };
  const choose = (event: ChangeEvent<HTMLInputElement>) => {
    acceptFiles(Array.from(event.target.files ?? []));
  };
  return (
    <div className="vault-overlay" role="presentation">
      <div
        className="vault-login__sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby="upload-dialog-title"
        style={{ margin: "12vh auto", maxWidth: 500 }}
      >
        <div
          style={{
            alignItems: "center",
            display: "flex",
            justifyContent: "space-between",
          }}
        >
          <h2
            id="upload-dialog-title"
            style={{ fontSize: 22, letterSpacing: "-.04em", margin: 0 }}
          >
            Add to the archive
          </h2>
          <button
            className="vault-button vault-button--icon"
            type="button"
            aria-label="Close upload dialog"
            onClick={onClose}
          >
            <X size={17} aria-hidden="true" />
          </button>
        </div>
        <p>
          Original files are preserved exactly as uploaded. PDF, JPEG, PNG, and
          WebP files up to 50 MB are supported.
        </p>
        <button
          className="vault-button vault-button--primary"
          type="button"
          onClick={() => inputRef.current?.click()}
        >
          <Upload size={15} aria-hidden="true" /> Choose files
        </button>
        <button
          type="button"
          onDragEnter={(event) => {
            event.preventDefault();
            setDragging(true);
          }}
          onDragOver={(event) => event.preventDefault()}
          onDragLeave={() => setDragging(false)}
          onDrop={(event) => {
            event.preventDefault();
            setDragging(false);
            acceptFiles(Array.from(event.dataTransfer.files));
          }}
          onClick={() => inputRef.current?.click()}
          style={{
            background: dragging ? "#f7fbed" : "#f5f6f2",
            border: "1px dashed #cbd3cc",
            borderRadius: 8,
            color: "#51606a",
            cursor: "pointer",
            display: "block",
            fontSize: 12,
            marginTop: 12,
            padding: "20px 14px",
            textAlign: "center",
            width: "100%",
          }}
        >
          Drop files here, or click to browse
        </button>
        {validationError && (
          <div className="vault-notice" role="alert" style={{ marginTop: 14 }}>
            <CircleAlert size={14} aria-hidden="true" /> {validationError}
          </div>
        )}
        <input
          ref={inputRef}
          hidden
          type="file"
          multiple
          accept="application/pdf,image/jpeg,image/png,image/webp"
          onChange={choose}
        />
        <p className="vault-login__note">
          You can add a title and reusable tags from the document inspector
          after upload.
        </p>
      </div>
    </div>
  );
}

function SignedOut({ disabled = false }: { disabled?: boolean }) {
  return (
    <main className="vault-login">
      <section className="vault-login__sheet">
        <Brand />
        <h1>
          {disabled ? "Access needs attention." : "Your private archive."}
        </h1>
        <p>
          {disabled
            ? "This account is not currently enabled for Aether. Contact an administrator to restore access."
            : "A considered home for the documents your team needs to keep close, with original files preserved."}
        </p>
        {!disabled && (
          <a
            className="vault-button vault-button--primary"
            href="/auth/google/start"
          >
            <Shield size={15} aria-hidden="true" /> Continue with Google
          </a>
        )}
        <p className="vault-login__note">
          Aether is private by default. Only allowlisted members can enter.
        </p>
      </section>
    </main>
  );
}

function ConnectionFailure({ message }: { message: string }) {
  return (
    <main className="vault-login">
      <section className="vault-login__sheet">
        <Brand />
        <h1>Vault unavailable.</h1>
        <p>{message}</p>
        <button className="vault-button vault-button--primary" type="button" onClick={() => window.location.reload()}>
          Try again
        </button>
      </section>
    </main>
  );
}

/** Renders the Archive Worktable vault with responsive navigation, catalog browsing, and inspector flows. */
export default function VaultApp() {
  const [session, setSession] = useState<Session | null>(null);
  const [sessionState, setSessionState] = useState<
    "loading" | "ready" | "signed-out" | "disabled" | "error"
  >("loading");
  const [route, setRoute] = useState<RouteName>(() =>
    getRoute(window.location.pathname),
  );
  const [documents, setDocuments] = useState<DocumentItem[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(() =>
    getDocumentIdFromPath(window.location.pathname),
  );
  const [query, setQuery] = useState("");
  const [navOpen, setNavOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadFiles, setUploadFiles] = useState<File[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const loadDocuments = useCallback(async () => {
    setLoading(true);
    try {
      const result = await api.listDocuments();
      setDocuments(result.data.documents);
      setError("");
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    const onPopState = () => {
      setRoute(getRoute(window.location.pathname));
      setSelectedId(getDocumentIdFromPath(window.location.pathname));
      setNavOpen(false);
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);
  const refreshMembers = useCallback(async () => {
    if (session?.member.role !== "admin") return;
    try {
      const result = await api.listMembers();
      setMembers(result.data.members);
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    }
  }, [session]);
  useEffect(() => {
    void api
      .getSession()
      .then((next) => {
        setSession(next.data);
        setSessionState("ready");
      })
      .catch((requestError: unknown) => {
        const status =
          requestError &&
          typeof requestError === "object" &&
          "status" in requestError &&
          typeof requestError.status === "number"
            ? requestError.status
            : -1;
        if (status === 403) setSessionState("disabled");
        else if (status === 401) setSessionState("signed-out");
        else {
          setError(getErrorMessage(requestError));
          setSessionState("error");
        }
      });
  }, []);
  useEffect(() => {
    if (sessionState !== "ready") return;
    const refreshTags = async () => {
      try {
        const result = await api.listTags("", 100);
        setTags(result.data.tags);
      } catch (requestError) {
        setError(getErrorMessage(requestError));
      }
    };
    void refreshTags();
    void refreshMembers();
    void loadDocuments();
  }, [loadDocuments, refreshMembers, sessionState]);
  const selected =
    documents.find((item) => item.document.id === selectedId) ?? null;
  const updateMetadata = async (title: string, nextTags: string[]) => {
    if (!selected) return;
    try {
      const result = await api.updateMetadata(selected.document.id, {
        title,
        tags: nextTags,
        version: selected.document.version,
      });
      const updated: DocumentItem = result.data;
      setDocuments((current) =>
        current.map((item) =>
          item.document.id === selected.document.id ? updated : item,
        ),
      );
    } catch (requestError) {
      throw new Error(getErrorMessage(requestError));
    }
  };
  const deleteSelected = async () => {
    if (!selected) return;
    await api.deleteDocument(selected.document.id);
    setDocuments((current) =>
      current.map((item) =>
        item.document.id === selected.document.id
          ? { ...item, document: { ...item.document, status: "deleted" } }
          : item,
      ),
    );
    setSelectedId(null);
  };
  const restoreDocument = async (id: string) => {
    try {
      await api.restoreDocument(id);
      setDocuments((current) =>
        current.map((item) =>
          item.document.id === id
            ? { ...item, document: { ...item.document, status: "ready" } }
            : item,
        ),
      );
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    }
  };
  const purgeDocument = async (id: string) => {
    try {
      await api.purgeDocument(id);
      setDocuments((current) =>
        current.filter((item) => item.document.id !== id),
      );
    } catch (requestError) {
      setError(getErrorMessage(requestError));
    }
  };
  const logout = async () => {
    try {
      await api.logout();
    } finally {
      setSession(null);
      setSessionState("signed-out");
    }
  };
  const completeUpload = useCallback(
    (item: DocumentItem, finished: boolean) => {
      setDocuments((current) => [item, ...current]);
      if (finished) {
        setUploadFiles([]);
        setUploadOpen(false);
      }
    },
    [],
  );
  const openDocument = (id: string) => {
    setSelectedId(id);
    navigate(`/documents/${encodeURIComponent(id)}`);
  };
  if (sessionState === "loading")
    return (
      <main className="vault-login">
        <section className="vault-login__sheet">
          <Brand />
          <p>Opening your archive…</p>
        </section>
      </main>
    );
  if (sessionState === "signed-out") return <SignedOut />;
  if (sessionState === "disabled") return <SignedOut disabled />;
  if (sessionState === "error") {
    return <ConnectionFailure message={error || "The server could not be reached."} />;
  }
  if (!session) return <SignedOut />;
  const isWorkspace = route === "library" || route === "recent";
  return (
    <div className="vault-app">
      <TopBar
        session={session}
        onMenu={() => setNavOpen(true)}
        onSearch={setQuery}
        onUpload={() => setUploadOpen(true)}
        onLogout={() => void logout()}
      />
      <div className="vault-layout">
        <Navigation
          session={session}
          documents={documents}
          tags={tags}
          open={navOpen}
          onClose={() => setNavOpen(false)}
        />
        <main className="vault-main">
          {error && (
            <div className="vault-notice" role="alert">
              <CircleAlert size={14} aria-hidden="true" /> {error}
              <button
                className="vault-button vault-button--icon"
                type="button"
                aria-label="Dismiss error"
                onClick={() => setError("")}
              >
                <X size={14} aria-hidden="true" />
              </button>
            </div>
          )}
          {isWorkspace && (
            <LibraryWorkspace
              route={route}
              documents={documents.filter(
                (item) => item.document.status !== "deleted",
              )}
              availableTags={tags}
              selectedId={selectedId}
              onSelect={openDocument}
              query={query}
              onUpload={() => setUploadOpen(true)}
              onRefresh={() => void loadDocuments()}
              loading={loading}
            />
          )}
          {route === "tags" && (
            <TagsPage
              tags={tags}
              documents={documents}
              onOpenTag={(tag) => {
                setQuery(tag.displayName);
                navigate("/library");
              }}
              onCreateTag={async (name) => {
                const created = await api.createTag(name);
                setTags((current) => [...current, created.data]);
              }}
            />
          )}
          {route === "trash" && (
            <TrashPage
              documents={documents}
              onRestore={restoreDocument}
              onPurge={purgeDocument}
            />
          )}
          {route === "members" && session.member.role === "admin" && (
            <MembersPage
              members={members}
              onAdd={async (email, role) => {
                const created = await api.createMember(email, role);
                setMembers((current) => [...current, created.data]);
              }}
              onUpdate={async (member) => {
                const updated = await api.updateMember(
                  member.id,
                  member.role,
                  member.status,
                );
                setMembers((current) =>
                  current.map((entry) =>
                    entry.id === updated.data.id ? updated.data : entry,
                  ),
                );
              }}
            />
          )}
        </main>
      </div>
      {selected && isWorkspace && (
        <Inspector
          key={selected.document.id}
          item={selected}
          onClose={() => {
            setSelectedId(null);
            navigate("/library");
          }}
          onUpdate={updateMetadata}
          onDelete={deleteSelected}
          onDownload={() => {
            window.location.href = api.getDocumentContentUrl(
              selected.document.id,
              { download: true },
            );
          }}
        />
      )}
      {uploadOpen && uploadFiles.length === 0 && (
        <UploadDialog
          onClose={() => setUploadOpen(false)}
          onUpload={setUploadFiles}
        />
      )}
      {uploadFiles.length > 0 && (
        <UploadQueue
          files={uploadFiles}
          onClose={() => setUploadFiles([])}
          onComplete={completeUpload}
        />
      )}
    </div>
  );
}
