import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createClient } from '../src/services/api.ts';

test('HTTP errors are propagated without a mock fallback', async () => {
  const client = createClient('', 'E0001', async () => new Response(
    JSON.stringify({ error: { message: 'Сотрудник не найден' } }), { status: 404 },
  ));
  await assert.rejects(client.employee(), /Сотрудник не найден/);
});

test('employee identity and locale are sent to Go', async () => {
  const requests = [];
  const client = createClient('', 'E0002', async (url, options) => {
    requests.push({ url, options });
    return new Response('{}', { status: 500 });
  });
  await assert.rejects(client.employee('kk'));
  assert.ok(requests.some(r => r.url.includes('/E0002/profile?locale=kk')));
  for (const { options } of requests) {
    assert.equal(options.headers['X-Demo-Role'], 'employee');
    assert.equal(options.headers['X-Employee-ID'], 'E0002');
  }
});
