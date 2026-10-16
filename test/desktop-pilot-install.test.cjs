const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const net=require('node:net');
const {execFileSync}=require('node:child_process');
const {packageDesktop}=require('../plugins/knit-desktop/package.cjs');

test('private pilot installs into isolated Codex home without repository configuration',{
 skip:!process.env.KNIT_PILOT_TEST_BINARY
},async()=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'knit-pilot-install-'));
 try {
  const target=(process.platform==='win32'?'windows':process.platform)+'_'+(process.arch==='x64'?'amd64':process.arch);
  const executable=process.platform==='win32'?'knit.exe':'knit';
  const {stage}=packageDesktop(process.env.KNIT_PILOT_TEST_BINARY,dir,target,'1.0.0-rc.3');
  const home=path.join(dir,'codex-home');fs.mkdirSync(home);
  const listener=net.createServer();
  await new Promise(resolve=>listener.listen(0,'127.0.0.1',resolve));
  const port=listener.address().port;
  await new Promise(resolve=>listener.close(resolve));
  const data=path.join(dir,'runtime');fs.mkdirSync(data);
  const options={cwd:dir,env:{...process.env,CODEX_HOME:home,KNIT_ADDR:`127.0.0.1:${port}`,KNIT_DATA_DIR:data,
    KNIT_CONFIG_PATH:path.join(data,'knit.toml'),KNIT_SQLITE_PATH:path.join(data,'knit.db')},encoding:'utf8'};
  const installed=JSON.parse(execFileSync('python3',[path.join(stage,'install-pilot.py')],options));
  assert.equal(installed.name,'knit-desktop');
  assert.equal(installed.version,'1.0.0-rc.3');
  assert.ok(fs.realpathSync(installed.installedPath).startsWith(fs.realpathSync(home)),installed.installedPath);
  assert.ok(fs.existsSync(path.join(installed.installedPath,'bin',executable)));
  const mcp=fs.readFileSync(path.join(installed.installedPath,'mcp.json'),'utf8');
  assert.ok(!mcp.includes(process.cwd()),'must not depend on source checkout');
  assert.equal(JSON.parse(mcp).mcpServers.knit_desktop.command,path.join(fs.realpathSync(installed.installedPath),'bin',executable));
  const diagnostic=path.resolve(__dirname,'../prototypes/codex-desktop/check-runtime.py');
  const output=execFileSync('python3',[diagnostic],{...options,timeout:60000,env:{...options.env,
    KNIT_DIAGNOSTIC_TOOL:'knit_status',KNIT_DIAGNOSTIC_SERVER:'knit_desktop',KNIT_ADDR:`127.0.0.1:${port}`,KNIT_DATA_DIR:data,
    KNIT_CONFIG_PATH:path.join(data,'knit.toml'),KNIT_SQLITE_PATH:path.join(data,'knit.db')
  }});
  const servers=JSON.parse(output);
  assert.ok(servers[0].tools.open_desktop_feedback || JSON.stringify(servers[0].tools).includes('open_desktop_feedback'));
  await assert.rejects(fetch(`http://127.0.0.1:${port}/healthz`),'diagnostic must not leave an owned daemon behind');
 } finally {fs.rmSync(dir,{recursive:true,force:true});}
});
