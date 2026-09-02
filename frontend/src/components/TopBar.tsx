import { ChevronDown, Menu, Search, Upload } from "lucide-react";
import { useState } from "react";

import type { Session } from "@/lib/api";

import { Brand } from "@/components/Brand";
import { AccountMenu } from "@/components/Navigation";
import { initials } from "@/lib/vault";

/** Renders the sticky header with search, upload, navigation, and account actions. */
export function TopBar({
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
