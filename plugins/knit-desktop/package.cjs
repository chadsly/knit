// Build-only packaging tool. The installed package needs neither Node nor Go.
const fs = require('node:fs');
const path = require('node:path');
const {execFileSync} = require('node:child_process');

function packageDesktop(binary, output, target, version) {
  if (!/^(darwin|linux|windows)_(amd64|arm64)$/.test(target)) throw Error('Invalid target');
  if (!/^\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?$/.test(version)) throw Error('Invalid version');
  if (!fs.statSync(binary).isFile()) throw Error('Expected a built Knit daemon binary');
  const stage=path.resolve(output,`knit-desktop-${version}-${target}`);
  fs.mkdirSync(stage,{recursive:false});
  fs.mkdirSync(path.join(stage,'bin'));
  const [os,arch]=target.split('_');
  fs.writeFileSync(path.join(stage,'platform.json'),JSON.stringify({os,arch})+'\n');
  fs.mkdirSync(path.join(stage,'.codex-plugin'));
  fs.writeFileSync(path.join(stage,'.codex-plugin','plugin.json'),JSON.stringify({
    name:'knit-desktop',version,description:'Browser feedback in your Codex conversation',
    mcpServers:'./mcp.json',requires_local_executor:true,
    interface:{displayName:'Knit Browser Feedback',shortDescription:'Browser feedback in this conversation'}
  },null,2)+'\n');
  fs.mkdirSync(path.join(stage,'.agents','plugins'),{recursive:true});
  fs.writeFileSync(path.join(stage,'.agents','plugins','marketplace.json'),JSON.stringify({
    name:'knit-private-pilot',interface:{displayName:'Knit Private Pilot'},plugins:[{
      name:'knit-desktop',source:{source:'local',path:'.'},
      policy:{installation:'AVAILABLE',authentication:'ON_INSTALL'},category:'Developer Tools'
    }]
  },null,2)+'\n');
  const executable=target.startsWith('windows_')?'knit.exe':'knit';
  fs.copyFileSync(binary,path.join(stage,'bin',executable));
  fs.copyFileSync(path.join(__dirname,'install-pilot.py'),path.join(stage,'install-pilot.py'));
  fs.writeFileSync(path.join(stage,'Install.command'),'#!/bin/sh\nset -eu\ncd "$(dirname "$0")"\npython3 ./install-pilot.py\nprintf "\\nInstalled. Restart Codex, then open Knit in the conversation side panel.\\n"\n');
  fs.chmodSync(path.join(stage,'Install.command'),0o755);
  fs.chmodSync(path.join(stage,'bin',executable),0o755);
  fs.writeFileSync(path.join(stage,'plugin.json'),JSON.stringify({
    name:'knit-desktop',version,description:'Browser feedback in your Codex conversation',
    license:'MIT',extensions:{'com.openai':{requires_local_executor:true}}
  },null,2)+'\n');
  fs.writeFileSync(path.join(stage,'mcp.json'),JSON.stringify({mcpServers:{knit_desktop:{
    type:'stdio',command:'${PLUGIN_ROOT}/bin/'+executable,args:['mcp','--desktop'],cwd:'${PLUGIN_ROOT}'
  }}},null,2)+'\n');
  fs.writeFileSync(path.join(stage,'README.md'),`# Knit Desktop ${version}\n\nTarget: ${target}. Install this plugin through the supported desktop plugin installation flow. Public local-MCP distribution requires OpenAI approval.\n\nOpen Knit beside your Codex conversation, connect once, and pair the browser extension using the short-lived code. Keep the persistent panel open while reviewing. Your Codex permissions remain in Codex.\n\nThe package contains its native Knit binary; no Go, Node, source checkout, or manual MCP configuration is required by this package. Video frame review additionally requires FFmpeg on the daemon host PATH. Local speech-to-text requires Python 3 and downloads its managed runtime/model when enabled. These dependencies are not bundled or silently installed by this package.\n\nDesktop state is stored in the OS user configuration directory under knit/desktop, outside the versioned plugin folder. Existing normal Knit instances are not converted or stopped. Port 7778 must be available or already serve a compatible MCP-managed daemon. An incompatible instance produces an error.\n`);
  fs.copyFileSync(path.resolve(__dirname,'../../LICENSE'),path.join(stage,'LICENSE'));
  fs.appendFileSync(path.join(stage,'README.md'),'\nPrivate pilot: install the official Codex CLI on PATH and Python 3 first. On macOS open Install.command; on Linux/Windows run python3 install-pilot.py. Restart Codex after installation. The installer registers the included local marketplace and resolves cached binary paths for older hosts without manual config edits. Do not bypass OS warnings or delete an existing Knit profile.\n');
  const archive=stage+'.zip';
  if(fs.existsSync(archive)) throw Error('Refusing to overwrite an existing archive');
  execFileSync('zip',['-q','-r',archive,'.'],{cwd:stage});
  return {stage,archive};
}
module.exports={packageDesktop};
if(require.main===module){
 const [binary,output,target,version]=process.argv.slice(2);
 if(!version)throw Error('Usage: node package.cjs <binary> <existing-output-dir> <os_arch> <version>');
 console.log(JSON.stringify(packageDesktop(path.resolve(binary),path.resolve(output),target,version)));
}
