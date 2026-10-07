import { ScanLine, ArrowUp } from 'lucide-react';

export function EmptyState() {
  return (
    <div className="empty-state">
      <div className="empty-symbol">
        <ScanLine size={34} strokeWidth={1} />
      </div>
      <h3>No active streams</h3>
      <p>Add an RTSP URL above to start monitoring a live stream.</p>
      <span className="empty-hint">
        <ArrowUp size={13} /> Your workspace starts with one connection
      </span>
    </div>
  );
}
