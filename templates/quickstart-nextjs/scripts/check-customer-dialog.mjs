// Offline browser regression for the real React dialog, table and Select.
// Only API/session/i18n boundaries and the decorative close icon are fixtures.
// Uses an isolated Chrome profile, embedded local assets and no running server.
import assert from "node:assert/strict";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { createRequire } from "node:module";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { spawnSync } from "node:child_process";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(join(root, "package.json"));
const ts = require("typescript");
const browser = process.argv[2] || ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/usr/bin/google-chrome", "/usr/bin/chromium"].find(existsSync);
assert.ok(browser && existsSync(browser), "Supply the Chrome/Chromium executable as the first argument.");
const output = process.argv[3] ? resolve(process.argv[3]) : mkdtempSync(join(tmpdir(), "saas-customer-dialog-"));
mkdirSync(output, { recursive: true });
const compile = file => ts.transpileModule(readFileSync(join(root, file), "utf8"), {
  compilerOptions: { jsx: ts.JsxEmit.React, module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true }
}).outputText;
const modules = Object.fromEntries(Object.entries({
  "./operator-user-dialog": "components/operator-user-dialog.tsx",
  "./operator-overview": "components/operator-overview.tsx",
  "./responsive-dialog": "components/responsive-dialog.tsx",
  "./select": "components/select.tsx",
  "./data-table": "components/data-table.tsx",
  "./status-badge": "components/status-badge.tsx",
  "@/lib/customer-records": "lib/customer-records.ts",
  "@/lib/dialog-focus": "lib/dialog-focus.ts",
  "../lib/table-state": "lib/table-state.ts",
  "@/lib/admin-presentation": "lib/admin-presentation.ts",
  "@/lib/format": "lib/format.ts",
  "@/lib/billing-money": "lib/billing-money.ts"
}).map(([name, file]) => [name, compile(file)]));
const umd = name => readFileSync(join(dirname(require.resolve(name)), "umd", `${name}.development.js`), "utf8");
const fontFile = require.resolve("@fontsource-variable/manrope/index.css");
const fonts = readFileSync(fontFile, "utf8").replace(/url\(([^)]+)\)/g, (_, asset) => `url(data:font/woff2;base64,${readFileSync(resolve(dirname(fontFile), asset.replaceAll(/["']/g, ""))).toString("base64")})`);
const css = ["app/globals.css", "components/admin-shell.module.css", "components/responsive-dialog.module.css", "components/status-badge.module.css"]
  .map(file => readFileSync(join(root, file), "utf8").replace(/:global\(([^)]+)\)/g, "$1")).join("\n");

function fixture(locale, mode) {
  const script = String.raw`
const modules = MODULES, cache = {}, checks = [];
const locale = LOCALE, mode = MODE;
const user = {id:'user-fixture',email:'customer@example.test',created_at:'2026-01-02T03:04:05Z',credits:1250,plan:'pro'};
const profile = {user,membership:{plan:'pro',active:true},billing:{plan:'free',active:false},override:null,subscription:null};
const ledger = ['admin_grant','referral_signup','stripe','monthly','campaign','note_export'].map((source,index)=>({
  id:index+1,kind:index===5?'consume':'grant',source,actor_id:index===0?'admin-fixture':'',
  amount:index===5?-5:50,balance_after:1000+index*50,reason:['Support adjustment','Invitation reward','Credit pack purchase','Monthly allowance','Campaign reward','Export usage'][index],
  source_id:'fixture-'+(index+1),created_at:'2026-01-02T03:04:05Z'
}));
const orders = ['unpaid','paid'].map((status,index)=>({id:'stripe:cs_fixture_'+index,provider:'stripe',record_type:'checkout',product_type:'subscription',plan:'pro',billing_interval:'monthly',status,currency:'usd',amount_due:1900,amount:status==='paid'?1900:null,created_at:'2026-01-02T03:04:05Z'}));
const api = {getOperatorAccess:async()=>profile,updateOperatorMembership:async()=>({}),grantCredits:async()=>({}),deductOperatorCredits:async()=>({}),
  listCreditTransactions:async()=>({items:ledger,total:ledger.length,page:1,limit:20}),listOperator:async()=>({items:orders,total:orders.length,page:1,limit:20})};
const kit = {newIntentKey:()=>crypto.randomUUID(),ConsoleError:()=>null,
  useConsoleAction:()=>{const[message,setMessage]=React.useState('');const[busy,setBusy]=React.useState(false);return{message,setMessage,busy,error:null,run:async(fn,done)=>{setBusy(true);try{done(await fn())}finally{setBusy(false)}}}},
  useConsoleResource:(token,loader)=>{const[data,setData]=React.useState(null);const[loading,setLoading]=React.useState(true);const refresh=React.useCallback(async()=>{setData(await loader(token));setLoading(false)},[token,loader]);React.useEffect(()=>{void refresh()},[refresh]);return{data,loading,error:null,refresh}}};
function load(name){
  if(name==='react')return React;
  if(name==='@/lib/i18n')return{useI18n:()=>({locale,t:value=>value[locale]})};
  if(name==='@/lib/operations-api')return api;
  if(name==='./console-kit')return kit;
  if(name==='./pending-label')return{PendingLabel:props=>React.createElement('span',null,props.children)};
  if(name==='@/components/interface-icon')return{InterfaceIcon:()=>React.createElement('svg',{viewBox:'0 0 24 24','aria-hidden':true},React.createElement('path',{d:'m6 6 12 12M6 18 18 6',stroke:'currentColor',fill:'none'}))};
  if(name.endsWith('.css'))return{__esModule:true,default:new Proxy({},{get:(_,key)=>String(key)})};
  if(!modules[name])throw Error('Unexpected dependency '+name);
  if(cache[name])return cache[name];const module={exports:{}};new Function('require','module','exports',modules[name])(load,module,module.exports);cache[name]=module.exports;return module.exports;
}
const content = mode==='filters'
  ? React.createElement(load('./operator-overview').OperatorOverview,{data:null,filters:{days:30,timezone:'Asia/Shanghai',mode:'live'},onFilters:()=>{},loading:false})
  : React.createElement(load('./operator-user-dialog').OperatorUserDialog,{user,token:'fixture',tab:'overview',onClose:()=>{},onSaved:()=>{}});
ReactDOM.createRoot(document.getElementById('root')).render(React.createElement('div',{className:'shell'},React.createElement('div',{className:'content',style:mode==='filters'?{padding:'24px'}:undefined},content)));
const settle=()=>new Promise(resolve=>setTimeout(resolve,40));
function check(ok,message){if(!ok)throw Error(message);checks.push(message)}
function click(en,zh){const text=locale==='zh'?zh:en;const button=[...document.querySelectorAll('button')].find(element=>element.textContent.trim()===text);if(!button)throw Error('Missing button '+text);button.click()}
async function run(){await document.fonts.ready;await settle();
  if(mode!=='filters'){
    check(document.querySelectorAll('[role=tab]').length===3,'Exactly three tabs');
    check(document.querySelector('.customerFacts').textContent.includes('2026'),'Registration date visible');
    for(let pass=0;pass<3;pass++){
      click('Credit changes','积分变更记录');await settle();check(!document.querySelector('form')&&document.querySelectorAll('tbody tr').length===6,'Unified read-only ledger, no retained editor');
      click('Payment records','支付记录');await settle();check(!document.querySelector('form')&&document.querySelectorAll('tbody tr').length===2,'Paid and unpaid records, no editor');
      click('Overview','用户概况');await settle();check(!document.querySelector('form'),'Overview has no duplicate form');
    }
    click('Adjust credits','调整积分');await settle();check(document.querySelectorAll('#user-credit-operation').length===1,'One credit editor');
    click('Cancel','取消');await settle();check(!document.querySelector('form'),'Cancel closes editor');
    if(mode==='editor')click('Adjust credits','调整积分');
    if(mode==='credits')click('Credit changes','积分变更记录');
    if(mode==='payments')click('Payment records','支付记录');
    await settle();
    const ids=[...document.querySelectorAll('[id]')].map(element=>element.id);check(ids.length===new Set(ids).size,'Unique DOM IDs');
    const dialog=document.querySelector('dialog');check(dialog.open&&dialog.scrollWidth<=dialog.clientWidth+1,'Dialog opens without horizontal overflow');
  }
  for(const select of document.querySelectorAll('select')){const style=getComputedStyle(select);check(style.appearance==='none'&&parseFloat(style.paddingRight)>=32&&select.getBoundingClientRect().height>=40,'Shared native select height/padding/appearance');}
  check(document.documentElement.scrollWidth<=innerWidth+1,'No page overflow');
  return{ok:true,checks:checks.length,viewport:innerWidth,mode,locale};
}
run().then(result=>parent.postMessage({type:'customer-result',result},'*'),error=>parent.postMessage({type:'customer-result',result:{ok:false,error:error.message,checks}},'*'));
`.replace("MODULES", JSON.stringify(modules)).replace("LOCALE", JSON.stringify(locale)).replace("MODE", JSON.stringify(mode));
  return `<!doctype html><html lang="${locale}"><head><style>${fonts}\n${css}</style></head><body><div id="root"></div><script>${umd("react")}</script><script>${umd("react-dom")}</script><script>${script.replaceAll("</script", "<\\/script")}</script></body></html>`;
}

for (const locale of ["en", "zh"]) {
  for (const width of [1440, 390]) {
    for (const mode of ["overview", "editor", "payments", "credits", "filters"]) {
      const name = `${locale}-${mode}-${width}`;
      const profile = mkdtempSync(join(tmpdir(), "saas-dialog-browser-"));
      try {
        // A real narrow iframe avoids Chrome's native 500px window minimum.
        const source = fixture(locale, mode).replaceAll("&", "&amp;").replaceAll('"', "&quot;");
        const file = join(output, `${name}.html`);
        writeFileSync(file, `<!doctype html><html><head><style>body{margin:0;overflow:hidden}iframe{display:block;border:0}</style><script>addEventListener('message',event=>{if(event.data?.type!=='customer-result')return;const pre=document.createElement('pre');pre.id='result';pre.hidden=true;pre.textContent=JSON.stringify(event.data.result);document.body.append(pre)})</script></head><body><iframe title="Customer fixture" width="${width}" height="1000" srcdoc="${source}"></iframe></body></html>`);
        const result = spawnSync(browser, ["--headless", "--no-first-run", "--no-default-browser-check", "--disable-background-networking", `--user-data-dir=${profile}`, `--window-size=${width},1000`, "--virtual-time-budget=4000", "--dump-dom", `--screenshot=${join(output, `${name}.png`)}`, pathToFileURL(file).href], { encoding: "utf8", timeout: 30000, maxBuffer: 30 * 1024 * 1024 });
        assert.equal(result.status, 0, result.error?.message || result.stderr.slice(-500));
        const raw = result.stdout.match(/<pre id="result"[^>]*>([^<]+)<\/pre>/)?.[1];
        assert.ok(raw, `${name}: browser returned results`);
        const report = JSON.parse(raw);
        assert.ok(report.ok, `${name}: ${report.error}`);
        assert.equal(report.viewport, width, `${name}: real CSS viewport`);
        console.log(`${name}: ${report.checks} checks passed`);
      } finally {
        rmSync(profile, { recursive: true, force: true });
      }
    }
  }
}
console.log(`Customer dialog browser checks passed: 20 fixtures. Screenshots: ${output}`);
