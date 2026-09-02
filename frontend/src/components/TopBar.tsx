import { LogOut, Menu, Search, Upload } from "lucide-react";
import { useState } from "react";

import type { Session } from "@/lib/api";

import { Brand } from "@/components/Brand";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
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
        <Input
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
        <DropdownMenu open={accountOpen} onOpenChange={setAccountOpen}>
          <DropdownMenuTrigger
            render={<button className="vault-account" type="button" />}
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
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            className="vault-account-menu"
          >
            <DropdownMenuItem onClick={onLogout}>
              <LogOut aria-hidden="true" /> Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
