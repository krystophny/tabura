import { expect, openLiveApp, test } from './live';
import { authenticate } from './helpers';
import { browserWsTtsToStt, waitForLiveAppReady } from './live-ui';

test.describe('browser TTS/STT roundtrip @local-only', () => {
  let sessionToken: string;

  test.beforeAll(async () => {
    sessionToken = await authenticate();
  });

  test('browser websocket TTS audio round-trips through HTTP STT', async ({ page }) => {
    await openLiveApp(page, sessionToken);
    await waitForLiveAppReady(page);

    const result = await browserWsTtsToStt(page, 'Computer browser roundtrip over HTTP.', 'http');
    expect(result.wavBytes).toBeGreaterThan(44);
    expect(result.status).toBe(200);
    expect(String(result.transcript || '').trim().length).toBeGreaterThan(5);
  });

  test('browser websocket TTS audio round-trips through websocket STT', async ({ page }) => {
    await openLiveApp(page, sessionToken);
    await waitForLiveAppReady(page);

    const result = await browserWsTtsToStt(page, 'Computer browser roundtrip over websocket.', 'ws');
    expect(result.wavBytes).toBeGreaterThan(44);
    expect(result.status).toBe('stt_result');
    expect(String(result.transcript || '').trim().length).toBeGreaterThan(5);
  });
});
