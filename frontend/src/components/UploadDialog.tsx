import { CircleAlert, Upload, X } from "lucide-react";
import { useRef, useState, type ChangeEvent } from "react";

import { supportedUploadTypes } from "@/lib/vault";

/** Validates and collects files before handing them to the upload queue. */
export function UploadDialog({
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
