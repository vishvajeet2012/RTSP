import { KeyRound, Radar } from 'lucide-react';
import type { BackendState } from '../types/stream';

const labels: Record<BackendState, string> = {
  connected: 'Backend connected',
  offline: 'Backend offline',
  checking: 'Checking backend',
  degraded: 'FFmpeg unavailable',
};
export function Header({ backend, onAccess }: { backend: BackendState; onAccess(): void }) {
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
