import test from 'node:test';
import assert from 'node:assert/strict';
import { deliverApprovedFeedback } from './delivery.mjs';
import { readFileSync } from 'node:fs';
import path from 'node:path';

test('installed plugin manifest points to valid MCP launch configuration', () => {
  const manifest = JSON.parse(readFileSync(new URL('./.codex-plugin/plugin.json', import.meta.url), 'utf8'));
  assert.equal(manifest.mcpServers, './.mcp.json');
  const config = JSON.parse(readFileSync(new URL(manifest.mcpServers, import.meta.url), 'utf8'));
  assert.ok(path.isAbsolute(config.mcpServers.knit_probe.command));
  assert.ok(path.isAbsolute(config.mcpServers.knit_probe.cwd));
  assert.match(path.basename(config.mcpServers.knit_probe.command), /^knit-desktop-probe(?:\.exe)?$/);
});

test('host acceptance does not claim agent completion', async () => {
  const messages = [];
  const result = await deliverApprovedFeedback({ sendFollowUpMessage: async p => messages.push(p) },
    { id: 'spike-1', approved: true, text: 'Reply KNIT_CONNECTION_OK. Do not edit files.' });
  assert.equal(messages.length, 1);
  assert.match(messages[0].prompt, /KNIT_CONNECTION_OK/);
  assert.equal(result.status, 'delivered');
  assert.equal(result.completion_confirmed, false);
});

test('missing host and unapproved feedback cannot be delivered', async () => {
  await assert.rejects(deliverApprovedFeedback({}, {}), /does not support/);
  let calls = 0;
  const host = { sendFollowUpMessage: async () => calls++ };
  await assert.rejects(deliverApprovedFeedback(host, {id:'x',text:'draft'}), /approve/);
  assert.equal(calls, 0);
});

test('host rejection is propagated without acknowledgement', async () => {
  await assert.rejects(deliverApprovedFeedback({ sendFollowUpMessage: async () => { throw Error('host disconnected'); } },
    { id:'x',text:'feedback',approved:true }), /host disconnected/);
});
