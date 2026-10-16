// Protocol + browser contract verification. This does not replace installed-host verification.
const { spawn } = require('node:child_process');
const { createInterface } = require('node:readline');
const assert = require('node:assert/strict');
const path = require('node:path');
const { chromium } = require('playwright');

async function main() {
  const binary = process.argv[2] || path.join(__dirname, 'bin/knit-desktop-probe');
  const child = spawn(binary, [], { stdio: ['pipe', 'pipe', 'pipe'] });
  const pending = new Map();
  let stderr = '';
  child.stderr.on('data', chunk => { stderr += chunk; });
  const lines = createInterface({ input: child.stdout });
  lines.on('line', line => {
    const response = JSON.parse(line);
    pending.get(response.id)?.(response);
  });
  let next = 0;
  function call(method, params) {
    const id = ++next;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => { pending.delete(id); reject(Error('MCP response timed out: ' + method)); }, 8000);
      pending.set(id, response => {
        clearTimeout(timer); pending.delete(id);
        if (response.error) reject(Error(JSON.stringify(response.error)));
        else resolve(response.result);
      });
      child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n');
    });
  }
  let browser;
  try {
    await call('initialize', {protocolVersion:'2025-06-18', capabilities:{}, clientInfo:{name:'knit-probe-test',version:'0.0.1'}});
    child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
    const tools = await call('tools/list', {});
    assert.equal(tools.tools.length, 1);
    const uri = tools.tools[0]._meta.ui.resourceUri;
    const result = await call('tools/call', {name:'open_connection_check',arguments:{}});
    assert.equal(result.structuredContent.status, 'awaiting_user_test');
    const resource = await call('resources/read', {uri});
    const html = resource.contents[0].text;
    assert.ok(html.includes('sendFollowUpMessage'));
    browser = await chromium.launch({headless:true, executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
    const page = await browser.newPage({viewport:{width:360,height:640}});
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.route('https://knit-probe.test/', route => route.fulfill({contentType:'text/html',body:html}));
    await page.goto('https://knit-probe.test/');
    assert.equal(await page.getByRole('button', {name:'Send test message'}).isDisabled(), true);
    assert.match(await page.locator('#status').innerText(), /not exposed/);
    await page.evaluate(() => {
      window.testMessages = [];
      window.openai = {sendFollowUpMessage: async message => window.testMessages.push(message)};
      window.dispatchEvent(new Event('openai:set_globals'));
    });
    await page.getByRole('button', {name:'Send test message'}).click();
    await page.waitForFunction(() => document.getElementById('status').textContent.startsWith('Host accepted'));
    assert.match(await page.locator('#status').innerText(), /Host accepted KNIT_DESKTOP_PROBE_/);
    const messages = await page.evaluate(() => window.testMessages);
    assert.equal(messages.length, 1);
    assert.match(messages[0].prompt, /Do not use tools or modify files/);
    assert.equal(await page.getByRole('button', {name:'Send test message'}).isDisabled(), true);
    await page.evaluate(() => {
      window.openai.sendFollowUpMessage = async () => { throw Error('host disconnected'); };
      window.dispatchEvent(new Event('openai:set_globals'));
    });
    await page.getByRole('button', {name:'Send test message'}).click();
    assert.match(await page.locator('#status').innerText(), /Delivery failed: host disconnected/);
    assert.deepEqual(errors, []);
    assert.equal(stderr, '');
    if (process.env.PROBE_SCREENSHOT) await page.screenshot({path:process.env.PROBE_SCREENSHOT,fullPage:true});
    console.log('PASS: installed binary MCP handshake, tool/resource discovery, isolated Chromium panel, missing-host state, one-shot host acceptance, and rejected delivery.');
    console.log('NOT VERIFIED: real Codex host panel delivery or agent response.');
  } finally {
    await browser?.close();
    lines.close(); child.stdin.end(); child.kill();
  }
}
main().catch(error => {console.error(error);process.exitCode=1;});
