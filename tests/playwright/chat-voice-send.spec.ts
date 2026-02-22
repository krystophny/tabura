import { expect, test, type Page } from '@playwright/test';

type HarnessLogEntry = { type: string; action: string; [key: string]: unknown };

async function getLog(page: Page): Promise<HarnessLogEntry[]> {
  return page.evaluate(() => (window as any).__harnessLog.slice());
}

async function clearLog(page: Page) {
  await page.evaluate(() => { (window as any).__harnessLog.splice(0); });
}

async function waitForLogEntry(page: Page, type: string, action: string) {
  await expect.poll(async () => {
    const log = await getLog(page);
    return log.some(e => e.type === type && e.action === action);
  }, { timeout: 5_000 }).toBe(true);
}

async function waitForSTTAction(page: Page, action: string) {
  await waitForLogEntry(page, 'stt', action);
}

function touchStart(page: Page, selector: string) {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel);
    if (!el) throw new Error(`element not found: ${sel}`);
    const rect = el.getBoundingClientRect();
    const cx = rect.left + rect.width / 2;
    const cy = rect.top + rect.height / 2;
    const touch = new Touch({
      identifier: 1,
      target: el,
      clientX: cx,
      clientY: cy,
      pageX: cx,
      pageY: cy,
    });
    el.dispatchEvent(new TouchEvent('touchstart', {
      bubbles: true,
      cancelable: true,
      touches: [touch],
      targetTouches: [touch],
      changedTouches: [touch],
    }));
  }, selector);
}

function touchEnd(page: Page, selector: string) {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel);
    if (!el) throw new Error(`element not found: ${sel}`);
    const rect = el.getBoundingClientRect();
    const cx = rect.left + rect.width / 2;
    const cy = rect.top + rect.height / 2;
    const touch = new Touch({
      identifier: 1,
      target: el,
      clientX: cx,
      clientY: cy,
      pageX: cx,
      pageY: cy,
    });
    window.dispatchEvent(new TouchEvent('touchend', {
      bubbles: true,
      cancelable: true,
      touches: [],
      targetTouches: [],
      changedTouches: [touch],
    }));
  }, selector);
}

test.beforeEach(async ({ page }) => {
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log(`BROWSER [error]: ${msg.text()}`);
  });
  page.on('pageerror', (err) => console.log(`PAGE ERROR: ${err.message}`));
  await page.goto('/tests/playwright/chat-harness.html');
  await page.waitForFunction(() => typeof (window as any)._tabulaApp?.getState === 'function');
  await page.waitForTimeout(200);
});

test('touch hold on Send button starts voice recording after threshold', async ({ page }) => {
  await clearLog(page);
  await touchStart(page, '#btn-chat-send');
  await page.waitForTimeout(500);

  await waitForSTTAction(page, 'start');
  await waitForLogEntry(page, 'recorder', 'start');

  await touchEnd(page, '#btn-chat-send');
  await waitForSTTAction(page, 'stop');
  await waitForLogEntry(page, 'recorder', 'stop');

  const log = await getLog(page);
  const sttActions = log.filter(e => e.type === 'stt').map(e => e.action);
  expect(sttActions).toContain('start');
  expect(sttActions).toContain('append');
  expect(sttActions).toContain('stop');
});

test('short tap on Send button does not start voice recording', async ({ page }) => {
  await clearLog(page);
  await touchStart(page, '#btn-chat-send');
  // Release before 300ms threshold
  await page.waitForTimeout(100);
  await touchEnd(page, '#btn-chat-send');
  // Wait a bit to ensure nothing fires
  await page.waitForTimeout(500);

  const log = await getLog(page);
  const sttActions = log.filter(e => e.type === 'stt');
  expect(sttActions).toHaveLength(0);
});

test('touch release during mic init still completes the recording flow', async ({ page }) => {
  // Slow down getUserMedia to simulate permission dialog
  await page.evaluate(() => {
    const original = navigator.mediaDevices.getUserMedia;
    navigator.mediaDevices.getUserMedia = async (constraints) => {
      await new Promise(r => setTimeout(r, 400));
      return original.call(navigator.mediaDevices, constraints);
    };
  });

  await clearLog(page);
  await touchStart(page, '#btn-chat-send');
  // Wait for hold threshold to fire beginChatVoiceCapture
  await page.waitForTimeout(350);
  // Release while getUserMedia is still pending (within 400ms delay)
  await touchEnd(page, '#btn-chat-send');
  // The stopRequested flag should be set, and once getUserMedia resolves
  // the flow should complete: recorder starts then immediately stops
  await waitForSTTAction(page, 'stop');

  const log = await getLog(page);
  const sttActions = log.filter(e => e.type === 'stt').map(e => e.action);
  expect(sttActions).toContain('start');
  expect(sttActions).toContain('stop');
});

test('mouse hold on Send button starts voice recording (desktop)', async ({ page }) => {
  await clearLog(page);
  const btn = page.locator('#btn-chat-send');
  const box = await btn.boundingBox();
  if (!box) throw new Error('Send button not found');

  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.waitForTimeout(500);
  await waitForSTTAction(page, 'start');

  await page.mouse.up();
  await waitForSTTAction(page, 'stop');

  const log = await getLog(page);
  const sttActions = log.filter(e => e.type === 'stt').map(e => e.action);
  expect(sttActions).toContain('start');
  expect(sttActions).toContain('stop');
});

test('mouse release during mic init still completes recording (desktop)', async ({ page }) => {
  await page.evaluate(() => {
    const original = navigator.mediaDevices.getUserMedia;
    navigator.mediaDevices.getUserMedia = async (constraints) => {
      await new Promise(r => setTimeout(r, 400));
      return original.call(navigator.mediaDevices, constraints);
    };
  });

  await clearLog(page);
  const btn = page.locator('#btn-chat-send');
  const box = await btn.boundingBox();
  if (!box) throw new Error('Send button not found');

  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.waitForTimeout(350);
  await page.mouse.up();
  await waitForSTTAction(page, 'stop');

  const log = await getLog(page);
  const sttActions = log.filter(e => e.type === 'stt').map(e => e.action);
  expect(sttActions).toContain('start');
  expect(sttActions).toContain('stop');
});
