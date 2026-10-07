import { CircleAlert, RotateCw } from 'lucide-react';

export function ErrorState({
  message,
  onRetry,
  busy,
}: {
  message?: string;
  onRetry(): void;
  busy: boolean;
}) {
  return (
    <div className="player-overlay error-state" role="alert">
      <CircleAlert size={25} strokeWidth={1.5} />
      <strong>Unable to connect</strong>
      <span>
        {message || 'Check the RTSP URL, credentials, network access, and camera availability.'}
      </span>
      <button className="button secondary compact" onClick={onRetry} disabled={busy}>
        <RotateCw size={13} />
        Retry
      </button>
    </div>
  );
}
