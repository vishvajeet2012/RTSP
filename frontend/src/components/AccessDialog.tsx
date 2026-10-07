import { useEffect, useRef, useState } from 'react';
import { KeyRound, X } from 'lucide-react';

export function AccessDialog({
  open,
  onClose,
  onSave,
}: {
  open: boolean;
  onClose(): void;
  onSave(token: string): void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [value, setValue] = useState('');
  useEffect(() => {
    if (open) dialog.current?.showModal();
    else dialog.current?.close();
  }, [open]);
  return (
    <dialog
      ref={dialog}
      className="access-dialog"
      onCancel={onClose}
      onClick={(event) => {
        if (event.target === dialog.current) onClose();
      }}
      aria-labelledby="access-title"
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          onSave(value);
          setValue('');
          onClose();
        }}
      >
        <div className="dialog-heading">
          <KeyRound size={22} />
          <button
            type="button"
            className="icon-button"
            onClick={onClose}
            aria-label="Close access settings"
          >
            <X size={18} />
          </button>
        </div>
        <h2 id="access-title">Workspace access</h2>
        <p>
          Enter the access token configured on your backend. It is saved in this browser so you do
          not have to enter it on every visit.
        </p>
        <label htmlFor="token">Backend access token</label>
        <input
          id="token"
          type="password"
          autoComplete="off"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="Enter token, or leave blank for local development"
        />
        <button className="button primary dialog-save" type="submit">
          Connect workspace
        </button>
      </form>
    </dialog>
  );
}
