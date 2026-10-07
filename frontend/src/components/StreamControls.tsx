import { LoaderCircle, Maximize2, Pause, Play, RotateCw, Square, Trash2 } from 'lucide-react';

interface Props {
  name: string;
  paused: boolean;
  stopped: boolean;
  busy: boolean;
  onPlay(): void;
  onRestart(): void;
  onRemove(): void;
  onFullscreen(): void;
  onStop(): void;
}
export function StreamControls({
  name,
  paused,
  stopped,
  busy,
  onPlay,
  onRestart,
  onRemove,
  onFullscreen,
  onStop,
}: Props) {
  const play = paused || stopped;
  return (
    <div className="stream-controls">
      <button
        className="button secondary playback-button"
        onClick={onPlay}
        disabled={busy}
        aria-label={`${play ? 'Play' : 'Pause'} ${name}`}
      >
        {play ? <Play size={14} /> : <Pause size={14} />}
        <span>{play ? 'Play' : 'Pause'}</span>
      </button>
      <span className="controls-spacer" />
      <button
        className="icon-button"
        onClick={onStop}
        disabled={busy || stopped}
        aria-label={`Stop source ${name} for all viewers`}
        title="Stop source for all viewers"
      >
        <Square size={14} />
      </button>
      <button
        className="icon-button"
        onClick={onRestart}
        disabled={busy}
        aria-label={`Restart ${name}`}
        title="Restart source for all viewers"
      >
        {busy ? <LoaderCircle size={15} className="spin" /> : <RotateCw size={15} />}
      </button>
      <button
        className="icon-button"
        onClick={onFullscreen}
        aria-label={`Fullscreen ${name}`}
        title="Fullscreen"
      >
        <Maximize2 size={15} />
      </button>
      <span className="controls-divider" />
      <button
        className="icon-button danger"
        onClick={onRemove}
        disabled={busy}
        aria-label={`Remove ${name}`}
        title="Remove stream for all viewers"
      >
        <Trash2 size={15} />
      </button>
    </div>
  );
}
