import { LoaderCircle } from 'lucide-react';

export function LoadingState({ reconnecting = false }: { reconnecting?: boolean }) {
  return (
    <div className="player-overlay loading-state" role="status">
      <span className="loading-orbit">
        <LoaderCircle size={23} className="spin" strokeWidth={1.5} />
      </span>
      <strong>{reconnecting ? 'Reconnecting to stream…' : 'Connecting to stream…'}</strong>
      <span>
        {reconnecting
          ? 'Waiting for the camera connection to recover'
          : 'Establishing a live video connection'}
      </span>
    </div>
  );
}
