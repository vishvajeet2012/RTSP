import { describe, expect, it } from 'vitest';
import { maskRTSPURL, validateRTSPURL, websocketBase } from './helpers';
import { reconnectDelay } from '../services/websocket';

describe('stream URL handling', () => {
  it('accepts RTSP/RTSPS cameras and rejects other schemes and invalid ports', () => {
    expect(validateRTSPURL('rtsp://admin:p%40ss@localhost:8554/test')).toBeNull();
    expect(validateRTSPURL('rtsps://camera.example/live')).toBeNull();
    for (const url of [
      'https://example.com',
      'rtsp:///live',
      'rtsp://camera:99999/live',
      'rtsp://camera:0/live',
      'rtsp://camera/live#frag',
      'rtsp://camera/ space',
    ])
      expect(validateRTSPURL(url)).not.toBeNull();
  });
  it('redacts userinfo and query tokens without losing the public address', () => {
    const masked = maskRTSPURL('rtsp://admin:secret@camera:554/live?token=private');
    expect(masked).not.toContain('admin');
    expect(masked).not.toContain('secret');
    expect(masked).not.toContain('private');
    expect(masked).toContain('camera:554/live');
  });
  it('derives WSS from HTTPS and preserves configured base paths', () => {
    expect(websocketBase('https://backend.example')).toBe('wss://backend.example');
    expect(websocketBase('http://localhost:8080')).toBe('ws://localhost:8080');
    expect(websocketBase('https://api.example', 'wss://socket.example/prefix/')).toBe(
      'wss://socket.example/prefix',
    );
    expect(() => websocketBase('https://example.com', 'https://wrong.example')).toThrow();
  });
  it('caps reconnect backoff', () => {
    expect([1, 2, 3, 4, 5, 8].map(reconnectDelay)).toEqual([1000, 2000, 4000, 8000, 15000, 15000]);
  });
});
