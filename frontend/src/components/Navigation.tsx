import {
  Archive,
  Clock3,
  Shield,
  Tags,
  Trash2,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";

import type { Session, Tag } from "@/lib/api";
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
