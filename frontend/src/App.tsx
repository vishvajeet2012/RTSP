import { useState } from 'react';
import { ArrowUpRight, CircleAlert, LayoutGrid, Radio, RotateCw, ShieldCheck } from 'lucide-react';
import { Toaster, toast } from 'sonner';
import { Header } from './components/Header';
import { AccessDialog } from './components/AccessDialog';
import { AddStreamForm } from './components/AddStreamForm';
import { EmptyState } from './components/EmptyState';
import { StreamGrid } from './components/StreamGrid';
import { useStreams } from './hooks/useStreams';
import { setAccessToken } from './services/api';

export default function App() {
  const [accessOpen, setAccessOpen] = useState(false);
  const [accessRevision, setAccessRevision] = useState(0);
  const store = useStreams(accessRevision);
  const connected = store.streams.filter((s) => s.status === 'live').length;
  return (
    <>
      <Header backend={store.backend} onAccess={() => setAccessOpen(true)} />
      <main className="shell main-content">
        <section className="hero">
          <div>
            <span className="eyebrow">
              <span />
              LIVE OPERATIONS
            </span>
            <h1>
              Every stream.
              <br className="mobile-break" /> One workspace.
            </h1>
            <p>Monitor multiple RTSP camera streams directly from your browser.</p>
          </div>
          <div className="hero-label">
            <Radio size={16} strokeWidth={1.5} />
            <span>
              Real-time video
              <br />
              <small>Connected. In view.</small>
            </span>
          </div>
        </section>
        {store.error || store.backend === 'degraded' ? (
          <div className="notice" role="alert">
            <CircleAlert size={17} />
            <span>
              {store.backend === 'degraded'
                ? 'FFmpeg is missing on the backend. Install it or update FFMPEG_PATH.'
                : store.error}
            </span>
            <button
              className="button secondary compact"
              onClick={store.authRequired ? () => setAccessOpen(true) : store.refresh}
            >
              {store.authRequired ? (
                'Enter token'
              ) : (
                <>
                  <RotateCw size={13} />
                  Retry
                </>
              )}
            </button>
          </div>
        ) : null}
        <AddStreamForm
          onAdd={store.add}
          busy={store.adding}
          disabled={store.authRequired || store.backend !== 'connected'}
        />
        <section className="streams-section" aria-labelledby="streams-title">
          <div className="streams-heading">
            <div>
              <LayoutGrid size={17} strokeWidth={1.5} />
              <h2 id="streams-title">Active streams</h2>
              <span className="count-badge">{store.streams.length}</span>
            </div>
            <span className="streams-summary">
              <i className={`status-dot ${connected ? 'connected' : ''}`} />
              {connected} connected
            </span>
          </div>
          {store.loading ? (
            <div className="workspace-loading" role="status">
              <span className="skeleton-bar" />
              Loading workspace…
            </div>
          ) : store.streams.length ? (
            <StreamGrid
              streams={store.streams}
              busy={store.busy}
              accessRevision={accessRevision}
              onOperate={store.operate}
            />
          ) : (
            <EmptyState />
          )}
        </section>
        <footer className="footer">
          <span>
            <ShieldCheck size={13} />
            Credentials stay on the backend
          </span>
          <a href="https://github.com/phoboslab/jsmpeg" target="_blank" rel="noreferrer">
            Powered by Go, FFmpeg &amp; JSMpeg
            <ArrowUpRight size={12} />
          </a>
        </footer>
      </main>
      <AccessDialog
        open={accessOpen}
        onClose={() => setAccessOpen(false)}
        onSave={(token) => {
          setAccessToken(token);
          setAccessRevision((v) => v + 1);
          toast('Connecting to workspace…');
        }}
      />
      <Toaster
        theme="dark"
        position="bottom-right"
        richColors
        closeButton
        toastOptions={{
          style: { background: '#111111', border: '1px solid #242424', color: '#f5f5f5' },
        }}
      />
    </>
  );
}
