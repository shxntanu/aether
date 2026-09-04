import { CircleAlert, Upload, X } from "lucide-react";
import { useEffect, useRef, useState, type ChangeEvent } from "react";

import { supportedUploadTypes } from "@/lib/vault";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

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
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => setOpen(true));
    return () => window.cancelAnimationFrame(frame);
  }, []);
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
    <Dialog
      open={open}
      onOpenChange={setOpen}
      onOpenChangeComplete={(isOpen) => {
        if (!isOpen) onClose();
      }}
    >
      <DialogContent
        showCloseButton={false}
        className="vault-upload-dialog vault-login__sheet"
      >
        <DialogHeader className="vault-upload-dialog__header">
          <DialogTitle id="upload-dialog-title">
            Add to the archive
          </DialogTitle>
          <Button
            variant="ghost"
            size="icon"
            className="vault-button vault-button--icon"
            type="button"
            aria-label="Close upload dialog"
            onClick={() => setOpen(false)}
          >
            <X size={17} aria-hidden="true" />
          </Button>
        </DialogHeader>
        <DialogDescription className="vault-upload-dialog__description">
          Original files are preserved exactly as uploaded. PDF, JPEG, PNG, and
          WebP files up to 50 MB are supported.
        </DialogDescription>
        <Button
          variant="default"
          className="vault-button vault-button--primary"
          type="button"
          onClick={() => inputRef.current?.click()}
        >
          <Upload size={15} aria-hidden="true" /> Choose files
        </Button>
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
          className={`vault-upload-dropzone ${dragging ? "is-dragging" : ""}`}
        >
          Drop files here, or click to browse
        </button>
        {validationError && (
          <div className="vault-notice" role="alert">
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
      </DialogContent>
    </Dialog>
  );
}
