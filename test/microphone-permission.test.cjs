const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const source = fs.readFileSync(path.join(__dirname, '../extension/chromium/microphone.js'), 'utf8');

for (const denied of [false, true]) {
  test('microphone setup requires a click and handles ' + (denied ? 'denial' : 'success without recording'), async () => {
    const elements = { enable: {}, result: {} };
    let requests = 0, stops = 0;
    vm.runInNewContext(source, {
      document: { getElementById: id => elements[id] },
      navigator: { mediaDevices: { getUserMedia: async options => {
        requests++;
        assert.equal(options.audio, true);
        assert.equal(options.video, false);
        if (denied) throw Error('Permission denied');
        return { getTracks: () => [{ stop: () => stops++ }] };
      } } }
    });
    assert.equal(requests, 0, 'no automatic microphone access on page load');
    await elements.enable.onclick();
    assert.equal(requests, 1);
    assert.equal(stops, denied ? 0 : 1, 'permission stream must be stopped');
    assert.equal(elements.enable.disabled, false);
    assert.match(elements.result.textContent, denied ? /not allowed/ : /access allowed/);
  });
}
