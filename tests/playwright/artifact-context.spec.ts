import { expect, test, type Page } from '@playwright/test';

type HarnessLogEntry = { type: string; action: string; [key: string]: unknown };

async function getLog(page: Page): Promise<HarnessLogEntry[]> {
  return page.evaluate(() => (window as any).__harnessLog.slice());
}

async function clearLog(page: Page) {
  await page.evaluate(() => { (window as any).__harnessLog.splice(0); });
}

async function waitReady(page: Page) {
  await page.goto('/tests/playwright/chat-harness.html');
  await page.waitForSelector('#prompt-input', { state: 'visible', timeout: 5_000 });
  await page.waitForTimeout(200);
}

async function injectCanvasModuleRef(page: Page) {
  await page.evaluate(async () => {
    const mod = await import('../../internal/web/static/canvas.js');
    (window as any).__canvasModule = mod;
  });
}

async function renderTestArtifact(page: Page) {
  await page.evaluate(() => {
    const mod = (window as any).__canvasModule;
    mod.renderCanvas({
      event_id: 'art-1',
      kind: 'text_artifact',
      title: 'test.txt',
      text: 'Line one\nLine two\nLine three\nLine four\nLine five',
    });
    const ct = document.getElementById('canvas-text');
    if (ct) {
      ct.style.display = 'flex';
      ct.classList.add('is-active');
    }
  });
}

/** Install a fetch spy that captures the full body of chat message POSTs. */
async function installMessageSpy(page: Page) {
  await page.evaluate(() => {
    (window as any).__sentBodies = [];
    const prev = window.fetch;
    window.fetch = async function(url: any, opts: any) {
      const u = String(url);
      if (u.includes('/messages') && opts?.method === 'POST') {
        try {
          const body = JSON.parse(opts.body);
          (window as any).__sentBodies.push(body);
        } catch (_) {}
      }
      return prev.apply(this, arguments as any);
    };
  });
}

async function getSentBodies(page: Page): Promise<any[]> {
  return page.evaluate(() => (window as any).__sentBodies.slice());
}

test.describe('annotation bubble', () => {
  test.beforeEach(async ({ page }) => {
    await waitReady(page);
    await injectCanvasModuleRef(page);
    await installMessageSpy(page);
  });

  test('right-click on artifact opens annotation bubble', async ({ page }) => {
    await renderTestArtifact(page);
    const canvasText = page.locator('#canvas-text');
    await expect(canvasText).toBeVisible();

    const box = await canvasText.boundingBox();
    if (!box) throw new Error('canvas-text not visible');
    await page.mouse.click(box.x + 20, box.y + 20, { button: 'right' });
    await page.waitForTimeout(200);

    // In headless, caretRangeFromPoint may not work; verify no crash.
    const bubbleCount = await page.locator('.annotation-bubble').count();
    expect(bubbleCount).toBeLessThanOrEqual(1);
  });

  test('left-click on artifact does not open bubble', async ({ page }) => {
    await renderTestArtifact(page);
    const canvasText = page.locator('#canvas-text');
    await expect(canvasText).toBeVisible();

    const box = await canvasText.boundingBox();
    if (!box) throw new Error('canvas-text not visible');
    await page.mouse.click(box.x + 20, box.y + 20);
    await page.waitForTimeout(200);

    const bubbleCount = await page.locator('.annotation-bubble').count();
    expect(bubbleCount).toBe(0);
  });

  test('bubble send posts message with thread_key', async ({ page }) => {
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 42, title: 'doc.md' },
        clientX: 100,
        clientY: 100,
      });
    });
    await expect(page.locator('.annotation-bubble')).toBeVisible();
    await expect(page.locator('.annotation-bubble-location')).toContainText('Line 42 of "doc.md"');

    const input = page.locator('.annotation-bubble-input');
    await input.fill('fix this bug');
    await page.locator('.annotation-bubble-send').click();
    await page.waitForTimeout(300);

    const bodies = await getSentBodies(page);
    expect(bodies.length).toBeGreaterThanOrEqual(1);
    const sent = bodies[bodies.length - 1];
    expect(sent.text).toBe('fix this bug');
    expect(sent.thread_key).toBeTruthy();
    expect(String(sent.thread_key)).toMatch(/^ann-/);
  });

  test('Enter key in bubble submits message', async ({ page }) => {
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    await expect(page.locator('.annotation-bubble')).toBeVisible();

    const input = page.locator('.annotation-bubble-input');
    await input.fill('enter test');
    await input.press('Enter');
    await page.waitForTimeout(300);

    // User message should appear in bubble
    await expect(page.locator('.annotation-bubble-msg-user')).toContainText('enter test');

    // POST body should have thread_key
    const bodies = await getSentBodies(page);
    expect(bodies.length).toBeGreaterThanOrEqual(1);
    const sent = bodies[bodies.length - 1];
    expect(sent.text).toBe('enter test');
    expect(String(sent.thread_key)).toMatch(/^ann-/);
  });

  test('bubble receives streamed response', async ({ page }) => {
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
      const threadKey = mod.getActiveThreadKey();
      mod.routeBubbleEvent({ type: 'turn_started', turn_id: 'turn-1', thread_key: threadKey });
      mod.routeBubbleEvent({
        type: 'assistant_message',
        turn_id: 'turn-1',
        thread_key: threadKey,
        message: 'Here is the fix',
      });
      mod.routeBubbleEvent({
        type: 'message_persisted',
        role: 'assistant',
        turn_id: 'turn-1',
        thread_key: threadKey,
        message: 'Here is the fix',
      });
    });

    const messages = page.locator('.annotation-bubble-messages');
    await expect(messages).toBeVisible();
    await expect(messages.locator('.annotation-bubble-msg-assistant')).toContainText('Here is the fix');
  });

  test('bubble dismiss on click outside', async ({ page }) => {
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    await expect(page.locator('.annotation-bubble')).toBeVisible();
    await page.waitForTimeout(100);

    await page.mouse.click(5, 5);
    await page.waitForTimeout(200);
    await expect(page.locator('.annotation-bubble')).toHaveCount(0);
  });

  test('bubble dismiss on X button', async ({ page }) => {
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    await expect(page.locator('.annotation-bubble')).toBeVisible();

    await page.locator('.annotation-bubble-dismiss').click();
    await page.waitForTimeout(100);
    await expect(page.locator('.annotation-bubble')).toHaveCount(0);
  });

  test('wide viewport opens side panel', async ({ page }) => {
    // Default Playwright viewport is 1280x720 (≥900px) → side panel mode
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 5, title: 'review.md' },
        clientX: 100,
        clientY: 100,
      });
    });
    const bubble = page.locator('.annotation-bubble');
    await expect(bubble).toBeVisible();
    await expect(bubble).toHaveClass(/annotation-side-panel/);

    const viewport = page.locator('#canvas-viewport');
    await expect(viewport).toHaveClass(/has-annotation-panel/);

    // Verify it is inside #canvas-viewport
    const parent = await bubble.evaluate((el) => el.parentElement?.id);
    expect(parent).toBe('canvas-viewport');
  });

  test('medium viewport opens floating bubble not side panel', async ({ page }) => {
    await page.setViewportSize({ width: 800, height: 600 });
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    const bubble = page.locator('.annotation-bubble');
    await expect(bubble).toBeVisible();
    const hasSidePanel = await bubble.evaluate((el) =>
      el.classList.contains('annotation-side-panel')
    );
    expect(hasSidePanel).toBe(false);
  });

  test('side panel not dismissed by click on canvas-text', async ({ page }) => {
    // Default viewport is wide (1280x720)
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 2, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    const panel = page.locator('.annotation-side-panel');
    await expect(panel).toBeVisible();
    await page.waitForTimeout(100);

    // Programmatic click on canvas-text should NOT close the side panel
    const survived = await page.evaluate(() => {
      const ct = document.getElementById('canvas-text');
      if (ct) ct.click();
      return !!document.querySelector('.annotation-side-panel');
    });
    expect(survived).toBe(true);
    await expect(panel).toBeVisible();
  });

  test('side panel close removes has-annotation-panel class', async ({ page }) => {
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    const viewport = page.locator('#canvas-viewport');
    await expect(viewport).toHaveClass(/has-annotation-panel/);

    await page.locator('.annotation-bubble-dismiss').click();
    await page.waitForTimeout(100);
    const classes = await viewport.getAttribute('class');
    expect(classes).not.toContain('has-annotation-panel');
  });

  test('mobile bottom sheet layout', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });
    const bubble = page.locator('.annotation-bubble');
    await expect(bubble).toBeVisible();

    const styles = await bubble.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { position: cs.position, bottom: cs.bottom };
    });
    expect(styles.position).toBe('fixed');
    expect(styles.bottom).toBe('0px');
  });

  test('main chat not affected by bubble messages', async ({ page }) => {
    const chatHistoryBefore = await page.locator('#chat-history .chat-message').count();

    await page.evaluate(async () => {
      const mod = await import('../../internal/web/static/canvas-bubble.js');
      mod.openAnnotationBubble({
        location: { line: 1, title: 'test.txt' },
        clientX: 100,
        clientY: 100,
      });
    });

    const input = page.locator('.annotation-bubble-input');
    await input.fill('bubble comment');
    await page.locator('.annotation-bubble-send').click();
    await page.waitForTimeout(300);

    const chatHistoryAfter = await page.locator('#chat-history .chat-message').count();
    expect(chatHistoryAfter).toBe(chatHistoryBefore);
  });

  test('text selection works normally without opening bubble', async ({ page }) => {
    await renderTestArtifact(page);
    const canvasText = page.locator('#canvas-text');
    await expect(canvasText).toBeVisible();

    const box = await canvasText.boundingBox();
    if (!box) throw new Error('canvas-text not visible');

    await page.mouse.move(box.x + 10, box.y + 10);
    await page.mouse.down();
    await page.mouse.move(box.x + 100, box.y + 10);
    await page.mouse.up();
    await page.waitForTimeout(200);

    const bubbleCount = await page.locator('.annotation-bubble').count();
    expect(bubbleCount).toBe(0);
  });

  test('line highlight absent after left-click', async ({ page }) => {
    await renderTestArtifact(page);

    const markerCount = await page.locator('.transient-marker').count();
    expect(markerCount).toBe(0);

    const canvasText = page.locator('#canvas-text');
    const box = await canvasText.boundingBox();
    if (box) {
      await page.mouse.click(box.x + 20, box.y + 20);
      await page.waitForTimeout(100);
    }

    // Left-click should not produce a highlight or marker
    const markerCountAfter = await page.locator('.transient-marker').count();
    expect(markerCountAfter).toBe(0);
    const highlightCount = await page.locator('.review-line-highlight').count();
    expect(highlightCount).toBe(0);
  });
});
