export interface BinaryDestination {
  write(data: ArrayBuffer): void;
}

/** Bridge to JSMpeg's TS demuxer; socket ownership stays in our React hook. */
export class PushSource {
  readonly streaming = true;
  completed = false;
  established = false;
  progress = 0;
  private destination: BinaryDestination | null = null;

  connect(destination: BinaryDestination): void {
    this.destination = destination;
  }
  start(): void {
    this.progress = 1;
  }
  resume(): void {
    /* Live packets are pushed directly, with no replay buffer. */
  }
  destroy(): void {
    this.destination = null;
    this.completed = true;
  }
  write(data: ArrayBuffer): void {
    this.established = true;
    this.destination?.write(data);
  }
}

export function reconnectDelay(attempt: number): number {
  return Math.min(1000 * 2 ** Math.max(0, attempt - 1), 15_000);
}
