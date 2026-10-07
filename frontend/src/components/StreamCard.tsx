import { memo, useRef, useState } from 'react';
import { Camera, Globe2 } from 'lucide-react';
import { toast } from 'sonner';
import type { PlayerStatus, Stream } from '../types/stream';
import { maskRTSPURL } from '../utils/helpers';
import { StreamPlayer } from './StreamPlayer';
import { StreamControls } from './StreamControls';

interface Props {
  stream: Stream;
  busy: boolean;
  accessRevision: number;
  onOperate(id: string, action: 'remove' | 'start' | 'stop' | 'restart'): Promise<boolean>;
}
const labels: Record<PlayerStatus, string> = {
  idle: 'Idle',
  connecting: 'Connecting',
  live: 'Live',
  paused: 'Paused',
  stopped: 'Stopped',
  error: 'Error',
  reconnecting: 'Reconnecting',
  offline: 'Offline',
};

export const StreamCard = memo(function StreamCard({
  stream,
  busy,
  accessRevision,
  onOperate,
}: Props) {
  const container = useRef<HTMLElement>(null);
  const [paused, setPaused] = useState(false);
  const [revision, setRevision] = useState(0);
  const [status, setStatus] = useState<PlayerStatus>('connecting');
  const stopped = stream.status === 'stopped' || stream.status === 'error';
  async function restart() {
    if (await onOperate(stream.id, 'restart')) {
      setPaused(false);
      setRevision((v) => v + 1);
    }
  }
  async function play() {
    if (stopped) {
      if (await onOperate(stream.id, 'start')) {
        setPaused(false);
        setRevision((v) => v + 1);
      }
    } else {
      setPaused((v) => !v);
    }
  }
  async function fullscreen() {
    try {
      if (document.fullscreenElement === container.current) await document.exitFullscreen();
      else await container.current?.requestFullscreen();
    } catch {
      toast.error('Fullscreen is unavailable in this browser.');
    }
  }

  return (
    <article ref={container} className="stream-card" aria-label={`${stream.name} stream`}>
      <div className="card-heading">
        <div className="camera-name">
          <Camera size={16} strokeWidth={1.5} />
          <h3 title={stream.name}>{stream.name}</h3>
        </div>
        <span className={`stream-status ${status}`} aria-live="polite">
          <i className="status-dot" />
          {labels[status]}
        </span>
      </div>
      <StreamPlayer
        stream={stream}
        paused={paused}
        revision={revision + accessRevision}
        busy={busy}
        onRetry={() => {
          void restart();
        }}
        onStatus={setStatus}
      />
      <div className="card-info">
        <span className="hostname" title={maskRTSPURL(stream.rtspUrl)}>
          <Globe2 size={12} />
          {stream.hostname}
        </span>
        <span>
          {stream.viewers} {stream.viewers === 1 ? 'viewer' : 'viewers'}
        </span>
      </div>
      <StreamControls
        name={stream.name}
        paused={paused}
        stopped={stopped}
        busy={busy}
        onPlay={() => {
          void play();
        }}
        onRestart={() => {
          void restart();
        }}
        onRemove={() => {
          void onOperate(stream.id, 'remove');
        }}
        onStop={() => {
          void onOperate(stream.id, 'stop');
        }}
        onFullscreen={() => {
          void fullscreen();
        }}
      />
    </article>
  );
});
