// Isolated main-binary MCP/daemon verification; never attaches to user data.
const {spawn} = require('node:child_process');
const {createInterface} = require('node:readline');
const {mkdtempSync,rmSync} = require('node:fs');
const {tmpdir} = require('node:os');
const net = require('node:net');
const path = require('node:path');
const assert = require('node:assert/strict');

async function main() {
  const listener=net.createServer();
  await new Promise(resolve=>listener.listen(0,'127.0.0.1',resolve));
  const port=listener.address().port;
  await new Promise(resolve=>listener.close(resolve));
  const data=mkdtempSync(path.join(tmpdir(),'knit-desktop-mcp-'));
  const binary=process.argv[2] || path.join(__dirname,'bin','knit'+(process.platform==='win32'?'.exe':''));
  const child=spawn(binary,process.env.KNIT_VERIFY_DESKTOP==='1'?['mcp','--desktop']:['mcp'],{cwd:path.dirname(binary),env:{...process.env,KNIT_ADDR:`127.0.0.1:${port}`,KNIT_DATA_DIR:data,KNIT_CONFIG_PATH:path.join(data,'knit.toml'),KNIT_SQLITE_PATH:path.join(data,'knit.db')},stdio:['pipe','pipe','pipe']});
  const pending=new Map();let next=0,stderr='',stdout='';
  child.stderr.on('data',chunk=>{stderr+=chunk;});
  child.stdout.on('data',chunk=>{stdout+=chunk;});
  const lines=createInterface({input:child.stdout});
  lines.on('line',line=>{
    const response=JSON.parse(line);
    pending.get(response.id)?.(response);
  });
  function call(method,params) {
    const id=++next;
    return new Promise((resolve,reject)=>{
      const timer=setTimeout(()=>{pending.delete(id);reject(Error('Timed out: '+method));},20000);
      pending.set(id,response=>{clearTimeout(timer);pending.delete(id);response.error?reject(Error(JSON.stringify(response.error))):resolve(response.result);});
      child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n');
    });
  }
  try {
    await call('initialize',{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'knit-desktop-verification',version:'1'}});
    child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
    const listing=await call('tools/list',{});
    for(const name of ['open_desktop_feedback','desktop_panel_action','register_worker','knit_status'])assert.ok(listing.tools.some(tool=>tool.name===name));
    const opened=await call('tools/call',{name:'open_desktop_feedback',arguments:{}});
    assert.ok(!opened.isError);
    const resource=await call('resources/read',{uri:'ui://knit/desktop-feedback'});
    assert.match(resource.contents[0].text,/sendFollowUpMessage/);
    const connected=await call('tools/call',{name:'desktop_panel_action',arguments:{action:'connect',label:'Isolated verification',workspace:data}});
    assert.ok(!connected.isError);
    const id=connected.structuredContent.result.id;
    const beat=await call('tools/call',{name:'desktop_panel_action',arguments:{action:'heartbeat',binding_id:id}});
    assert.equal(beat.structuredContent.result.bindings[0].id,id);
    const pairing=await call('tools/call',{name:'desktop_panel_action',arguments:{action:'pair_browser'}});
    assert.ok(pairing.structuredContent.pairing_code);
    assert.equal(pairing.structuredContent.extension_token,undefined);
    assert.equal(pairing.structuredContent.token,undefined);
    const status=await call('tools/call',{name:'knit_status',arguments:{}});
    assert.equal(status.structuredContent.default_execution_mode,'mcp_worker');
    assert.equal(status.structuredContent.daemon_instance.daemon_instance_mode,'mcp_managed');
    assert.ok(stderr.includes('Chrome extension token:'));
    const secret=stderr.match(/Chrome extension token:\s*(\S+)/)?.[1];
    assert.ok(secret,'startup token missing from stderr');
    assert.ok(!stdout.includes(secret),'MCP stdout leaked startup token');
    child.stdin.end();
    await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('MCP did not shut down')),10000);child.once('exit',code=>{clearTimeout(timer);code===0?resolve():reject(Error('MCP exit '+code));});});
    await assert.rejects(fetch(`http://127.0.0.1:${port}/healthz`));
    console.log('PASS: main-binary MCP tools/resource, daemon-backed panel registration, short-code pairing, token confined to stderr, owned daemon shutdown.');
  } finally {
    if(child.exitCode===null){child.kill('SIGTERM');await new Promise(resolve=>child.once('exit',resolve));}
    lines.close();rmSync(data,{recursive:true,force:true});
  }
}
main().catch(error=>{console.error(error);process.exitCode=1;});
