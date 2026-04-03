import type { Page } from '@playwright/test';
import { applySessionCookie, expect, openLiveApp, test } from './live';
import { authenticate } from './helpers';

async function enableDialogueMode(page: Page) {
  await page.evaluate(() => {
    const circle = document.getElementById('slopshell-circle-dot');
    if (circle instanceof HTMLButtonElement) {
      circle.click();
    }
  });
  await expect(page.locator('#slopshell-circle')).toHaveAttribute('data-state', 'expanded');
  await page.evaluate(() => {
    const button = document.querySelector('#slopshell-circle-menu .slopshell-circle-segment[data-segment="dialogue"]');
    if (!(button instanceof HTMLButtonElement)) {
      throw new Error('dialogue circle segment not found');
    }
    button.click();
  });
}

test.describe('dialogue mode diagnostics @local-only', () => {
  let sessionToken: string;

  test.beforeAll(async () => {
    sessionToken = await authenticate();
  });

  test('dialogue mode connects turn intelligence and exposes diagnostics', async ({ page }) => {
    await applySessionCookie(page, sessionToken);
    await openLiveApp(page, sessionToken);

    await page.evaluate(() => {
      const app = (window as any)._slopshellApp;
      app?.clearDialogueDiagnostics?.();
    });
    await enableDialogueMode(page);

    await expect(page.locator('#edge-top-models .edge-live-status')).toContainText('Dialogue', { timeout: 8_000 });

    await expect.poll(async () => {
      return page.evaluate(() => {
        const app = (window as any)._slopshellApp;
        const state = app?.getState?.();
        return Boolean(state?.liveSessionActive && state?.liveSessionMode === 'dialogue');
      });
    }, { timeout: 10_000 }).toBe(true);

    await expect.poll(async () => {
      return page.evaluate(() => {
        const diagnostics = (window as any)._slopshellApp?.getDialogueDiagnostics?.();
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

    const diagnostics = await page.evaluate(() => (window as any)._slopshellApp?.getDialogueDiagnostics?.());
    expect(Array.isArray(diagnostics?.recentEvents)).toBe(true);
    expect(diagnostics.recentEvents.some((entry: any) => entry?.kind === 'turn_metrics')).toBe(true);
  });
});
