export function validateRTSPURL(value: string): string | null {
  const hasControl = [...value].some(
    (char) => char.charCodeAt(0) < 32 || char.charCodeAt(0) === 127,
  );
  if (!value || value.length > 4096 || /\s/u.test(value) || hasControl)
    return 'Enter an RTSP URL without spaces.';
  try {
    const url = new URL(value);
    if (!['rtsp:', 'rtsps:'].includes(url.protocol)) return 'Use an rtsp:// or rtsps:// URL.';
    if (!url.hostname || url.hash) return 'Include a camera hostname and remove URL fragments.';
    if (url.port && (!/^\d+$/.test(url.port) || +url.port < 1 || +url.port > 65535))
      return 'Enter a port between 1 and 65535.';
    return null;
  } catch {
    return 'Enter a valid RTSP URL. Encode special characters in credentials.';
  }
}

export function maskRTSPURL(value: string): string {
  try {
    const url = new URL(value);
    if (url.username || url.password) {
      url.username = '****';
      url.password = '****';
    }
    for (const key of [...url.searchParams.keys()]) url.searchParams.set(key, '****');
    url.hash = '';
    return url.toString();
  } catch {
    return '[invalid URL]';
  }
}

export function websocketBase(apiURL: string, configured?: string): string {
  const base = (configured?.trim() || apiURL.replace(/^http/i, 'ws')).replace(/\/+$/, '');
  const url = new URL(base);
  if (!['ws:', 'wss:'].includes(url.protocol))
    throw new Error('VITE_WS_URL must use ws:// or wss://.');
  return base;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Something went wrong. Please try again.';
}
