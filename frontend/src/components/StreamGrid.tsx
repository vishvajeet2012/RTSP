import type { Stream } from '../types/stream';
import { StreamCard } from './StreamCard';

export function StreamGrid({
  streams,
  busy,
  accessRevision,
  onOperate,
}: {
  streams: Stream[];
  busy: Set<string>;
  accessRevision: number;
  onOperate(id: string, action: 'remove' | 'start' | 'stop' | 'restart'): Promise<boolean>;
}) {
  return (
    <div className={`stream-grid count-${Math.min(streams.length, 5)}`}>
      {streams.map((stream) => (
        <StreamCard
          key={stream.id}
          stream={stream}
          busy={busy.has(stream.id)}
          accessRevision={accessRevision}
          onOperate={onOperate}
        />
      ))}
    </div>
  );
}
