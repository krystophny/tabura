import { expect, test, type Page } from '@playwright/test';

type HarnessLogEntry = Record<string, unknown>;

async function waitReady(page: Page) {
  await page.goto('/tests/playwright/harness.html');
  await page.waitForFunction(() => {
    const app = (window as any)._taburaApp;
    if (typeof app?.getState !== 'function') return false;
    const s = app.getState();
    const wsOpen = (window as any).WebSocket.OPEN;
    return s.chatWs?.readyState === wsOpen && s.canvasWs?.readyState === wsOpen;
  }, null, { timeout: 8_000 });
}

async function openInbox(page: Page) {
  await page.locator('#edge-left-tap').click();
  await expect(page.locator('#pr-file-pane')).toHaveClass(/is-open/);
  await page.locator('.sidebar-tab', { hasText: 'Inbox' }).click();
  await expect(page.locator('.sidebar-tab.is-active')).toContainText('Inbox');
}

async function getLog(page: Page): Promise<HarnessLogEntry[]> {
  return page.evaluate(() => {
    const log = (window as any).__harnessLog;
    return Array.isArray(log) ? log : [];
  });
}

async function clearLog(page: Page) {
  await page.evaluate(() => {
    (window as any).__harnessLog = [];
  });
}

async function waitForLogEntry(page: Page, type: string, action?: string) {
  await expect.poll(async () => {
    const log = await getLog(page);
    return log.some((entry) => {
      if (String(entry?.type || '') !== type) return false;
      if (!action) return true;
      return String(entry?.action || '') === action;
    });
  }).toBe(true);
}

test.describe('mail drafts', () => {
  test('new mail remains available in an empty inbox and supports save, reopen, suggestions, and send', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await waitReady(page);

    await page.evaluate(() => {
      (window as any).__setItemSidebarData({
        inbox: [],
        waiting: [],
        someday: [],
        done: [],
      });
      (window as any).__setItemSidebarActors([
        { id: 1, name: 'Ada', kind: 'human', email: 'ada@example.com' },
        { id: 2, name: 'Bob', kind: 'human', email: 'bob@example.com' },
      ]);
    });

    await openInbox(page);
    await expect(page.locator('#pr-file-list')).toContainText('No inbox items.');
    await expect(page.locator('#new-mail-trigger')).toBeVisible();

    await clearLog(page);
    await page.locator('#new-mail-trigger').click();
    await waitForLogEntry(page, 'api_fetch', 'mail_draft_create');

    await expect(page.locator('#canvas-text')).toHaveClass(/mail-draft-canvas/);
    await expect(page.locator('.mail-draft-title')).toContainText('Draft email');
    await expect.poll(async () => page.locator('#mail-draft-recipient-suggestions option').count()).toBe(2);
    await expect(page.locator('#mail-draft-recipient-suggestions option').nth(0)).toHaveAttribute('value', 'ada@example.com');
    await expect(page.locator('#mail-draft-recipient-suggestions option').nth(1)).toHaveAttribute('value', 'bob@example.com');

    await clearLog(page);
    await page.locator('[name="to"]').fill('ada@example.com');
    await page.locator('[name="cc"]').fill('bob@example.com');
    await page.locator('[name="subject"]').fill('Quarterly update');
    await page.locator('[name="body"]').fill('Ship the revised agenda today.');
    await waitForLogEntry(page, 'api_fetch', 'mail_draft_update');
    await expect(page.locator('#mail-draft-status')).toContainText('Draft saved');

    await page.locator('.sidebar-tab', { hasText: 'Done' }).click();
    await expect(page.locator('.sidebar-tab.is-active')).toContainText('Done');
    await page.locator('.sidebar-tab', { hasText: 'Inbox' }).click();
    await expect(page.locator('#pr-file-list')).toContainText('Quarterly update');

    await clearLog(page);
    await page.locator('#pr-file-list .pr-file-item', { hasText: 'Quarterly update' }).click();
    await waitForLogEntry(page, 'api_fetch', 'mail_draft_get');
    await expect(page.locator('[name="to"]')).toHaveValue('ada@example.com');
    await expect(page.locator('[name="cc"]')).toHaveValue('bob@example.com');
    await expect(page.locator('[name="subject"]')).toHaveValue('Quarterly update');
    await expect(page.locator('[name="body"]')).toHaveValue('Ship the revised agenda today.');

    await page.locator('#edge-left-tap').click();
    await expect(page.locator('#pr-file-pane')).not.toHaveClass(/is-open/);

    await clearLog(page);
    await page.locator('[name="body"]').fill('Ship the revised agenda before lunch.');
    await page.locator('#mail-draft-send').click();
    await waitForLogEntry(page, 'api_fetch', 'mail_draft_update');
    await waitForLogEntry(page, 'api_fetch', 'mail_draft_send');
    await expect(page.locator('#mail-draft-status')).toContainText('Sent');

    await page.locator('#edge-left-tap').click();
    await expect(page.locator('#pr-file-pane')).toHaveClass(/is-open/);
    await page.locator('.sidebar-tab', { hasText: 'Inbox' }).click();
    await expect(page.locator('#pr-file-list')).not.toContainText('Quarterly update');

    await page.locator('.sidebar-tab', { hasText: 'Done' }).click();
    await expect(page.locator('#pr-file-list')).toContainText('Quarterly update');
  });

  test('reply drafts seed recipient and subject from the selected email item', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await waitReady(page);

    await page.evaluate(() => {
      (window as any).__setItemSidebarData({
        inbox: [{
          id: 812,
          title: 'Reply to client',
          state: 'inbox',
          sphere: 'private',
          artifact_id: 612,
          source: 'exchange',
          source_ref: 'msg-812',
          artifact_title: 'Client question',
          artifact_kind: 'email',
          actor_name: 'Client',
          created_at: '2026-03-10 10:00:00',
          updated_at: '2026-03-10 10:05:00',
        }],
        waiting: [],
        someday: [],
        done: [],
      });
      (window as any).__setItemSidebarArtifacts({
        612: {
          id: 612,
          kind: 'email',
          title: 'Client question',
          meta_json: JSON.stringify({
            subject: 'Client question',
            sender: 'Client <client@example.com>',
            thread_id: 'thread-812',
            body: 'Can you send the revised proposal?',
          }),
        },
      });
    });

    await openInbox(page);
    await page.locator('#pr-file-list .pr-file-item').first().click();
    await expect(page.locator('#canvas-text')).toContainText('Can you send the revised proposal?');
    await expect(page.locator('#reply-mail-trigger')).toBeVisible();

    await clearLog(page);
    await page.locator('#reply-mail-trigger').click();
    await waitForLogEntry(page, 'api_fetch', 'mail_draft_reply');

    await expect(page.locator('#canvas-text')).toHaveClass(/mail-draft-canvas/);
    await expect(page.locator('.mail-draft-account')).toContainText('Work Exchange');
    await expect(page.locator('.mail-draft-account')).toContainText('exchange');
    await expect(page.locator('[name="to"]')).toHaveValue('client@example.com');
    await expect(page.locator('[name="subject"]')).toHaveValue('Re: Client question');
  });
});
