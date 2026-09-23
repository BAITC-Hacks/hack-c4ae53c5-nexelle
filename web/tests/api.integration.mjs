// Real client -> Go server -> official dataset -> completion -> fresh HR analytics.
// Run against a fresh server: CQ_TEST_URL=http://127.0.0.1:8080 node --test tests/api.integration.mjs
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createClient } from '../src/services/api.ts';

test('UI client uses Go for profile, recommendation, completion and HR', async () => {
  const base = process.env.CQ_TEST_URL;
  assert.ok(base, 'CQ_TEST_URL must point to an isolated running server');
  const client = createClient(base);
  const before = await client.employee('ru');
  assert.equal(before.employee.employee_id, 'E0001');
  assert.equal(before.employee.name, 'Marat Yessenov');
  assert.ok(before.recommendations.length > 0);
  const hrBefore = await client.hr();
  assert.equal(hrBefore.total_employees, 200);
  const selected = before.recommendations.find(r => r.event_id !== 'EV_036');
  assert.ok(selected, 'fixture needs a nonrepeatable recommendation');
  const completed = await client.complete(selected.event_id, 'ru');
  assert.equal(completed.employee.history.length, before.employee.history.length + 1);
  assert.equal(completed.employee.history[0].status, 'completed');
  assert.ok(!completed.recommendations.some(r => r.event_id === selected.event_id));
  const after = await client.employee('kk');
  assert.equal(after.employee.history.length, completed.employee.history.length);
  assert.ok(after.recommendations.every(r => r.explanation.why.length > 0));
  for (const skill of completed.employee.skills) {
    const old = before.employee.skills.find(s => s.name === skill.name);
    if (old) assert.ok(skill.current_level >= old.current_level && skill.current_level <= 5);
  }
  await assert.rejects(client.complete(selected.event_id), /завершено/);
  const hrAfter = await client.hr();
  assert.equal(hrAfter.activity_participation.completed, hrBefore.activity_participation.completed + 1);
  const page = await fetch(base);
  assert.equal(page.status, 200);
  assert.match(await page.text(), /assets\/index-/);
});
