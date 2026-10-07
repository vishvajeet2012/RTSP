import { useEffect, useRef, useState } from 'react';
import { Pause, Radio } from 'lucide-react';
import { toast } from 'sonner';
import { useWebSocket } from '../hooks/useWebSocket';
import { PushSource } from '../services/websocket';
import type { JSMpegPlayer } from '../types/jsmpeg';
import type { PlayerStatus, Stream } from '../types/stream';
import { LoadingState } from './LoadingState';
import { ErrorState } from './ErrorState';

interface Props {
  stream: Stream;
  paused: boolean;
  revision: number;
  busy: boolean;
  onRetry(): void;
  onStatus(status: PlayerStatus): void;
}

export function StreamPlayer({ stream, paused, revision, busy, onRetry, onStatus }: Props) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const player = useRef<JSMpegPlayer | null>(null);
  const [hasFrame, setHasFrame] = useState(false);
  const [playerError, setPlayerError] = useState<string | null>(null);
  const rendered = useRef(false);
  const firstConnection = useRef(true);
  const active = ['connecting', 'live', 'reconnecting'].includes(stream.status);

  const socket = useWebSocket({
    id: stream.id,
    enabled: active && !paused,
    revision,
    onOpen: () => {
      player.current?.destroy();
      player.current = null;
      rendered.current = false;
      setHasFrame(false);
      setPlayerError(null);
      if (!window.JSMpeg || !canvas.current) {
        setPlayerError('The video decoder could not load. Reload this page.');
        return;
      }
      try {
        player.current = new window.JSMpeg.Player('live', {
          source: PushSource,
          canvas: canvas.current,
          audio: false,
          autoplay: true,
          disableGl: true,
          disableWebAssembly: true,
          pauseWhenHidden: false,
          videoBufferSize: 512 * 1024,
          onVideoDecode: () => {
            if (!rendered.current) {
              rendered.current = true;
              setHasFrame(true);
              if (firstConnection.current) {
                firstConnection.current = false;
                toast.success(`${stream.name} connected`);
              }
            }
          },
        });
      } catch {
        setPlayerError('Unable to initialize the video decoder. Reload this page.');
      }
    },
    onData: (data) => {
      player.current?.source.write(data);
    },
    onDisconnect: () => {
      rendered.current = false;
      setHasFrame(false);
      player.current?.pause();
    },
  });

  useEffect(() => {
    if (paused || !active) player.current?.pause();
  }, [paused, active]);
  useEffect(() => {
    if (socket.state !== 'connected' || paused || !active) return;
    const timer = setTimeout(() => {
      if (!rendered.current) {
        player.current?.pause();
        setPlayerError('Video data arrived but could not be decoded. Try restarting the source.');
      }
    }, 20_000);
    return () => clearTimeout(timer);
  }, [socket.state, paused, active, revision]);
  useEffect(
    () => () => {
      player.current?.destroy();
      player.current = null;
    },
    [],
  );

  const status: PlayerStatus =
    stream.status === 'stopped'
      ? 'stopped'
      : stream.status === 'error'
        ? 'error'
        : paused
          ? 'paused'
          : playerError || socket.state === 'error'
            ? 'error'
            : socket.state === 'reconnecting' || stream.status === 'reconnecting'
              ? 'reconnecting'
              : hasFrame && socket.state === 'connected'
                ? 'live'
                : 'connecting';
  useEffect(() => {
    onStatus(status);
  }, [status, onStatus]);

  return (
    <div className={`player-area ${hasFrame ? 'has-frame' : ''}`} data-player-status={status}>
      <canvas ref={canvas} aria-label={`Live video from ${stream.name}`} />
      {status === 'connecting' || status === 'reconnecting' ? (
        <LoadingState reconnecting={status === 'reconnecting'} />
      ) : null}
      {status === 'error' ? (
        <ErrorState
          message={stream.lastError || playerError || socket.error || undefined}
          onRetry={onRetry}
          busy={busy}
        />
      ) : null}
      {status === 'paused' ? (
        <div className="player-overlay paused-state">
          <Pause size={28} strokeWidth={1.5} />
          <strong>Playback paused</strong>
          <span>Resume to return to the live feed</span>
        </div>
      ) : null}
      {status === 'stopped' ? (
        <div className="player-overlay paused-state">
          <Radio size={27} strokeWidth={1.5} />
          <strong>Source stopped</strong>
          <span>Press Play to start this source again</span>
        </div>
      ) : null}
      {status === 'live' ? (
        <span className="live-watermark">
          <i />
          LIVE
        </span>
      ) : null}
    </div>
  );
}
