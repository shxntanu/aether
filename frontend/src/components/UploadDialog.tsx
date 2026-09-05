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
          <div className="vault-upload-dialog__heading">
            <DialogTitle id="upload-dialog-title">Add documents</DialogTitle>
            <DialogDescription className="vault-upload-dialog__description">
              Preserve your original files exactly as uploaded.
            </DialogDescription>
          </div>
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
        <div className="vault-upload-dialog__body">
          <button
            type="button"
            aria-describedby="upload-dialog-formats"
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
            <span className="vault-upload-dropzone__icon" aria-hidden="true">
              <Upload size={22} strokeWidth={1.8} />
            </span>
            <span className="vault-upload-dropzone__copy">
              <strong>Drop files here</strong>
              <span>or browse from your device</span>
            </span>
            <span className="vault-upload-dropzone__choose">Choose files</span>
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
        </div>
        <div className="vault-upload-dialog__details">
          <p
            className="vault-upload-dialog__formats"
            id="upload-dialog-formats"
          >
            PDF, JPEG, PNG or WebP <span aria-hidden="true">·</span> 50 MB
            maximum
          </p>
          <p className="vault-upload-dialog__note">
            Add titles and reusable tags from the inspector after upload.
          </p>
        </div>
      </DialogContent>
    </Dialog>
  );
}
