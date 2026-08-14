import { expect, test } from '@playwright/test';

test.describe('Signal North HTMX showcase', () => {
  test('renders the homepage with local HTMX', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Signal North keeps the signal visible.' })).toBeVisible();
    expect(await page.evaluate(() => (window as Window & { htmx?: { version?: string } }).htmx?.version)).toBe('2.0.4');
  });

  test('refreshes telemetry over HTTP', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Refresh telemetry' }).click();
    await expect(page.locator('#telemetry-panel')).toContainText('Telemetry refreshed');
  });

  test('debounces command search', async ({ page }) => {
    const searchRequests: string[] = [];
    page.on('request', request => {
      if (request.url().includes('/demo/search')) searchRequests.push(request.url());
    });
    await page.goto('/');
    await page.locator('#command-search').fill('dep');
    await expect(page.locator('#search-results')).toContainText('deploy api');
    expect(searchRequests).toHaveLength(1);
  });

  test('submits a command and applies the OOB metric update', async ({ page }) => {
    await page.goto('/');
    const before = await page.locator('#metric-requests .metric-value').innerText();
    await page.locator('#command-input').fill('deploy api');
    await page.getByRole('button', { name: 'Send command' }).click();
    await expect(page.locator('#command-result')).toContainText('Command accepted');
    await expect.poll(() => page.locator('#metric-requests .metric-value').innerText()).not.toBe(before);
  });

  test('adds and deletes activity', async ({ page }) => {
    await page.goto('/');
    await page.locator('#activity-input').fill('Browser verification complete');
    await page.getByRole('button', { name: 'Add note' }).click();
    const item = page.locator('#activity-list article').filter({ hasText: 'Browser verification complete' });
    await expect(item).toBeVisible();
    page.once('dialog', dialog => dialog.accept());
    await item.getByRole('button', { name: /Remove Browser verification complete/ }).click();
    await expect(item).toHaveCount(0);
  });

  test('edits the active profile inline', async ({ page }) => {
    await page.goto('/');
    await page.locator('#profile-input').fill('staging-west');
    await page.getByRole('button', { name: 'Save profile' }).click();
    await expect(page.locator('#profile-panel')).toContainText('staging-west');
  });

  test('updates polling status and loads the revealed panel', async ({ page }) => {
    await page.goto('/');
    const initialStatus = await page.locator('#health-status').innerText();
    await expect.poll(() => page.locator('#health-status').innerText(), { timeout: 12_000 }).not.toBe(initialStatus);
    await expect(page.locator('#health-status')).toContainText('CHECK');
    await page.locator('#lazy-panel').scrollIntoViewIfNeeded();
    await expect(page.locator('#lazy-panel')).toContainText('Architecture map is ready');
  });

  test('receives SSE events', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('#event-stream')).toContainText('Signal 1 received', { timeout: 8_000 });
  });

  test('boosts a server-rendered status link into its target', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('link', { name: 'Open live status' }).click();
    await expect(page.locator('#boost-target')).toContainText('All core routes responding');
  });

  test('explains every demo card through a server-rendered fragment', async ({ page }) => {
    await page.goto('/');
    const demos = ['telemetry', 'search', 'command', 'health', 'activity', 'profile', 'lazy', 'sse'];
    await expect(page.getByRole('button', { name: 'Explain' })).toHaveCount(demos.length);
    for (const demo of demos) {
      await page.locator(`button[hx-get="/demo/explain?demo=${demo}"]`).click();
      const explanation = page.locator(`#explain-${demo}`);
      await expect(explanation).toContainText('HTMX');
      await expect(explanation).toContainText('Server / Go');
      await expect(explanation).toContainText('Browser / client');
    }
  });

  test('has no horizontal overflow on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/');
    const dimensions = await page.evaluate(() => ({ width: document.documentElement.scrollWidth, viewport: window.innerWidth }));
    expect(dimensions.width).toBeLessThanOrEqual(dimensions.viewport);
  });

  test('does not emit browser console errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', message => {
      if (message.type() === 'error') errors.push(message.text());
    });
    await page.goto('/');
    await expect(page.locator('#event-stream')).toContainText('Signal 1 received', { timeout: 8_000 });
    expect(errors).toEqual([]);
  });

  test('returns 404 for a missing fragment', async ({ request }) => {
    const response = await request.get('/demo/missing');
    expect(response.status()).toBe(404);
    expect(await response.text()).not.toContain('Signal North');
  });
});
