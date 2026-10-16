// Real daemon HTTP + browser panel contract. The desktop host is injected;
// this does not replace verification in the installed Codex application.
const { chromium } = require('playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { randomUUID } = require('node:crypto');

async function main() {
  const base = process.env.KNIT_TEST_URL;
  const token = process.env.KNIT_TEST_TOKEN;
  assert.ok(base && token, 'Run through TestDesktopBrowserWorkflow');
  async function post(endpoint, body) {
    const res = await fetch(base + endpoint, {method:'POST', headers:{'Content-Type':'application/json','X-Knit-Token':token,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()}, body:JSON.stringify(body)});
    const data = await res.json();
    if (!res.ok) throw Error(data.error?.message || data.message || JSON.stringify(data));
    return data;
  }
  const browser = await chromium.launch({headless:true, executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try {
    const page = await browser.newPage({viewport:{width:360,height:740}});
    const errors = [];
    page.on('pageerror', e=>errors.push(e.message));
    await page.route('https://knit-panel.test/', route=>route.fulfill({contentType:'text/html',body:fs.readFileSync(path.join(__dirname,'panel.html'),'utf8')}));
    await page.exposeFunction('daemonAction', async input=>input.action==='pair_browser'?{...await post('/api/extension/pair/start',{name:'Browser E2E',browser:'chromium'}),daemon_url:base}:post('/api/desktop/panel',input));
    await page.exposeFunction('daemonCapture',input=>post('/api/desktop/capture',input));
    await page.goto('https://knit-panel.test/');
    assert.equal(await page.locator('#connect').isDisabled(),true);
    const installHost=workspace=>{
      window.sent = [];
      window.reviewedCaptures=[];
      window.openai = {
        toolOutput:{workspace},
        callTool:async(name,input)=>{
          if(input.action==='heartbeat'&&window.failHeartbeat)throw Error('Knit connection unavailable');
          return {structuredContent:await window.daemonAction(name==='desktop_delivery_settings'?{...input,action:'delivery_settings'}:input)};
        },
        setWidgetState:async state=>{await new Promise(resolve=>setTimeout(resolve,20));window.openai.widgetState=state;},
        sendFollowUpMessage:async message=>{
          window.sent.push(message);
          const ids=message.prompt.match(/binding_id=([^,]+), delivery_id=([^\.]+)\./);
          await window.daemonAction({action:'report',binding_id:ids[1],delivery_id:ids[2],status:'working'});
          const request=JSON.parse(message.prompt.split('\n')[1]);
          for(const capture of request.artifacts){const context=await window.daemonCapture({binding_id:ids[1],delivery_id:ids[2],capture_index:capture.capture_index});window.reviewedCaptures.push({mime_type:context.mime_type,size:context.data.length});}
          await window.daemonAction({action:'report',binding_id:ids[1],delivery_id:ids[2],status:'completed',result:'Updated checkout spacing; tests passed.'});
        }
      };
      window.dispatchEvent(new Event('openai:set_globals'));
    };
    await page.evaluate(installHost,process.env.KNIT_TEST_WORKSPACE);
    assert.equal(await page.locator('#keepVisible').isVisible(),false);
    assert.match(await page.locator('#displayHint').innerText(),/does not offer a persistent connection view/);
    await page.evaluate(()=>{
      window.openai.requestDisplayMode=async()=>{throw Error('Host declined');};
      window.dispatchEvent(new Event('openai:set_globals'));
    });
    await page.locator('#keepVisible').click();
    await page.waitForFunction(()=>document.getElementById('status').textContent.includes('did not confirm a floating or expanded view'));
    await page.evaluate(()=>{
      window.openai.requestDisplayMode=async({mode})=>{
        window.requestedMode=mode;
        setTimeout(()=>{window.openai.displayMode=mode;window.dispatchEvent(new CustomEvent('openai:set_globals',{detail:{globals:{displayMode:mode}}}));},100);
      };
    });
    await page.locator('#keepVisible').click();
    await page.waitForFunction(()=>window.openai.displayMode==='pip');
    assert.equal(await page.evaluate(()=>window.requestedMode),'pip');
    assert.equal(await page.locator('#keepVisible').isDisabled(),true);
    await page.evaluate(()=>{
      window.openai.displayMode='inline';window.modeRequests=[];
      window.openai.requestDisplayMode=async({mode})=>{
        window.modeRequests.push(mode);
        if(mode==='pip')return {mode:'inline'};
        window.openai.displayMode=mode;return {mode};
      };
      window.dispatchEvent(new Event('openai:set_globals'));
    });
    await page.locator('#keepVisible').click();
    await page.waitForFunction(()=>window.openai.displayMode==='fullscreen');
    assert.deepEqual(await page.evaluate(()=>window.modeRequests),['pip','fullscreen']);
    await page.locator('#label').fill('Checkout fixes');
    await page.locator('#connect').click();
    await page.locator('#connected').waitFor({state:'visible'});
    const binding=await page.evaluate(()=>window.openai.widgetState.binding_id);
    await page.locator('#handoff').waitFor({state:'visible'});
    const connectionCode=await page.locator('#connectionCode').inputValue();
    assert.ok(connectionCode.startsWith('knit-connect:'));
    assert.equal(await page.locator('#automaticDelivery').isChecked(),false);
    await page.locator('#automaticDelivery').check();
    // Load the actual extension side-panel assets. Only Chrome's platform APIs
    // are shimmed; all session/approval/submit calls reach the real daemon.
    const recorder=await browser.newPage({viewport:{width:360,height:740}});
    recorder.on('pageerror',e=>errors.push(e.message));
    const extensionRoot=path.resolve(__dirname,'../../extension/chromium');
    await recorder.route(base+'/__test_extension/**',route=>{
      const name=new URL(route.request().url()).pathname.split('/').pop();
      if(name==='icon32.png')return route.fulfill({contentType:'image/png',body:fs.readFileSync(path.join(extensionRoot,'icons/icon32.png'))});
      if(!['recorder.html','recorder.js','desktop-connect.js','recorder.css'].includes(name))return route.abort();
      route.fulfill({contentType:name.endsWith('.html')?'text/html':name.endsWith('.js')?'application/javascript':'text/css',body:fs.readFileSync(path.join(extensionRoot,name))});
    });
    await recorder.addInitScript(({base,token})=>{
      const stored={};
      let config={daemonBaseURL:base,daemonToken:''};
      window.chrome={
        runtime:{sendMessage:async message=>{if(message.type==='knit:get-config')return config;if(message.type==='knit:set-config')config=message.payload;return {};}},
        storage:{local:{get:async()=>stored,set:async values=>Object.assign(stored,values),remove:async keys=>{for(const key of keys)delete stored[key];}}}
      };
    },{base});
    await recorder.goto(base+'/__test_extension/recorder.html');
    await recorder.locator('#desktopConnectionCode').fill(connectionCode);
    if(process.env.KNIT_ONBOARDING_SCREENSHOT)await recorder.screenshot({path:process.env.KNIT_ONBOARDING_SCREENSHOT,fullPage:true});
    await recorder.locator('#desktopConnectConfirm').click();
    await recorder.getByRole('button',{name:'End review',exact:true}).waitFor({state:'visible'});
    assert.equal(await recorder.locator('#desktopConnectSetup').isVisible(),false);
    assert.equal(await recorder.locator('#sessionState').innerText(),'Session active');
    await recorder.getByRole('button',{name:'End review',exact:true}).click();
    await recorder.getByRole('button',{name:'Start review',exact:true}).waitFor({state:'visible'});
    assert.equal(await recorder.locator('#sessionState').innerText(),'Session idle');
    assert.equal(await recorder.locator('#desktopConnectSetup').isVisible(),false);
    await recorder.getByRole('button',{name:'Start review',exact:true}).click();
    await recorder.getByRole('button',{name:'End review',exact:true}).waitFor({state:'visible'});
    if(process.env.KNIT_CONNECTED_SCREENSHOT)await recorder.screenshot({path:process.env.KNIT_CONNECTED_SCREENSHOT,fullPage:true});
    assert.equal(await recorder.locator('#desktopDestination').inputValue(),binding);
    const browserConfig=await recorder.evaluate(()=>chrome.runtime.sendMessage({type:'knit:get-config'}));
    const forbidden=await fetch(base+'/api/desktop/panel',{method:'POST',headers:{'Content-Type':'application/json','X-Knit-Token':browserConfig.daemonToken,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:JSON.stringify({action:'heartbeat',binding_id:binding})});
    assert.equal(forbidden.status,403,'extension credential must not control desktop panels');
    const captureDenied=await fetch(base+'/api/desktop/capture',{method:'POST',headers:{'Content-Type':'application/json','X-Knit-Token':browserConfig.daemonToken,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:JSON.stringify({binding_id:binding,delivery_id:'unclaimed',capture_index:0})});
    assert.equal(captureDenied.status,403,'extension credential must not retrieve model capture payloads');
    const firstNote=new FormData();
    await post('/api/companion/pointer',{x:100,y:100,event_type:'click',url:'https://example.com',target_tag:'button',target_id:'primary-action',target_selector:'#primary-action'});
    firstNote.append('raw_transcript','Fix checkout spacing');firstNote.append('normalized','Fix checkout spacing');
    firstNote.append('screenshot',new Blob([Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wl6LsAAAAAASUVORK5CYII=','base64')],{type:'image/png'}),'snapshot.png');
    const noteResponse=await fetch(base+'/api/session/feedback/note',{method:'POST',headers:{'X-Knit-Token':token,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:firstNote});
    assert.equal(noteResponse.status,200);
    await recorder.evaluate(()=>refreshSession(true));
    await recorder.locator('#submitBtn').click();
    await recorder.waitForFunction(()=>document.getElementById('submitNotice').textContent.includes('Queued for your Codex panel'));
    const publicState=await fetch(base+'/api/desktop',{headers:{'X-Knit-Token':token}}).then(r=>r.json());
    const queued=publicState.deliveries[0];
    assert.equal(queued.status,'queued');
    await page.waitForFunction(()=>document.getElementById('deliveries').textContent.includes('completed'));
    await recorder.evaluate(()=>refreshSession(true));
    assert.match(await recorder.locator('#desktopDestinationHint').innerText(),/automatically/);
    assert.equal(await page.evaluate(()=>window.sent.length),1);
    assert.equal(await page.evaluate(()=>window.sent[0].scrollToBottom),false,'delivery must not scroll its own connection off screen');
    assert.equal(await page.evaluate(()=>window.reviewedCaptures[0].mime_type),'image/png');
    assert.deepEqual(await page.evaluate(()=>window.reviewedCaptures.map(item=>item.mime_type)),['image/png','application/json','text/plain']);
    assert.equal(await page.evaluate(()=>window.sent[0].prompt.includes('.enc')),false);
    assert.match(await page.evaluate(()=>window.sent[0].prompt),/Intended workspace:/);
    assert.match(await page.locator('#deliveries').innerText(),/Updated checkout spacing; tests passed/);
    assert.equal(await page.locator('#status').innerText(),'');
    assert.equal(await page.getByRole('button',{name:'Send to this conversation',exact:true}).count(),0);
    await recorder.evaluate(()=>{
      Object.defineProperty(navigator.mediaDevices,'getUserMedia',{configurable:true,value:async()=>{throw new DOMException('Permission denied','NotAllowedError');}});
      chrome.runtime.getURL=name=>'chrome-extension://test/'+name;
      chrome.tabs={query:async()=>[{id:1,url:'https://example.com',title:'Test page'}],create:async options=>{window.permissionSetupURL=options.url;}};
      chrome.scripting={executeScript:async()=>[{result:{url:'https://example.com',title:'Test page'}}]};
    });
    await recorder.getByRole('button',{name:'Record a voice note for the current page.',exact:true}).click();
    await recorder.getByRole('button',{name:'Enable microphone',exact:true}).waitFor({state:'visible'});
    await recorder.getByRole('button',{name:'Enable microphone',exact:true}).click();
    assert.equal(await recorder.evaluate(()=>window.permissionSetupURL),'chrome-extension://test/microphone.html');
    // Reopening uses Chrome storage in production; this harness retains the
    // page to verify periodic status updates without resetting its API shim.
    await recorder.evaluate(()=>refreshSession(true));
    await recorder.waitForFunction(()=>document.getElementById('desktopDestination').options.length>1);
    await recorder.locator('#desktopDestination').selectOption(binding);
    await recorder.waitForFunction(()=>document.getElementById('queueIndicator').textContent.includes('completed'));
    const requestLink=await recorder.locator('#queueIndicator a').getAttribute('href');
    for(const theme of ['light','dark']){
      const colors=await recorder.evaluate(theme=>{
        document.documentElement.dataset.theme=theme;
        const link=document.querySelector('#queueIndicator a');
        const style=getComputedStyle(link);
        const rgb=value=>value.match(/\d+(?:\.\d+)?/g).slice(0,3).map(Number);
        const luminance=values=>values.map(v=>{v/=255;return v<=0.04045?v/12.92:((v+0.055)/1.055)**2.4;}).reduce((sum,v,i)=>sum+v*[0.2126,0.7152,0.0722][i],0);
        const hex=getComputedStyle(document.documentElement).getPropertyValue('--bg').trim();
        const background=[1,3,5].map(i=>parseInt(hex.slice(i,i+2),16));
        const a=luminance(rgb(style.color)),b=luminance(background);
        return {color:style.color,parent:getComputedStyle(link.parentElement).color,contrast:(Math.max(a,b)+0.05)/(Math.min(a,b)+0.05)};
      },theme);
      assert.equal(colors.color,colors.parent,'request links must inherit readable theme color');
      assert.ok(colors.contrast>=4.5,theme+' request link contrast must meet WCAG AA');
      await recorder.keyboard.press('Tab');
      await recorder.locator('#queueIndicator a').first().focus();
      assert.equal(await recorder.locator('#queueIndicator a').first().evaluate(link=>getComputedStyle(link).outlineStyle),'solid');
      await recorder.locator('#queueIndicator a').first().evaluate(link=>link.blur());
    }
    const mainUI=await browser.newPage();
    mainUI.on('pageerror',e=>errors.push(e.message));
    await mainUI.goto(requestLink);
    await mainUI.waitForFunction(()=>document.getElementById('desktopConnections').textContent.includes('Updated checkout spacing; tests passed.'));
    assert.equal(await mainUI.locator('#desktopConnections').evaluate(root=>root.closest('details').open),true);
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
    if(process.env.KNIT_PANEL_SCREENSHOT)await page.screenshot({path:process.env.KNIT_PANEL_SCREENSHOT,fullPage:true});
    // Native follow-up turns may remount the widget before its saved state arrives.
    // Restore only this widget's binding, never choose a destination from a list.
    await page.reload();
    await page.evaluate(installHost,process.env.KNIT_TEST_WORKSPACE);
    assert.equal(await page.locator('#connectForm').isVisible(),true);
    await page.evaluate(binding=>{
      window.openai.widgetState={binding_id:binding};
      window.dispatchEvent(new CustomEvent('openai:set_globals',{detail:{globals:{widgetState:{binding_id:binding}}}}));
    },binding);
    await page.locator('#connected').waitFor({state:'visible'});
    await page.waitForFunction(()=>document.getElementById('automaticDelivery').checked);
    for(let iteration=0;iteration<2;iteration++){
      // A cached panel must resume polling after pagehide/pageshow too.
      await page.evaluate(()=>{
        window.dispatchEvent(new PageTransitionEvent('pagehide',{persisted:true}));
        window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}));
      });
      const repeatNote=new FormData();
      repeatNote.append('raw_transcript','Consecutive correction '+iteration);
      repeatNote.append('normalized','Consecutive correction '+iteration);
      const repeatResponse=await fetch(base+'/api/session/feedback/note',{method:'POST',headers:{'X-Knit-Token':token,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:repeatNote});
      assert.equal(repeatResponse.status,200);
      await recorder.evaluate(()=>refreshSession(true));
      await recorder.locator('#submitBtn').click();
      await page.waitForFunction(count=>window.sent.length===count,iteration+1);
      await page.waitForFunction(()=>!busy);
      assert.equal(await page.locator('#connectForm').isVisible(),false);
      assert.equal(await page.evaluate(()=>window.openai.widgetState.binding_id),binding);
      const repeatState=await fetch(base+'/api/desktop',{headers:{'X-Knit-Token':token}}).then(r=>r.json());
      assert.equal(repeatState.bindings.length,1,'must not create replacement connections');
      assert.equal(repeatState.deliveries.filter(item=>item.status==='completed').length,iteration+2);
    }
    await page.evaluate(()=>{window.failHeartbeat=true;});
    await page.locator('#refresh').click();
    await page.waitForFunction(()=>document.getElementById('status').textContent.includes('Queued feedback is retained'));
    assert.equal(await page.evaluate(()=>window.openai.widgetState.binding_id),binding);
    await page.evaluate(()=>{window.failHeartbeat=false;});
    await page.locator('#refresh').click();
    await page.waitForFunction(()=>!document.getElementById('status').classList.contains('error'));
    // Reported execution failures must be readable without hover in every UI.
    await page.evaluate(()=>{window.openai.sendFollowUpMessage=async()=>{throw Error('Unexpected automatic send during failure fixture');};});
    await post('/api/desktop/panel',{action:'delivery_settings',binding_id:binding,automatic_delivery:false});
    for(const [index,message] of [
      'Wrong workspace: select the matching conversation and workspace before retrying.',
      'Read-only: allow edits in your Codex permission settings before retrying.',
      'No input: capture a note or snapshot before submitting.',
      'Unsupported media: this host could not interpret the recording. Send a snapshot or text note.'
    ].entries()){
      const failureNote=new FormData();
      failureNote.append('raw_transcript','Failure display check '+index);
      failureNote.append('normalized','Failure display check '+index);
      const captured=await fetch(base+'/api/session/feedback/note',{method:'POST',headers:{'X-Knit-Token':token,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:failureNote});
      assert.equal(captured.status,200);
      await post('/api/session/approve',{summary:'Failure display check'});
      const failure=await post('/api/desktop/submit',{binding_id:binding,request_id:'failure-display-'+index});
      await post('/api/desktop/panel',{action:'prepare',binding_id:binding,delivery_id:failure.delivery_id});
      await post('/api/desktop/panel',{action:'report',binding_id:binding,delivery_id:failure.delivery_id,status:'working'});
      await post('/api/desktop/panel',{action:'report',binding_id:binding,delivery_id:failure.delivery_id,status:'failed',result:message});
      await recorder.evaluate(()=>refreshSession(true));
      await recorder.getByRole('alert').filter({hasText:message}).waitFor({state:'visible'});
      await page.locator('#refresh').click();
      await page.getByRole('alert').filter({hasText:message}).waitFor({state:'visible'});
      await mainUI.reload();
      await mainUI.waitForFunction(message=>document.getElementById('desktopConnections').textContent.includes(message),message);
    }
    await post('/api/desktop/panel',{action:'delivery_settings',binding_id:binding,automatic_delivery:true});
    const note=new FormData();
    note.append('raw_transcript','Second request: verify recovery');
    note.append('normalized','Second request: verify recovery');
    const saved=await fetch(base+'/api/session/feedback/note',{method:'POST',headers:{'X-Knit-Token':token,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:note});
    assert.equal(saved.status,200);
    await post('/api/session/approve',{summary:'Verify recovery'});
    const uncertain=await post('/api/desktop/submit',{binding_id:binding,request_id:'uncertain-host-request'});
    const duplicate=await post('/api/desktop/submit',{binding_id:binding,request_id:'uncertain-host-request'});
    assert.equal(duplicate.delivery_id,uncertain.delivery_id,'duplicate browser submit must reuse the durable delivery');
    await page.evaluate(()=>{window.openai.sendFollowUpMessage=async()=>{throw Error('Host disconnected');};});
    await page.waitForFunction(()=>document.getElementById('deliveries').textContent.includes('needs attention'));
    assert.match(await page.locator('#status').innerText(),/Could not confirm delivery: Host disconnected/);
    assert.equal(await page.getByRole('button',{name:'Send to this conversation',exact:true}).count(),0);
    const blocked=await fetch(base+'/api/desktop/panel',{method:'POST',headers:{'Content-Type':'application/json','X-Knit-Token':token,'X-Knit-Timestamp':String(Date.now()),'X-Knit-Nonce':randomUUID()},body:JSON.stringify({action:'prepare',binding_id:binding,delivery_id:uncertain.delivery_id})});
    assert.equal(blocked.status,409,'uncertain feedback must never be silently resent');
    await page.locator('#disconnect').click();
    await page.locator('#connectForm').waitFor({state:'visible'});
    assert.deepEqual(errors,[]);
    console.log('PASS: scoped extension pairing/submit -> real daemon -> panel -> injected host -> durable completion -> extension/main UI; delayed widget restoration, consecutive deliveries without reconnecting, cached-page polling recovery, host rejection, no replay, disconnect, mobile layout, no bridge.');
  } finally { await browser.close(); }
}
main().catch(error=>{console.error(error);process.exitCode=1;});
