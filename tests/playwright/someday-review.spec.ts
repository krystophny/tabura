import { expect, test, type Page } from '@playwright/test';

async function waitReady(page: Page) {
  await page.goto('/tests/playwright/harness.html');
  await page.waitForFunction(() => {
    const app = (window as any)._sloppadApp;
    if (typeof app?.getState !== 'function') return false;
    const s = app.getState();
    const wsOpen = (window as any).WebSocket.OPEN;
    return s.chatWs?.readyState === wsOpen && s.canvasWs?.readyState === wsOpen;
  }, null, { timeout: 8_000 });
}

async function injectChatEvent(page: Page, payload: Record<string, unknown>) {
  await page.evaluate((p) => {
    const app = (window as any)._sloppadApp;
    const activeChatWs = app?.getState?.().chatWs;
    if (activeChatWs && typeof activeChatWs.injectEvent === 'function') {
      activeChatWs.injectEvent(p);
      return;
    }
    const sessions = (window as any).__mockWsSessions || [];
    const candidates = sessions.filter((ws: any) => ws.url && ws.url.includes('/ws/chat/'));
    const chatWs = candidates[candidates.length - 1];
    if (chatWs && typeof chatWs.injectEvent === 'function') chatWs.injectEvent(p);
  }, payload);
}

async function openSidebar(page: Page) {
  await page.locator('#edge-left-tap').click();
  await expect(page.locator('#pr-file-pane')).toHaveClass(/is-open/);
}

test.describe('someday review flow', () => {
  test('system action opens the someday view', async ({ page }) => {
    await waitReady(page);

    await injectChatEvent(page, {
      type: 'system_action',
      action: { type: 'show_item_sidebar_view', view: 'someday' },
    });

    await expect(page.locator('#pr-file-pane')).toHaveClass(/is-open/);
    await expect(page.locator('.sidebar-tab.is-active')).toContainText('Someday');
    await expect(page.locator('#pr-file-list')).toContainText('Sketch mobile inbox gestures');
  });

  test('keyboard shortcuts can defer an inbox item to someday and promote it back', async ({ page }) => {
    await waitReady(page);
    await openSidebar(page);

    const inboxRow = page.locator('#pr-file-list .pr-file-item[data-item-id="101"]');
    await inboxRow.click();
    await page.keyboard.press('KeyS');
    await expect(page.locator('#pr-file-list')).not.toContainText('Review parser cleanup');

    await page.locator('.sidebar-tab', { hasText: 'Someday' }).click();
    await expect(page.locator('#pr-file-list')).toContainText('Review parser cleanup');

    const somedayRow = page.locator('#pr-file-list .pr-file-item[data-item-id="101"]');
    await somedayRow.click();
    await page.keyboard.press('KeyA');
    await expect(page.locator('#pr-file-list')).not.toContainText('Review parser cleanup');

    await page.locator('.sidebar-tab', { hasText: 'Inbox' }).click();
    await expect(page.locator('#pr-file-list')).toContainText('Review parser cleanup');

    const log = await page.evaluate(() => (window as any).__harnessLog || []);
    expect(log.some((entry: any) => entry?.action === 'item_triage' && entry?.payload?.action === 'someday')).toBe(true);
    expect(log.some((entry: any) => entry?.action === 'item_state' && entry?.payload?.state === 'inbox')).toBe(true);
  });

  test('weekly someday nudge appears and the disable action persists the preference', async ({ page }) => {
    await waitReady(page);
    await page.evaluate(() => {
      window.localStorage.setItem('sloppad.somedayReviewNudgeEnabled', 'true');
      window.localStorage.removeItem('sloppad.somedayReviewNudgeLastShownAt');
    });

    await openSidebar(page);
    await page.locator('.sidebar-tab', { hasText: 'Waiting' }).click();
    await expect(page.locator('#chat-history .chat-message.chat-system .chat-bubble').last())
      .toContainText('You have 1 item in someday.');

    await injectChatEvent(page, {
      type: 'system_action',
      action: { type: 'set_someday_review_nudge', enabled: false },
    });
    await expect.poll(async () => page.evaluate(() => window.localStorage.getItem('sloppad.somedayReviewNudgeEnabled')))
      .toBe('false');
  });

  test('disabled someday reminders suppress the weekly nudge', async ({ page }) => {
    await page.addInitScript(() => {
      window.localStorage.setItem('sloppad.somedayReviewNudgeEnabled', 'false');
      window.localStorage.removeItem('sloppad.somedayReviewNudgeLastShownAt');
    });

    await waitReady(page);
    await expect(page.locator('#chat-history')).not.toContainText('You have 1 item in someday.');
  });
});
