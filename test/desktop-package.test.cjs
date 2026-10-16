const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const {packageDesktop}=require('../plugins/knit-desktop/package.cjs');

test('portable desktop package contains only distributable files and relative host paths',()=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'knit-package-'));
 try{
  const binary=path.join(dir,'daemon');fs.writeFileSync(binary,'isolated binary fixture');
  const {stage,archive}=packageDesktop(binary,dir,'darwin_arm64','1.0.0-rc.1');
  assert.deepEqual(fs.readdirSync(stage).sort(),['.agents','.codex-plugin','Install.command','LICENSE','README.md','bin','install-pilot.py','mcp.json','platform.json','plugin.json']);
  const mcp=JSON.parse(fs.readFileSync(path.join(stage,'mcp.json'))).mcpServers.knit_desktop;
  assert.equal(mcp.command,'${PLUGIN_ROOT}/bin/knit');
  assert.deepEqual(mcp.args,['mcp','--desktop']);
  assert.equal(mcp.env,undefined,'package must not embed developer paths or secrets');
  assert.equal(JSON.parse(fs.readFileSync(path.join(stage,'plugin.json'))).extensions['com.openai'].requires_local_executor,true);
  assert.ok(fs.statSync(archive).size>0);
  assert.throws(()=>packageDesktop(binary,dir,'../bad','1.0.0'));
  assert.throws(()=>packageDesktop(binary,dir,'darwin_arm64','../bad'));
  fs.writeFileSync(path.join(stage,'platform.json'),JSON.stringify({os:'unsupported',arch:'unsupported'}));
  const rejected=spawnSync('python3',[path.join(stage,'install-pilot.py')],{encoding:'utf8'});
  assert.equal(rejected.status,1);
  assert.match(rejected.stderr,/different operating system or processor/);
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
});
