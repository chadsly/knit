// Development installation: explicit, isolated daemon instance; no existing
// Knit configuration, daemon, or extension pairing is replaced.
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const root = __dirname;
const repo = path.resolve(root, '../..');
const data = path.join(root, 'runtime');
const command = path.join(root, 'bin', 'knit' + (process.platform === 'win32' ? '.exe' : ''));
fs.mkdirSync(path.dirname(command), {recursive:true});
fs.mkdirSync(data, {recursive:true,mode:0o700});
execFileSync('go', ['build', '-o', command, './cmd/daemon'], {cwd:repo,stdio:'inherit'});
fs.writeFileSync(path.join(root,'.mcp.json'),JSON.stringify({mcpServers:{knit_desktop:{
  command,args:['mcp'],cwd:repo,
  env:{KNIT_ADDR:'127.0.0.1:7778',KNIT_DATA_DIR:data,KNIT_CONFIG_PATH:path.join(data,'knit.toml'),KNIT_SQLITE_PATH:path.join(data,'knit.db')}
}}},null,2)+'\n',{mode:0o600});
execFileSync('codex',['plugin','add','knit-desktop@knit-development','--json'],{cwd:repo,stdio:'inherit'});
