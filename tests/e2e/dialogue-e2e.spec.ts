import { applySessionCookie, expect, openLiveApp, test } from './live';
import { authenticate } from './helpers';

test.describe('dialogue mode diagnostics @local-only', () => {
  let sessionToken: string;

  test.beforeAll(async () => {
    sessionToken = await authenticate();
  });

  test('dialogue mode connects turn intelligence and exposes diagnostics', async ({ page }) => {
    await applySessionCookie(page, sessionToken);
    await openLiveApp(page, sessionToken);

    await page.evaluate(() => {
      const app = (window as any)._taburaApp;
      app?.clearDialogueDiagnostics?.();
      const btn = document.querySelector('#edge-top-models .edge-live-dialogue-btn');
      if (btn instanceof HTMLButtonElement) {
        btn.click();
      }
    });

    await expect(page.locator('#edge-top-models .edge-live-status')).toContainText('Dialogue', { timeout: 8_000 });

    await expect.poll(async () => {
      return page.evaluate(() => {
        const app = (window as any)._taburaApp;
        const state = app?.getState?.();
        return Boolean(state?.liveSessionActive && state?.liveSessionMode === 'dialogue');
      });
    }, { timeout: 10_000 }).toBe(true);

    await expect.poll(async () => {
      return page.evaluate(() => {
        const diagnostics = (window as any)._taburaApp?.getDialogueDiagnostics?.();
        if (!diagnostics) return null;
        return {
          connected: diagnostics.connected === true,
          profile: String(diagnostics.profile || '').trim(),
          hasMetrics: Boolean(diagnostics.lastMetrics),
          events: Array.isArray(diagnostics.recentEvents) ? diagnostics.recentEvents.length : 0,
        };
      });
    }, {
      timeout: 15_000,
      intervals: [250, 500, 1000],
      message: 'dialogue mode should expose connected turn diagnostics',
    }).toMatchObject({
      connected: true,
      profile: 'balanced',
      hasMetrics: true,
    });

    const diagnostics = await page.evaluate(() => (window as any)._taburaApp?.getDialogueDiagnostics?.());
    expect(Array.isArray(diagnostics?.recentEvents)).toBe(true);
    expect(diagnostics.recentEvents.some((entry: any) => entry?.kind === 'turn_metrics')).toBe(true);
  });
});
