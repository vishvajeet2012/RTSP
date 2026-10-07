import { useState, type FormEvent } from 'react';
import { ArrowUpRight, Eye, EyeOff, Link2, LoaderCircle, Plus } from 'lucide-react';
import type { AddStreamInput } from '../types/stream';
import { validateRTSPURL } from '../utils/helpers';

export function AddStreamForm({
  onAdd,
  busy,
  disabled,
}: {
  onAdd(input: AddStreamInput): Promise<boolean>;
  busy: boolean;
  disabled: boolean;
}) {
  const [name, setName] = useState('');
  const [url, setURL] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [visible, setVisible] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    const message = validateRTSPURL(url.trim());
    setError(message);
    if (message) return;
    if (await onAdd({ name: name.trim(), rtspUrl: url.trim() })) {
      setURL('');
      setName('');
      setVisible(false);
    }
  }
  return (
    <section className="add-panel" aria-labelledby="add-title">
      <div className="panel-heading">
        <span className="section-icon">
          <Link2 size={17} />
        </span>
        <h2 id="add-title">Connect a stream</h2>
        <span className="panel-badge">RTSP / RTSPS</span>
      </div>
      <form
        className="stream-form"
        onSubmit={(e) => {
          void submit(e);
        }}
        noValidate
      >
        <div className="field name-field">
          <label htmlFor="stream-name">
            Stream name <span>optional</span>
          </label>
          <input
            id="stream-name"
            placeholder="e.g. Office camera"
            value={name}
            maxLength={80}
            onChange={(e) => setName(e.target.value)}
            disabled={busy}
          />
        </div>
        <div className="field url-field">
          <label htmlFor="stream-url">RTSP URL</label>
          <div className="url-input">
            <input
              id="stream-url"
              type={visible ? 'text' : 'password'}
              placeholder="rtsp://username:password@192.168.1.10:554/stream"
              value={url}
              maxLength={4096}
              autoComplete="off"
              spellCheck={false}
              aria-invalid={Boolean(error)}
              aria-describedby={error ? 'url-error' : 'url-help'}
              onChange={(e) => {
                setURL(e.target.value);
                setError(null);
              }}
              disabled={busy}
            />
            <button
              type="button"
              className="icon-button"
              aria-label={visible ? 'Hide RTSP URL' : 'Show RTSP URL'}
              onClick={() => setVisible((v) => !v)}
            >
              {visible ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>
        </div>
        <button className="button primary add-button" type="submit" disabled={busy || disabled}>
          {busy ? <LoaderCircle size={17} className="spin" /> : <Plus size={17} />}
          <span>{busy ? 'Adding…' : 'Add stream'}</span>
          <ArrowUpRight size={15} />
        </button>
      </form>
      {error ? (
        <p id="url-error" className="form-error" role="alert">
          {error}
        </p>
      ) : (
        <p id="url-help" className="form-hint">
          Credentials are masked after adding. The backend must be able to reach your camera.
        </p>
      )}
    </section>
  );
}
