import type { PushSource } from '../services/websocket';

export interface JSMpegOptions {
  canvas: HTMLCanvasElement;
  source: typeof PushSource;
  audio: boolean;
  autoplay: boolean;
  disableWebAssembly: boolean;
  disableGl: boolean;
  pauseWhenHidden: boolean;
  videoBufferSize: number;
  onVideoDecode: () => void;
}
export interface JSMpegPlayer {
  source: PushSource;
  play(): void;
  pause(): void;
  destroy(): void;
}

declare global {
  interface Window {
    JSMpeg?: { Player: new (url: string, options: JSMpegOptions) => JSMpegPlayer };
  }
}
