import {
  Archive,
  Clock3,
  LogOut,
  Shield,
  Tags,
  Trash2,
} from "lucide-react";

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
};

function NavItem({ href, route, icon: Icon, label, count }: NavItemProps) {
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

type AccountMenuProps = {
  onLogout: () => void;
  onClose: () => void;
};

/** Renders the account action menu and closes it before signing out. */
export function AccountMenu({ onLogout, onClose }: AccountMenuProps) {
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
