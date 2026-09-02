import { Plus, UserRound } from "lucide-react";
import { useState, type FormEvent } from "react";

import type { Member } from "@/lib/api";
import { getErrorMessage, initials } from "@/lib/vault";

import { Status } from "@/components/Status";

/** Renders the administrator-only member allowlist and access controls. */
export function MembersPage({
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
