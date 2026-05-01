import { expect, test, type Page } from '@playwright/test';

async function waitReady(page: Page) {
  await page.goto('/tests/playwright/harness.html');
  await page.waitForFunction(() => {
    const app = (window as any)._slopshellApp;
    if (typeof app?.getState !== 'function') return false;
    const state = app.getState();
    return state.chatWs && state.chatWs.readyState === (window as any).WebSocket.OPEN;
  }, null, { timeout: 5_000 });
}

async function seedBrainWorkspace(page: Page) {
  await page.evaluate(() => {
    (window as any).__setProjects([
      {
        id: 'brain',
        name: 'Brain',
        kind: 'linked',
        sphere: 'work',
        workspace_path: '/tmp/vault/brain',
        root_path: '/tmp/vault/brain',
        chat_session_id: 'chat-brain',
        canvas_session_id: 'brain',
        chat_mode: 'chat',
        chat_model: 'local',
        chat_model_reasoning_effort: 'none',
        unread: false,
        review_pending: false,
        run_state: { active_turns: 0, queued_turns: 0, is_working: false, status: 'idle' },
      },
    ], 'brain');
    const state = (window as any)._slopshellApp?.getState?.();
    if (state) {
      state.activeWorkspaceId = 'brain';
      state.projects = [
        {
          id: 'brain',
          name: 'Brain',
          sphere: 'work',
          workspace_path: '/tmp/vault/brain',
          root_path: '/tmp/vault/brain',
        },
      ];
    }
  });
  await page.waitForFunction(() => {
    const state = (window as any)._slopshellApp?.getState?.();
    return String(state?.activeWorkspaceId || '') === 'brain';
  }, null, { timeout: 5_000 });
}

async function renderBrainCanvasArtifact(page: Page) {
  await page.evaluate(async () => {
    const mod = await import('../../internal/web/static/canvas.js');
    (window as any).__canvasModule = mod;
    (window as any).__mockMarkdownLinkPanel = {
      ok: true,
      source_path: 'topics/active.md',
      outgoing: [],
      broken_count: 0,
      backlinks: [],
    };
    (window as any).__mockBrainCanvas = {
      ok: true,
      name: 'default',
      cards: [
        {
          id: 'card-note',
          type: 'file',
          x: 20,
          y: 30,
          width: 220,
          height: 140,
          title: 'active',
          body: '# Active',
          open_url: '/api/workspaces/brain/markdown-link/file?path=brain%2Ftopics%2Factive.md',
          binding: { kind: 'note', path: 'topics/active.md' },
        },
        {
          id: 'card-item',
          type: 'text',
          x: 260,
          y: 30,
          width: 220,
          height: 140,
          title: 'Follow up',
          body: 'action · next',
          binding: { kind: 'item', id: 7 },
        },
      ],
      edges: [
        {
          id: 'edge-review',
          fromNode: 'card-note',
          toNode: 'card-item',
          label: 'follow up',
          mode: 'visual',
        },
      ],
    };
    (window as any).__mockMarkdownLinkFileText = '# Active note opened';
    mod.renderCanvas({
      event_id: 'brain-canvas-active',
      kind: 'text_artifact',
      title: 'topics/active.md',
      path: 'topics/active.md',
      text: 'Active note',
    });
  });
}

test.beforeEach(async ({ page }) => {
  await waitReady(page);
  await seedBrainWorkspace(page);
});

test('brain canvas cards persist drag/resize and open backing object', async ({ page }) => {
  await renderBrainCanvasArtifact(page);

  await expect(page.locator('.canvas-brain-cards-board')).toBeVisible();
  await expect(page.locator('.canvas-brain-card[data-card-id="card-note"]')).toBeVisible();
  await expect(page.locator('.canvas-brain-card[data-card-id="card-item"]')).toBeVisible();
  await expect(page.locator('.canvas-brain-edge[data-edge-id="edge-review"]')).toHaveCount(1);
  await expect(page.locator('.canvas-brain-edge-label[data-edge-id="edge-review"]')).toHaveText('follow up');

  const noteCard = page.locator('.canvas-brain-card[data-card-id="card-note"]');
  expect(await noteCard.evaluate((el) => (el as HTMLElement).style.transform)).toContain('translate(20px, 30px)');

  await page.evaluate(() => {
    const card = document.querySelector('.canvas-brain-card[data-card-id="card-note"]');
    if (!card) throw new Error('note card missing');
    const evt = (type: string, x: number, y: number) =>
      new PointerEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y, pointerId: 1, pointerType: 'mouse' });
    card.dispatchEvent(evt('pointerdown', 100, 100));
    window.dispatchEvent(evt('pointermove', 220, 240));
    window.dispatchEvent(evt('pointerup', 220, 240));
  });

  await expect.poll(async () => {
    const log = await page.evaluate(() => (window as any).__brainCanvasPatchLog || []);
    return log.some((entry: any) => entry?.card_id === 'card-note'
      && entry?.payload && Object.prototype.hasOwnProperty.call(entry.payload, 'x')
      && entry.payload.x !== 20);
  }, { timeout: 5_000 }).toBe(true);

  await page.evaluate(() => {
    const handle = document.querySelector('.canvas-brain-card[data-card-id="card-note"] .canvas-brain-card-resize');
    if (!handle) throw new Error('resize handle missing');
    const evt = (type: string, x: number, y: number) =>
      new PointerEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y, pointerId: 2, pointerType: 'mouse' });
    handle.dispatchEvent(evt('pointerdown', 0, 0));
    window.dispatchEvent(evt('pointermove', 90, 70));
    window.dispatchEvent(evt('pointerup', 90, 70));
  });

  await expect.poll(async () => {
    const log = await page.evaluate(() => (window as any).__brainCanvasPatchLog || []);
    return log.some((entry: any) => entry?.card_id === 'card-note'
      && entry?.payload && Object.prototype.hasOwnProperty.call(entry.payload, 'width')
      && entry.payload.width > 220);
  }, { timeout: 5_000 }).toBe(true);

  await noteCard.locator('.canvas-brain-card-open').click();
  await expect(page.locator('#canvas-text')).toContainText('Active note opened');
});

test('brain canvas item card writes title through to backend', async ({ page }) => {
  await renderBrainCanvasArtifact(page);

  await expect(page.locator('.canvas-brain-cards-board')).toBeVisible({ timeout: 10_000 });
  await expect(page.locator('.canvas-brain-card[data-card-id="card-item"]')).toBeVisible({ timeout: 10_000 });
  const titleInput = page.locator('.canvas-brain-card[data-card-id="card-item"] .canvas-brain-card-title');
  await expect(titleInput).toBeVisible({ timeout: 10_000 });
  await page.evaluate(() => {
    const title = document.querySelector('.canvas-brain-card[data-card-id="card-item"] .canvas-brain-card-title') as HTMLElement;
    if (!title) throw new Error('item title missing');
    title.textContent = 'Item retitled via canvas';
    title.dispatchEvent(new FocusEvent('blur', { bubbles: true }));
  });

  await expect.poll(async () => {
    const log = await page.evaluate(() => (window as any).__brainCanvasPatchLog || []);
    return log.some((entry: any) => entry?.card_id === 'card-item'
      && entry?.payload?.title === 'Item retitled via canvas');
  }, { timeout: 5_000 }).toBe(true);
});

test('brain canvas edges can be created, promoted, and deleted', async ({ page }) => {
  await renderBrainCanvasArtifact(page);

  await expect(page.locator('.canvas-brain-edge-row[data-edge-id="edge-review"]')).toBeVisible();
  const row = page.locator('.canvas-brain-edge-row[data-edge-id="edge-review"]');
  await row.locator('.canvas-brain-edge-relation').fill('supports');
  await row.getByRole('button', { name: 'Promote' }).click();

  await expect.poll(async () => {
    const log = await page.evaluate(() => (window as any).__brainCanvasEdgeLog || []);
    return log.some((entry: any) => entry?.action === 'promote'
      && entry?.edge_id === 'edge-review'
      && entry?.payload?.relation === 'supports');
  }, { timeout: 5_000 }).toBe(true);
  await expect(page.locator('.canvas-brain-edge[data-edge-id="edge-review"][data-edge-mode="semantic"]')).toHaveCount(1);

  await page.locator('.canvas-brain-edge-form .canvas-brain-edge-card').first().selectOption('card-item');
  await page.locator('.canvas-brain-edge-form .canvas-brain-edge-card').nth(1).selectOption('card-note');
  await page.locator('.canvas-brain-edge-form .canvas-brain-edge-label-input').fill('blocks');
  await page.locator('.canvas-brain-edge-form .canvas-brain-edge-mode').selectOption('semantic');
  await page.locator('.canvas-brain-edge-form .canvas-brain-edge-relation').fill('blocks');
  await page.locator('.canvas-brain-edge-form').getByRole('button', { name: 'Add edge' }).click();

  await expect.poll(async () => {
    const log = await page.evaluate(() => (window as any).__brainCanvasEdgeLog || []);
    return log.some((entry: any) => entry?.action === 'create'
      && entry?.payload?.fromNode === 'card-item'
      && entry?.payload?.toNode === 'card-note'
      && entry?.payload?.mode === 'semantic'
      && entry?.payload?.relation === 'blocks');
  }, { timeout: 5_000 }).toBe(true);
  await expect(page.locator('.canvas-brain-edge[data-edge-id="edge-2"][data-edge-mode="semantic"]')).toHaveCount(1);

  await page.locator('.canvas-brain-edge-row[data-edge-id="edge-review"]').getByRole('button', { name: 'Delete' }).click();
  await expect.poll(async () => {
    const log = await page.evaluate(() => (window as any).__brainCanvasEdgeLog || []);
    return log.some((entry: any) => entry?.action === 'delete' && entry?.edge_id === 'edge-review');
  }, { timeout: 5_000 }).toBe(true);
  await expect(page.locator('.canvas-brain-edge[data-edge-id="edge-review"]')).toHaveCount(0);
});
