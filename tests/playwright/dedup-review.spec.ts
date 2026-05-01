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

async function openDedupReview(page: Page) {
  await page.locator('#edge-left-tap').click();
  await expect(page.locator('#pr-file-pane')).toHaveClass(/is-open/);
  await page.locator('#sidebar-secondary-toggle').click();
  await page.locator('.sidebar-secondary-row[data-section-id="dedup"]').click();
  await expect.poll(async () => page.evaluate(() => {
    const app = (window as any)._slopshellApp;
    return String(app?.getState?.().itemSidebarFilters?.section || '');
  })).toBe('dedup');
}

function actionCandidate() {
  return {
    id: 910,
    kind: 'action',
    state: 'open',
    confidence: 0.92,
    outcome: 'Review the budget revision',
    reasoning: 'local LLM matched outcome text, person overlap, and source context',
    detector: 'local-llm',
    detected_at: '2026-04-30T09:30:00Z',
    items: [
      {
        item: { id: 701, title: 'Review revised budget', kind: 'action', state: 'next', sphere: 'private', updated_at: '2026-04-30T09:00:00Z' },
        outcome: 'Review revised budget',
        source_bindings: [{ provider: 'todoist', object_type: 'task', remote_id: 'task-701', container_ref: 'Finance', remote_updated_at: '2026-04-30T08:00:00Z' }],
        source_containers: ['Finance'],
        dates: ['due=2026-05-03T12:00:00Z'],
      },
      {
        item: { id: 702, title: 'Check budget revision', kind: 'action', state: 'waiting', sphere: 'private', updated_at: '2026-04-30T09:05:00Z' },
        outcome: 'Check budget revision',
        source_bindings: [{ provider: 'github', object_type: 'issue', remote_id: 'sloppy-org/slopshell#745', container_ref: 'Roadmap', remote_updated_at: '2026-04-30T08:30:00Z' }],
        source_containers: ['Roadmap'],
        dates: ['follow_up=2026-05-02T09:00:00Z'],
      },
    ],
  };
}

function projectCandidate() {
  return {
    id: 920,
    kind: 'project',
    state: 'open',
    confidence: 0.87,
    outcome: 'Ship dedup review queue',
    reasoning: 'deterministic outcome overlap',
    detector: 'deterministic',
    detected_at: '2026-04-30T10:00:00Z',
    items: [
      {
        item: { id: 801, title: 'Ship dedup review queue', kind: 'project', state: 'next', sphere: 'private', updated_at: '2026-04-30T09:00:00Z' },
        outcome: 'Ship dedup review queue',
        source_bindings: [{ provider: 'markdown', object_type: 'commitment', remote_id: 'brain/ship-dedup.md', container_ref: 'Brain' }],
        source_containers: ['Brain'],
        dates: ['visible=2026-04-30T09:00:00Z'],
      },
      {
        item: { id: 802, title: 'Dedup review UX', kind: 'project', state: 'next', sphere: 'private', updated_at: '2026-04-30T09:10:00Z' },
        outcome: 'Dedup review UX',
        source_bindings: [{ provider: 'todoist', object_type: 'project', remote_id: 'project-802', container_ref: 'GTD' }],
        source_containers: ['GTD'],
        dates: ['due=2026-05-05T12:00:00Z'],
      },
    ],
  };
}

test.describe('GTD dedup review queue (#745)', () => {
  test('renders action candidate source bindings and records keep-separate/review-later decisions', async ({ page }) => {
    await waitReady(page);
    await page.evaluate((candidate) => {
      (window as any).__itemDedupCandidates = [candidate];
      (window as any).__itemSidebarSectionCounts = { dedup_review: 1 };
    }, actionCandidate());
    await openDedupReview(page);

    const row = page.locator('.dedup-candidate-row[data-item-id="910"]');
    await expect(row).toContainText('Action duplicate: Review the budget revision');
    await expect(row).toContainText('local LLM matched outcome text');
    await expect(row).toContainText('todoist:task:task-701');
    await expect(row).toContainText('github:issue:sloppy-org/slopshell#745');
    await expect(row).toContainText('containers Finance');
    await expect(row).toContainText('containers Roadmap');
    await expect(row).toContainText('due=2026-05-03T12:00:00Z');
    await expect(row).toContainText('follow_up=2026-05-02T09:00:00Z');

    await row.getByRole('button', { name: 'Review later' }).click();
    await expect(page.locator('.dedup-candidate-row[data-item-id="910"]')).toBeVisible();
    await row.getByRole('button', { name: 'Keep separate' }).click();
    await expect(page.locator('#pr-file-list')).toContainText('No duplicate candidates.');
    await expect.poll(async () => page.evaluate(() => (window as any).__harnessLog
      .filter((entry: any) => entry.action === 'dedup_action')
      .map((entry: any) => entry.payload.action))).toEqual(['review_later', 'keep_separate']);
  });

  test('renders project-item candidates separately and merges into the selected canonical item', async ({ page }) => {
    await waitReady(page);
    await page.evaluate((candidate) => {
      (window as any).__itemDedupCandidates = [candidate];
      (window as any).__itemSidebarSectionCounts = { dedup_review: 1 };
    }, projectCandidate());
    await openDedupReview(page);

    const row = page.locator('.dedup-candidate-row[data-item-id="920"]');
    await expect(row).toContainText('Project-item duplicate: Ship dedup review queue');
    await expect(row).toContainText('markdown:commitment:brain/ship-dedup.md');
    await expect(row).toContainText('todoist:project:project-802');

    await row.getByRole('button', { name: 'Merge into #801' }).click();
    await expect(page.locator('#pr-file-list')).toContainText('No duplicate candidates.');
    await expect.poll(async () => page.evaluate(() => (window as any).__harnessLog
      .find((entry: any) => entry.action === 'dedup_action')?.payload)).toMatchObject({
        candidate_id: 920,
        action: 'merge',
        canonical_item_id: 801,
      });
  });
});
