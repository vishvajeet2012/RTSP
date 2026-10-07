import { ClipboardPaste, KeyRound, Radar } from 'lucide-react';
import type { BackendState } from '../types/stream';
import { workspaceToken } from '../services/api';

const labels: Record<BackendState, string> = {
  connected: 'Backend connected',
  offline: 'Backend offline',
  checking: 'Checking backend',
  degraded: 'FFmpeg unavailable',
};
export function Header({
  backend,
  onAccess,
  onPasteToken,
}: {
  backend: BackendState;
  onAccess(): void;
  onPasteToken?(): void;
}) {
  return (
    <header className="topbar">
      <div className="shell nav-inner">
        <a href="/" className="brand" aria-label="RTSP Viewer home">
          <span className="brand-mark">
            <Radar size={23} strokeWidth={1.5} />
          </span>
          <span>
            RTSP <span className="brand-light">Viewer</span>
            <small>LIVE RTSP MONITORING</small>
          </span>
        </a>
        <div className="nav-actions">
          <span className={`backend-status ${backend}`} role="status">
            <i className="status-dot" />
            <span>{labels[backend]}</span>
          </span>
          {workspaceToken && onPasteToken ? (
            <button
              className="button secondary compact"
              onClick={onPasteToken}
              aria-label="Paste workspace access token"
            >
              <ClipboardPaste size={15} />
              <span>Paste token</span>
            </button>
          ) : null}
          <button
            className="button secondary compact"
            onClick={onAccess}
            aria-label="Configure backend access token"
          >
            <KeyRound size={15} />
            <span>Access</span>
          </button>
        </div>
      </div>
    </header>
  );
}
