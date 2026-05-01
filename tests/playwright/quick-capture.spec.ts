import { expect, test, type Page } from '@playwright/test';

async function waitReady(page: Page) {
  await page.goto('/tests/playwright/harness.html');
  await page.waitForFunction(() => {
    const app = (window as any)._slopshellApp;
    if (typeof app?.getState !== 'function') return false;
    const s = app.getState();
    return s.chatWs && s.chatWs.readyState === (window as any).WebSocket.OPEN;
  }, null, { timeout: 5_000 });
}

async function captureLog(page: Page) {
  return page.evaluate(() => (window as any).__harnessLog.filter((entry: any) => entry?.type === 'api_fetch' && entry?.action === 'item_capture'));
}

test.describe('GTD quick capture affordance', () => {
  test('always-visible toggle opens the capture sheet and posts an action by default', async ({ page }) => {
    await waitReady(page);

    const toggle = page.locator('#quick-capture-toggle');
    await expect(toggle).toBeVisible();

    await toggle.click();
    await expect(page.locator('#quick-capture-sheet')).toBeVisible();

    await page.locator('#quick-capture-title').fill('Reply to grant request');
    await page.locator('#quick-capture-submit').click();

    await expect.poll(async () => (await captureLog(page)).length, { timeout: 5_000 }).toBe(1);
    const log = await captureLog(page);
    expect(log[0].payload.title).toBe('Reply to grant request');
    expect(log[0].payload.kind).toBe('action');
    await expect(page.locator('#quick-capture-status')).toContainText('inbox');
  });

  test('switching to project kind sends kind=project and hides the project link field', async ({ page }) => {
    await waitReady(page);
    await page.locator('#quick-capture-toggle').click();
    await page.locator('input[name="quick-capture-kind"][value="project"]').check();
    await expect(page.locator('[data-quick-capture-action-only]')).toBeHidden();
    await page.locator('#quick-capture-title').fill('Ship dialog refresh');
    await page.locator('#quick-capture-submit').click();
    await expect.poll(async () => (await captureLog(page)).length, { timeout: 5_000 }).toBe(1);
    const log = await captureLog(page);
    expect(log[0].payload.kind).toBe('project');
  });

  test('action capture under a project item id passes the link in the payload', async ({ page }) => {
    await waitReady(page);
    await page.locator('#quick-capture-toggle').click();
    await page.locator('#quick-capture-title').fill('Schedule first 1:1');
    await page.locator('summary').first().click();
    await page.locator('#quick-capture-project-item-id').fill('42');
    await page.locator('#quick-capture-submit').click();
    await expect.poll(async () => (await captureLog(page)).length, { timeout: 5_000 }).toBe(1);
    const log = await captureLog(page);
    expect(log[0].payload.project_item_id).toBe(42);
    expect(log[0].payload.kind).toBe('action');
  });

  test('rejects empty title without sending a request', async ({ page }) => {
    await waitReady(page);
    await page.locator('#quick-capture-toggle').click();
    await page.locator('#quick-capture-submit').click();
    await expect(page.locator('#quick-capture-error')).toContainText('Title is required');
    expect((await captureLog(page)).length).toBe(0);
  });
});
