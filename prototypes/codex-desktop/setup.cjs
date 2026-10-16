// Generate machine-local launch paths rather than relying on host-specific interpolation.
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const root = __dirname;
const repo = path.resolve(root, '../..');
const command = path.join(root, 'bin', 'knit-desktop-probe' + (process.platform === 'win32' ? '.exe' : ''));
fs.mkdirSync(path.dirname(command), {recursive:true});
execFileSync('go', ['build', '-o', command, './prototypes/codex-desktop/server'], {cwd:repo,stdio:'inherit'});
fs.writeFileSync(path.join(root, '.mcp.json'), JSON.stringify({mcpServers:{knit_probe:{command,cwd:root}}}, null, 2)+'\n');
execFileSync('codex', ['plugin','add','knit-desktop-probe@knit-development','--json'], {cwd:repo,stdio:'inherit'});
