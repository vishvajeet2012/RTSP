import { expect, test } from '@playwright/test';

const firstURL = process.env.TEST_RTSP_URL;
const secondURL = process.env.TEST_RTSP_URL_2;
const apiOrigin = (process.env.TEST_API_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '');
const accessToken = process.env.TEST_API_TOKEN;
const headers: Record<string, string> = accessToken
  ? { Authorization: `Bearer ${accessToken}` }
  : {};

test('real live video, multiple streams, pause, restart, stop, remove and mobile layout', async ({
  page,
  request,
}) => {
  test.skip(
    !firstURL || !secondURL,
    'Set TEST_RTSP_URL and TEST_RTSP_URL_2 to running RTSP publishers.',
  );
  const existing = (await (await request.get(`${apiOrigin}/api/streams`, { headers })).json()) as {
    id: string;
  }[];
  for (const stream of existing)
    await request.delete(`${apiOrigin}/api/streams/${stream.id}`, { headers });
  const browserErrors: string[] = [];
  page.on('pageerror', (error) => browserErrors.push(error.message));
  await page.goto('/');
  if (accessToken) {
    await page.getByRole('button', { name: 'Configure backend access token' }).click();
    await page.getByLabel('Backend access token', { exact: true }).fill(accessToken);
    await page.getByRole('button', { name: 'Connect workspace', exact: true }).click();
  }
  await expect(page.getByText('No active streams', { exact: true })).toBeVisible();
  await page.getByLabel('RTSP URL', { exact: true }).fill('https://invalid.example');
  await page.getByRole('button', { name: 'Add stream', exact: true }).click();
  await expect(page.getByText('Use an rtsp:// or rtsps:// URL.')).toBeVisible();
  for (const [name, url] of [
    ['Office camera', firstURL!],
    ['Entrance camera', secondURL!],
  ]) {
    await page.getByLabel('Stream name', { exact: false }).fill(name);
    await page.getByLabel('RTSP URL', { exact: true }).fill(url);
    await page.getByRole('button', { name: 'Add stream', exact: true }).click();
    await expect(
      page.getByRole('article', { name: `${name} stream` }).locator('[data-player-status="live"]'),
    ).toBeVisible();
  }
  const first = page.getByRole('article', { name: 'Office camera stream' });
  const canvas = first.locator('canvas');
  const pixels = await canvas.evaluate((element) => {
    const c = element as HTMLCanvasElement;
    const bytes = c.getContext('2d')!.getImageData(0, 0, c.width, c.height).data;
    const colors = new Set<number>();
    for (let i = 0; i < bytes.length; i += 160)
      colors.add((bytes[i] << 16) | (bytes[i + 1] << 8) | bytes[i + 2]);
    return { width: c.width, colors: colors.size };
  });
  expect(pixels.width).toBeGreaterThan(300);
  expect(pixels.colors).toBeGreaterThan(20);
  await expect(page.getByText('2 connected', { exact: true })).toBeVisible();
  await expect(page.locator('[data-sonner-toast]')).toHaveCount(0);
  await page.screenshot({ path: 'docs/dashboard.png', fullPage: true });
  await first.getByRole('button', { name: 'Pause Office camera', exact: true }).click();
  await expect(first.locator('[data-player-status="paused"]')).toBeVisible();
  const paused = await canvas.evaluate((element) => (element as HTMLCanvasElement).toDataURL());
  await page.waitForTimeout(1200);
  expect(await canvas.evaluate((element) => (element as HTMLCanvasElement).toDataURL())).toBe(
    paused,
  );
  await expect(
    page
      .getByRole('article', { name: 'Entrance camera stream' })
      .locator('[data-player-status="live"]'),
  ).toBeVisible();
  await first.getByRole('button', { name: 'Play Office camera', exact: true }).click();
  await expect(first.locator('[data-player-status="live"]')).toBeVisible();
  await first.getByRole('button', { name: 'Restart Office camera', exact: true }).click();
  await expect(first.locator('[data-player-status="live"]')).toBeVisible();
  await first
    .getByRole('button', { name: 'Stop source Office camera for all viewers', exact: true })
    .click();
  await expect(first.locator('[data-player-status="stopped"]')).toBeVisible();
  await first.getByRole('button', { name: 'Play Office camera', exact: true }).click();
  await expect(first.locator('[data-player-status="live"]')).toBeVisible();
  for (const width of [1920, 1440, 1024, 768, 375]) {
    await page.setViewportSize({ width, height: 900 });
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true);
    const columns = await page
      .locator('.stream-grid')
      .evaluate((grid) => getComputedStyle(grid).gridTemplateColumns.split(' ').length);
    expect(columns).toBe(width >= 768 ? 2 : 1);
  }
  await expect(page.getByText('2 connected', { exact: true })).toBeVisible();
  await expect(page.locator('[data-sonner-toast]')).toHaveCount(0);
  await page.screenshot({ path: 'docs/mobile.png', fullPage: true });
  await first.getByRole('button', { name: 'Remove Office camera', exact: true }).click();
  await expect(first).toHaveCount(0);
  await page.getByRole('button', { name: 'Remove Entrance camera', exact: true }).click();
  await expect(page.getByText('No active streams', { exact: true })).toBeVisible();
  expect(browserErrors).toEqual([]);
});
