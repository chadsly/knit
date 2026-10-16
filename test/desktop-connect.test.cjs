const {test}=require('node:test');
const assert=require('node:assert/strict');
require('../extension/chromium/desktop-connect.js');
const connector=globalThis.KnitDesktopConnect;
const encode=patch=>'knit-connect:'+encodeURIComponent(JSON.stringify({version:1,daemon_url:'http://127.0.0.1:7778',pairing_code:'ABC123',binding_id:'panel-1',label:'Shop',expires_at:new Date(Date.now()+60000).toISOString(),...patch}));
test('handoff accepts loopback and rejects remote addresses, credentials, redirects and expired codes',()=>{
 assert.equal(connector.parse(encode()).daemon_url,'http://127.0.0.1:7778');
 for(const daemon_url of ['https://example.com','http://127.0.0.1.evil.test','http://user:pass@localhost','http://localhost/path','http://localhost?token=x'])assert.throws(()=>connector.parse(encode({daemon_url})));
 assert.throws(()=>connector.parse(encode({expires_at:'2000-01-01'})),/expired/);
 assert.throws(()=>connector.parse('knit-connect:%broken'),/damaged/);
 assert.throws(()=>connector.parse('old-raw-token'),/connection code/);
});
test('one handoff verifies daemon binding and starts a review only when needed',async()=>{
 const calls=[];
 const fetcher=async(url,options)=>{
  calls.push({url,options});
  const data=url.endsWith('/complete')?{token:'test-scoped-credential'}:url.endsWith('/start')?{id:'review'}:{desktop:{bindings:[{id:'panel-1',label:'Verified shop'}]}};
  return {ok:true,json:async()=>data};
 };
 const connected=await connector.connect(encode(),fetcher);
 assert.equal(connected.label,'Verified shop');
 assert.equal(connected.desktopDestinationID,'panel-1');
 assert.equal(calls.filter(c=>c.url.endsWith('/start')).length,1);
 assert.ok(calls.every(c=>c.options.redirect==='error'));
 assert.equal(calls[0].options.headers.Authorization,undefined);
 assert.equal(calls[1].options.headers.Authorization,'Bearer test-scoped-credential');
});
test('an existing review is not reset and missing bindings never start a review',async()=>{
 for(const active of [true,false]){
  let starts=0;
  const fetcher=async(url)=>{
   if(url.endsWith('/start'))starts++;
   return {ok:true,json:async()=>url.endsWith('/complete')?{token:'test-only'}:{session:{id:'existing',status:'active'},desktop:{bindings:active?[{id:'panel-1',label:'Shop'}]:[]}}};
  };
  if(active)await connector.connect(encode(),fetcher);
  else await assert.rejects(connector.connect(encode(),fetcher),/disconnected/);
  assert.equal(starts,0);
 }
});
test('rejected pairing and network failures propagate without accepting a connection',async()=>{
 await assert.rejects(connector.connect(encode(),async()=>({ok:false,status:401})),/fresh connection code/);
 await assert.rejects(connector.connect(encode(),async()=>{throw Error('offline');}),/offline/);
});
