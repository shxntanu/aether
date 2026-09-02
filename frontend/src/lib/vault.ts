import type { DocumentRecord } from "@/lib/api";

/** Names of the primary vault routes rendered by the application shell. */
export type RouteName = "library" | "recent" | "tags" | "trash" | "members";

/** Available presentations for the document collection. */
export type ViewMode = "list" | "grid";

/** Controls how a selected tag is matched against a document. */
export type TagMatch = "all" | "any";

/** Alias for a catalog document together with its attached reusable tags. */
export type DocumentItem = DocumentRecord;

/** Returns a compact uppercase avatar label from a display name. */
export function initials(name: string): string {
  return name
    .split(" ")
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

/** Formats byte counts using the compact units shown in the document list. */
export function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(bytes > 5 * 1024 * 1024 ? 1 : 2)} MB`;
}

/** Formats server timestamps using the vault's locale-specific date display. */
export function formatDate(value: string): string {
  return new Intl.DateTimeFormat("en-IN", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(new Date(value));
}

/** Categorizes a media type for the matching icon and preview treatment. */
export function fileKind(mediaType: string): "pdf" | "image" | "file" {
  if (mediaType === "application/pdf") return "pdf";
  if (mediaType.startsWith("image/")) return "image";
  return "file";
}

/** Resolves a browser pathname to the route understood by the vault shell. */
export function getRoute(pathname: string): RouteName {
  if (pathname === "/library/recent") return "recent";
  if (pathname.startsWith("/tags")) return "tags";
  if (pathname.startsWith("/trash")) return "trash";
  if (pathname.startsWith("/admin/members")) return "members";
  return "library";
}

/** Extracts a document id from the inspector pathname, if one is present. */
export function getDocumentIdFromPath(pathname: string): string | null {
  const match = pathname.match(/^\/documents\/([^/]+)$/);
  return match ? decodeURIComponent(match[1]) : null;
}

/** Pushes an internal pathname and notifies the app's lightweight router. */
export function navigate(pathname: string): void {
  window.history.pushState({}, "", pathname);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

/** Converts unknown request failures into a message suitable for the UI. */
export function getErrorMessage(error: unknown): string {
  if (
    error &&
    typeof error === "object" &&
    "message" in error &&
    typeof error.message === "string"
  )
    return error.message;
  return "The vault could not complete that request.";
}

/** MIME types accepted by the upload dialog and upload queue. */
export const supportedUploadTypes = new Set([
  "application/pdf",
  "image/jpeg",
  "image/png",
  "image/webp",
]);
