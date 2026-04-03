import { expect, test, type Page } from '@playwright/test';

async function waitReady(page: Page) {
  await page.goto('/tests/playwright/harness.html');
  await page.waitForFunction(() => {
    const app = (window as any)._slopshellApp;
    if (typeof app?.getState !== 'function') return false;
    const s = app.getState();
    const wsOpen = (window as any).WebSocket.OPEN;
    return s.chatWs?.readyState === wsOpen && s.canvasWs?.readyState === wsOpen;
  }, null, { timeout: 8_000 });
}

async function injectCanvasEvent(page: Page, payload: Record<string, unknown>) {
  await page.evaluate((eventPayload) => {
    const app = (window as any)._slopshellApp;
    const canvasWs = app?.getState?.().canvasWs;
    if (!canvasWs || typeof canvasWs.injectEvent !== 'function') {
      throw new Error('canvas websocket unavailable');
    }
    canvasWs.injectEvent(eventPayload);
  }, payload);
}

async function injectChatEvent(page: Page, payload: Record<string, unknown>) {
  await page.evaluate((eventPayload) => {
    const app = (window as any)._slopshellApp;
    const sessionId = String(app?.getState?.().chatSessionId || '');
    const sessions = (window as any).__mockWsSessions || [];
    const chatWs = sessions.find((ws: any) => typeof ws.url === 'string'
      && ws.url.includes('/ws/chat/')
      && (!sessionId || ws.url.includes(`/ws/chat/${sessionId}`)));
    if (!chatWs?.injectEvent) {
      throw new Error('chat websocket unavailable');
    }
    chatWs.injectEvent(eventPayload);
  }, payload);
}

function longMarkdownDocument() {
  return Array.from({ length: 48 }, (_, index) => [
    `## Section ${index + 1}`,
    '',
    `This is the body for section ${index + 1}. `.repeat(12),
    '',
  ].join('\n')).join('\n');
}

async function wheelFlip(page: Page, deltaX: number) {
  await page.locator('#canvas-viewport').evaluate((el, dX) => {
    el.dispatchEvent(new WheelEvent('wheel', {
      deltaX: Number(dX),
      deltaY: 0,
      bubbles: true,
      cancelable: true,
    }));
  }, deltaX);
}

async function holdSwipe(page: Page, { startX, endX, y, holdMs }: { startX: number; endX: number; y: number; holdMs: number }) {
  await page.locator('#canvas-viewport').evaluate(async (el, payload) => {
    if (typeof Touch === 'undefined') return;
    const start = { x: Number(payload.startX), y: Number(payload.y) };
    const end = { x: Number(payload.endX), y: Number(payload.y) };
    const target = document.elementFromPoint(start.x, start.y) || el;
    const makeTouch = (clientX: number, clientY: number) => new Touch({
      identifier: 1,
      target,
      clientX,
      clientY,
      pageX: clientX,
      pageY: clientY,
      screenX: clientX,
      screenY: clientY,
    });
    const startTouch = makeTouch(start.x, start.y);
    target.dispatchEvent(new TouchEvent('touchstart', {
      bubbles: true,
      cancelable: true,
      touches: [startTouch],
      changedTouches: [startTouch],
      targetTouches: [startTouch],
    }));
    await new Promise((resolve) => window.setTimeout(resolve, Number(payload.holdMs || 0)));
    const moveTouch = makeTouch(end.x, end.y);
    target.dispatchEvent(new TouchEvent('touchmove', {
      bubbles: true,
      cancelable: true,
      touches: [moveTouch],
      changedTouches: [moveTouch],
      targetTouches: [moveTouch],
    }));
    target.dispatchEvent(new TouchEvent('touchend', {
      bubbles: true,
      cancelable: true,
      touches: [],
      changedTouches: [moveTouch],
      targetTouches: [],
    }));
  }, { startX, endX, y, holdMs });
}

function twoFileDiff(): string {
  return [
    'diff --git a/docs/one.md b/docs/one.md',
    'index 1111111..2222222 100644',
    '--- a/docs/one.md',
    '+++ b/docs/one.md',
    '@@ -1 +1 @@',
    '-old',
    '+new',
    'diff --git a/src/two.js b/src/two.js',
    'index 3333333..4444444 100644',
    '--- a/src/two.js',
    '+++ b/src/two.js',
    '@@ -1 +1 @@',
    '-console.log("before");',
    '+console.log("after");',
  ].join('\n');
}

test.describe('canvas page flipping', () => {
  test('horizontal wheel flips document pages instead of scrolling', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await waitReady(page);

    await injectCanvasEvent(page, {
      kind: 'text_artifact',
      event_id: 'evt-pages-1',
      title: 'notes.md',
      path: 'notes.md',
      text: longMarkdownDocument(),
    });

    await expect(page.locator('#canvas-text .canvas-page-indicator')).toContainText(/^Page 1 \/ /);
    await expect(page.locator('#canvas-text')).toContainText('Section 1');

    await wheelFlip(page, 120);

    await expect(page.locator('#canvas-text .canvas-page-indicator')).toContainText(/^Page 2 \/ /);
  });

  test('long-held horizontal swipe flips artifacts while short flips stay page-scoped', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await waitReady(page);

    await injectCanvasEvent(page, {
      kind: 'text_artifact',
      event_id: 'evt-pr-pages',
      title: '.slopshell/artifacts/pr/pr-19.diff',
      text: twoFileDiff(),
    });

    await expect(page.locator('body')).toHaveClass(/pr-review-mode/);
    await expect(page.locator('#canvas-text')).toContainText('docs/one.md');

    await holdSwipe(page, {
      startX: 980,
      endX: 820,
      y: 420,
      holdMs: 360,
    });

    await expect(page.locator('#canvas-text')).toContainText('src/two.js');
  });

  test('navigate_canvas system action flips the current document page', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await waitReady(page);

    await injectCanvasEvent(page, {
      kind: 'text_artifact',
      event_id: 'evt-pages-voice-1',
      title: 'slides.md',
      path: 'slides.md',
      text: longMarkdownDocument(),
    });

    await expect(page.locator('#canvas-text .canvas-page-indicator')).toContainText(/^Page 1 \/ /);

    await injectChatEvent(page, {
      type: 'system_action',
      action: {
        type: 'navigate_canvas',
        scope: 'page_or_artifact',
        direction: 'next',
      },
    });

    await expect(page.locator('#canvas-text .canvas-page-indicator')).toContainText(/^Page 2 \/ /);
    await expect(page.locator('#status-text')).toContainText('next page');
  });

  test('navigate_canvas artifact scope jumps directly between artifacts', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await waitReady(page);

    await injectCanvasEvent(page, {
      kind: 'text_artifact',
      event_id: 'evt-pr-artifact-action',
      title: '.slopshell/artifacts/pr/pr-23.diff',
      text: twoFileDiff(),
    });

    await expect(page.locator('#canvas-text')).toContainText('docs/one.md');

    await injectChatEvent(page, {
      type: 'system_action',
      action: {
        type: 'navigate_canvas',
        scope: 'artifact',
        direction: 'next',
      },
    });

    await expect(page.locator('#canvas-text')).toContainText('src/two.js');
    await expect(page.locator('#status-text')).toContainText('next document');
  });
});
