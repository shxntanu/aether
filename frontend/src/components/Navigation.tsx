import {
  Archive,
  Clock3,
  Shield,
  Tags,
  Trash2,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";

import type { Session, StorageUsage, Tag } from "@/lib/api";
import {
  getRoute,
  navigate,
  type DocumentItem,
  type RouteName,
} from "@/lib/vault";

type NavItemProps = {
  href: string;
  route: RouteName;
  icon: typeof Archive;
  label: string;
  count?: string;
  onNavigate?: () => void;
};

function formatStorageBytes(bytes: number): string {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = Math.max(0, bytes);
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const precision = value >= 100 || unit === 0 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(precision)} ${units[unit]}`;
}

function NavItem({
  href,
  route,
  icon: Icon,
  label,
  count,
  onNavigate,
}: NavItemProps) {
  return (
    <a
      href={href}
      aria-current={
        getRoute(window.location.pathname) === route ? "page" : undefined
      }
      onClick={(event) => {
        event.preventDefault();
        navigate(href);
        onNavigate?.();
      }}
    >
      <Icon aria-hidden="true" />
      <span>{label}</span>
      {count && <span className="vault-nav__count">{count}</span>}
    </a>
  );
}

/** Renders the responsive primary and administrative navigation rail. */
export function Navigation({
  session,
  documents,
  tags,
  storageUsage,
  storageUsageState,
  open,
  onClose,
}: {
  session: Session;
  documents: DocumentItem[];
  tags: Tag[];
  storageUsage: StorageUsage | null;
  storageUsageState: "loading" | "ready" | "unavailable";
  open: boolean;
  onClose: () => void;
}) {
  const isAdmin = session.member.role === "admin";
  const [isMobile, setIsMobile] = useState(false);
  const railRef = useRef<HTMLElement>(null);
  useEffect(() => {
    const media = window.matchMedia("(max-width: 780px)");
    const update = () => setIsMobile(media.matches);
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  useEffect(() => {
    if (!isMobile || !open) return;
    const rail = railRef.current;
    const trigger = document.querySelector<HTMLButtonElement>(
      ".vault-mobile-menu",
    );
    const background = [
      document.querySelector<HTMLElement>(".vault-topbar"),
      document.querySelector<HTMLElement>(".vault-main"),
      document.querySelector<HTMLElement>(".vault-upload-queue"),
    ].filter((element): element is HTMLElement => element !== null);
    const focusable = rail?.querySelectorAll<HTMLElement>("a[href], button");
    background.forEach((element) => {
      element.inert = true;
    });
    focusable?.[0]?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key !== "Tab" || !focusable || focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      background.forEach((element) => {
        element.inert = false;
      });
      trigger?.focus();
    };
  }, [isMobile, onClose, open]);
  return (
    <>
      {open && isMobile && (
        <button
          className="vault-overlay"
          aria-label="Close navigation"
          type="button"
          onClick={onClose}
        />
      )}
      <aside
        ref={railRef}
        id="vault-navigation"
        className={`vault-rail ${open ? "is-open" : ""}`}
        aria-label="Archive navigation"
        aria-hidden={isMobile && !open ? true : undefined}
        inert={isMobile && !open ? true : undefined}
      >
        <div className="vault-rail__title" aria-label="Celestial Archive Garden">
          <span>Celestial</span>
          <strong>Archive Garden</strong>
        </div>
        <nav className="vault-nav" aria-label="Primary navigation">
          <NavItem
            href="/library"
            route="library"
            icon={Archive}
            label="Library"
            onNavigate={onClose}
            count={String(
              documents.filter((item) => item.document.status !== "deleted")
                .length,
            )}
          />
          <NavItem
            href="/library/recent"
            route="recent"
            icon={Clock3}
            label="Recent"
            onNavigate={onClose}
          />
          <NavItem
            href="/tags"
            route="tags"
            icon={Tags}
            label="Tags"
            count={String(tags.length)}
            onNavigate={onClose}
          />
          <NavItem
            href="/trash"
            route="trash"
            icon={Trash2}
            label="Trash"
            onNavigate={onClose}
            count={String(
              documents.filter((item) => item.document.status === "deleted")
                .length,
            )}
          />
        </nav>
        {isAdmin && (
          <>
            <div className="vault-rail__rule" />
            <span className="vault-rail__label">Administration</span>
            <nav className="vault-nav" aria-label="Administration navigation">
              <NavItem
                href="/admin/members"
                route="members"
                icon={Shield}
                label="Members"
                onNavigate={onClose}
              />
            </nav>
          </>
        )}
        <div className="vault-storage-card" aria-live="polite">
          <strong>Google Drive storage</strong>
          {storageUsageState === "loading" && (
            <p className="vault-storage-card__status">Reading host account…</p>
          )}
          {storageUsageState === "unavailable" && (
            <p className="vault-storage-card__status">
              Storage usage is unavailable.
            </p>
          )}
          {storageUsageState === "ready" && storageUsage && (
            <>
              {storageUsage.limitBytes === null ? (
                <p className="vault-storage-card__summary">
                  {formatStorageBytes(storageUsage.usedBytes)} used · No
                  storage limit
                </p>
              ) : (
                <>
                  <p className="vault-storage-card__summary">
                    {formatStorageBytes(storageUsage.usedBytes)} of{" "}
                    {formatStorageBytes(storageUsage.limitBytes)} used
                  </p>
                  <div
                    className="vault-storage-card__track"
                    role="progressbar"
                    aria-label="Host Google Drive storage used"
                    aria-valuemin={0}
                    aria-valuemax={storageUsage.limitBytes}
                    aria-valuenow={Math.min(
                      storageUsage.usedBytes,
                      storageUsage.limitBytes,
                    )}
                    aria-valuetext={`${formatStorageBytes(
                      storageUsage.usedBytes,
                    )} of ${formatStorageBytes(storageUsage.limitBytes)} used`}
                  >
                    <span
                      style={{
                        width: `${Math.min(
                          100,
                          (storageUsage.usedBytes /
                            storageUsage.limitBytes) *
                            100,
                        )}%`,
                      }}
                    />
                  </div>
                  <span className="vault-storage-card__remaining">
                    {formatStorageBytes(storageUsage.remainingBytes ?? 0)}{" "}
                    remaining
                  </span>
                </>
              )}
            </>
          )}
        </div>
      </aside>
    </>
  );
}
