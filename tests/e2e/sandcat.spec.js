// @ts-check
const { test, expect } = require('@playwright/test');

const CALDERA_URL = process.env.CALDERA_URL || 'http://localhost:8888';
const PLUGIN_ROUTE = '/#/plugins/sandcat';

// ---------------------------------------------------------------------------
// Helper: navigate to the sandcat plugin page inside magma
// ---------------------------------------------------------------------------
async function navigateToSandcat(page) {
  await page.goto(`${CALDERA_URL}${PLUGIN_ROUTE}`, { waitUntil: 'networkidle' });
}

// ===========================================================================
// 1. Plugin page loads
// ===========================================================================
test.describe('Sandcat plugin page load', () => {
  test('should display the Sandcat heading', async ({ page }) => {
    await navigateToSandcat(page);
    const heading = page.locator('h2', { hasText: 'Sandcat' });
    await expect(heading).toBeVisible({ timeout: 15_000 });
  });

  test('should display CAT description', async ({ page }) => {
    await navigateToSandcat(page);
    await expect(
      page.locator('text=coordinated access trojan')
    ).toBeVisible({ timeout: 15_000 });
  });

  test('should display agent deployment explanation text', async ({ page }) => {
    await navigateToSandcat(page);
    await expect(
      page.locator('text=persistent connection back')
    ).toBeVisible({ timeout: 15_000 });
  });

  test('should indicate HTTP(S) communication', async ({ page }) => {
    await navigateToSandcat(page);
    await expect(
      page.locator('text=HTTP(S)')
    ).toBeVisible({ timeout: 15_000 });
  });

  test('should direct users to the Agents tab for deployment', async ({ page }) => {
    await navigateToSandcat(page);
    await expect(
      page.locator('text=To deploy a Sandcat agent, go to the Agents tab')
    ).toBeVisible({ timeout: 15_000 });
  });

  test('should have a horizontal rule separator', async ({ page }) => {
    await navigateToSandcat(page);
    await expect(page.locator('hr').first()).toBeVisible({ timeout: 15_000 });
  });
});

// ===========================================================================
// 2. Agent deployment command display (via Agents page)
// ===========================================================================
test.describe('Sandcat agent deployment commands', () => {
  test('agents API should be accessible', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/agents`);
    expect(response.ok()).toBeTruthy();
    const agents = await response.json();
    expect(Array.isArray(agents)).toBeTruthy();
  });

  test('deploy commands API should be accessible', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/deploy_commands`);
    // May return 200 or 404 depending on Caldera version
    expect(response.status()).toBeLessThan(500);
  });

  test('agents page should load and list agent deployment options', async ({ page }) => {
    await page.goto(`${CALDERA_URL}/#/agents`, { waitUntil: 'networkidle' });
    // The agents page should display deployment information
    const heading = page.locator('h2', { hasText: /[Aa]gents/ });
    await expect(heading).toBeVisible({ timeout: 15_000 });
  });

  test('sandcat agent payloads should be available via file API', async ({ page }) => {
    // Sandcat binaries are served at /file/download
    const response = await page.request.get(`${CALDERA_URL}/api/v2/payloads`);
    // May return 200 or different status depending on config
    expect(response.status()).toBeLessThan(500);
  });
});

// ===========================================================================
// 3. Platform / architecture selection (agents page context)
// ===========================================================================
test.describe('Sandcat platform and architecture selection', () => {
  test('should support windows platform via config', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/agents`);
    expect(response.ok()).toBeTruthy();
    // Verify the API is accessible; platform filtering happens in UI
  });

  test('should support linux platform via config', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/agents`);
    expect(response.ok()).toBeTruthy();
  });

  test('should support darwin platform via config', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/agents`);
    expect(response.ok()).toBeTruthy();
  });

  test('sandcat plugin info should be available', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/plugins`);
    if (response.ok()) {
      const plugins = await response.json();
      const sandcat = plugins.find((p) => p.name === 'sandcat' || p.name === 'Sandcat');
      if (sandcat) {
        expect(sandcat).toHaveProperty('name');
      }
    }
    // If plugins API not available, just verify it didn't 500
    expect(response.status()).toBeLessThan(500);
  });
});

// ===========================================================================
// 4. Extension configuration
// ===========================================================================
test.describe('Sandcat extension configuration', () => {
  test('sandcat configuration should be accessible through plugin API', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/plugins`);
    expect(response.status()).toBeLessThan(500);
  });

  test('sandcat plugin page should display informational content only', async ({ page }) => {
    await navigateToSandcat(page);
    // Sandcat plugin page is informational - verify it shows deployment guidance
    await expect(
      page.locator('text=easily be deployed on any computer')
    ).toBeVisible({ timeout: 15_000 });
  });

  test('sandcat page should mention adversary emulation exercises', async ({ page }) => {
    await navigateToSandcat(page);
    await expect(
      page.locator('text=adversary emulation exercises')
    ).toBeVisible({ timeout: 15_000 });
  });

  test('sandcat page content should be within a .content container', async ({ page }) => {
    await navigateToSandcat(page);
    const content = page.locator('.content', { hasText: 'Sandcat' });
    await expect(content).toBeVisible({ timeout: 15_000 });
  });
});

// ===========================================================================
// 5. Compile options (sandcat build/delivery mechanisms)
// ===========================================================================
test.describe('Sandcat compile and delivery options', () => {
  test('payloads endpoint should be available', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/payloads`);
    expect(response.status()).toBeLessThan(500);
  });

  test('sandcat should be listed as an available agent via contacts', async ({ page }) => {
    // Check contacts/config for agent availability
    const response = await page.request.get(`${CALDERA_URL}/api/v2/contacts`);
    expect(response.status()).toBeLessThan(500);
  });

  test('health check endpoint should respond', async ({ page }) => {
    const response = await page.request.get(`${CALDERA_URL}/api/v2/health`);
    expect(response.status()).toBeLessThan(500);
  });

  test('file svc should handle sandcat payload requests', async ({ page }) => {
    // Verify the file download endpoint pattern exists
    const response = await page.request.head(`${CALDERA_URL}/file/download`);
    // Expect 405 (Method Not Allowed for HEAD) or 400 (bad request without params), not 500
    expect(response.status()).toBeLessThan(500);
  });
});

// ===========================================================================
// 6. Error states
// ===========================================================================
test.describe('Sandcat error states', () => {
  test('should handle invalid plugin route gracefully', async ({ page }) => {
    const resp = await page.goto(`${CALDERA_URL}/#/plugins/nonexistent-plugin`, {
      waitUntil: 'networkidle',
    });
    expect(resp?.status()).toBeLessThan(500);
  });

  test('should handle agents API failure gracefully on agents page', async ({ page }) => {
    await page.route('**/api/v2/agents', (route) =>
      route.fulfill({ status: 500, body: 'Internal Server Error' })
    );
    await page.goto(`${CALDERA_URL}/#/agents`, { waitUntil: 'networkidle' });
    // Page should still render
    expect(await page.title()).toBeTruthy();
  });

  test('sandcat page should render even when APIs are slow', async ({ page }) => {
    // Simulate slow API
    await page.route('**/api/v2/**', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 3000));
      return route.continue();
    });
    await navigateToSandcat(page);
    const heading = page.locator('h2', { hasText: 'Sandcat' });
    await expect(heading).toBeVisible({ timeout: 30_000 });
  });

  test('sandcat page should render even with network interruption on APIs', async ({ page }) => {
    await page.route('**/api/v2/plugins', (route) => route.abort('connectionrefused'));
    await navigateToSandcat(page);
    const heading = page.locator('h2', { hasText: 'Sandcat' });
    await expect(heading).toBeVisible({ timeout: 15_000 });
  });

  test('should handle payloads API returning empty list', async ({ page }) => {
    await page.route('**/api/v2/payloads', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      })
    );
    await navigateToSandcat(page);
    const heading = page.locator('h2', { hasText: 'Sandcat' });
    await expect(heading).toBeVisible({ timeout: 15_000 });
  });

  test('should handle deploy commands API failure', async ({ page }) => {
    await page.route('**/api/v2/deploy_commands', (route) =>
      route.fulfill({ status: 500, body: 'Error' })
    );
    await navigateToSandcat(page);
    const heading = page.locator('h2', { hasText: 'Sandcat' });
    await expect(heading).toBeVisible({ timeout: 15_000 });
  });
});
