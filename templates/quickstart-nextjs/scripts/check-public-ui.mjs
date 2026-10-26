// Real React/CSS, offline content, fake session/API boundaries and a fresh
// Chrome profile. DevTools pipe stays local; no user browser or service is used.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(join(root, "package.json"));
const ts = require("typescript");
const browser = process.argv[2] || ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/usr/bin/google-chrome", "/usr/bin/chromium"].find(existsSync);
assert.ok(browser && existsSync(browser), "Supply Chrome/Chromium as the first argument.");
const output = process.argv[3] ? resolve(process.argv[3]) : mkdtempSync(join(tmpdir(), "saas-public-ui-"));
mkdirSync(output, { recursive: true });
const compile = file => ts.transpileModule(readFileSync(join(root, file), "utf8"), {
  compilerOptions: { jsx: ts.JsxEmit.React, module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true }
}).outputText;
const files = ["components/language-picker.tsx", "components/account-menu.tsx", "components/sign-out-button.tsx", "components/interface-icon.tsx", "components/localized-link.tsx", "components/public-navigation.tsx", "components/published-content.tsx", "components/site-footer.tsx", "lib/guide-index.ts", "lib/disclosure-focus.ts", "lib/locale.ts", "content/navigation.ts", "content/docs.ts", "content/articles.ts"];
files.push("components/sign-in-dialog.tsx", "components/sign-in-panel.tsx", "components/responsive-dialog.tsx", "components/ui.tsx", "components/resource-feedback.tsx", "lib/dialog-focus.ts", "lib/request-state.ts", "lib/resource-data.ts", "lib/request-generation.ts", "lib/auth-methods.ts", "lib/sign-in-methods-cache.ts");
files.push("components/brand-mark.tsx", "lib/site-branding.ts", "components/content/contact.tsx", "components/site-shell.tsx", "components/breadcrumbs.tsx");
files.push("components/content-discovery.tsx", "lib/content-discovery.ts", "lib/markdown.ts", "lib/workspace.ts", "components/navigation-index.tsx");
const modules = Object.fromEntries(files.map(file => [file.replace(/\.tsx?$/, ""), compile(file)]));
const prefix = file => `m_${file.split("/").at(-1).replace(".module.css", "").replaceAll("-", "_")}_`;
const stylesheets = ["components/account-menu.module.css", "components/published-content.module.css", "components/content/marketing.module.css", "components/site-footer.module.css"];
stylesheets.push("components/sign-in-dialog.module.css", "components/responsive-dialog.module.css");
stylesheets.push("components/content-discovery.module.css", "components/brand-mark.module.css", "components/content/content.module.css");
// Apply module namespacing, leaving :global selectors intact.
const css = readFileSync(join(root, "app/globals.css"), "utf8") + stylesheets.map(file => {
  const globals = [];
  return readFileSync(join(root, file), "utf8")
    .replace(/:global\(([^)]+)\)/g, (_, value) => { globals.push(value); return `__GLOBAL_${globals.length - 1}__`; })
    .replace(/\.([a-zA-Z_][\w-]*)/g, (_, name) => `.${prefix(file)}${name}`)
    .replace(/__GLOBAL_(\d+)__/g, (_, index) => globals[Number(index)]);
}).join("\n");
const umd = name => readFileSync(join(dirname(require.resolve(name)), "umd", `${name}.development.js`), "utf8");
const fontFile = require.resolve("@fontsource-variable/manrope/index.css");
const fonts = readFileSync(fontFile, "utf8").replace(/url\(([^)]+)\)/g, (_, asset) => `url(data:font/woff2;base64,${readFileSync(resolve(dirname(fontFile), asset.replaceAll(/["']/g, ""))).toString("base64")})`);
function page(locale, signIn = false, support = false) {
  const js = String.raw`
const modules=MODULES,cache={},locale=LOCALE,authFixture=AUTH_FIXTURE,supportFixture=SUPPORT_FIXTURE;
const fixture=window.fixture={session:{token:'fixture-token',expires_at:'2099-01-01T00:00:00Z',user:{id:'fixture',username:'Demo Customer',email:'customer@example.test',role:'user'}},calls:0,redirects:[],mode:'pending'};
fixture.capabilityRequests=0;fixture.oauthRequests=0;fixture.now=100;fixture.advance=ms=>fixture.now+=ms;
if(authFixture)Date.now=()=>fixture.now;
const originalSession={...fixture.session};
const i18n={useI18n:()=>({locale,t:copy=>copy[locale]})};
const icon=React.createElement;
function resolveModule(name,parent){if(name.startsWith('@/'))return name.slice(2);if(name.startsWith('.'))return new URL(name,'https://fixture/'+parent).pathname.slice(1);return name}
function load(name,parent=''){
  const key=resolveModule(name,parent);
  if(key==='react')return React;
  if(key==='next/link')return{__esModule:true,default:({href,onClick,onNavigate,...props})=>icon('a',{...props,href,onClick:event=>{event.preventDefault();onClick?.(event);onNavigate?.()}})};
  if(key==='next/navigation')return{usePathname:()=>supportFixture?'/contact':'/resources',useRouter:()=>({replace:path=>fixture.redirects.push(path)})};
  if(key==='lib/i18n')return i18n;
  // The guide fixture saves menus as an administrator could; the support fixture keeps the defaults.
  if(key==='components/site-presentation')return{useSitePresentation:()=>supportFixture?null:{navigation:{version:1,header:[{section:'features',label:{en:'',zh:''},menu:'auto',items:[]},{section:'use-cases',label:{en:'',zh:''},menu:'auto',items:[]}],footer:[{title:{en:'Product',zh:'产品'},links:['features','use-cases','compare','examples','pricing']},{title:{en:'About & support',zh:'关于与支持'},links:['about','contact']}]}}};
  if(supportFixture && key==='components/account-menu')return{AccountMenu:()=>null};
  if(supportFixture && key==='components/locale-switch')return{LocaleSwitch:()=>null};
  if(supportFixture && key==='components/customer-shell')return{CustomerPage:()=>null};
  if(supportFixture && key==='lib/seo')return{publicPaths:['/contact']};
  if(supportFixture && key==='lib/site-settings')return{publicSiteSettingsFallback:{support_email:'',support_url:''},SITE_SETTINGS_EVENT:'fixture-settings-change',getPublicSiteSettings:async()=>({support_email:'support@example.test',support_url:'https://support.example.test'})};
  if(authFixture && key==='lib/api')return{ApiError:class extends Error{},getCapabilities:()=>{fixture.capabilityRequests++;return new Promise((resolve,reject)=>{fixture.resolveMethods=auth=>resolve({auth,billing:{enabled:true}});fixture.rejectMethods=()=>reject(new TypeError('fixture offline'))})}};
  if(authFixture && key==='lib/auth')return{readSession:()=>null,readReferralCode:()=>'',REFERRAL_EVENT:'fixture-referral'};
  if(authFixture && key==='lib/env')return{appEnv:{authEmailConfigured:false,authEmailEnabled:true,authOAuthProvidersConfigured:false,authOAuthProviders:[]}};
  if(authFixture && key==='lib/oauth-flow')return{resolveSignInDestination:()=>'/account',getOAuthRedirectURL:async()=>{fixture.oauthRequests++;throw new TypeError('fixture prevents external navigation')}};
  if(key==='lib/env')return{appEnv:{demoMode:true,appName:'SaaS Starter',appUrl:'https://example.test'}};
  if(key==='lib/auth')return{readSession:()=>fixture.session};
  if(key==='lib/api')return{userLabel:user=>user.username||user.email,logout:token=>{fixture.calls++;return new Promise((resolve,reject)=>{fixture.resolve=()=>{if(fixture.session?.token===token){fixture.session=null;dispatchEvent(new Event('fixture-session'))}resolve({ok:true})};fixture.reject=()=>reject(new Error('private raw failure'))})}};
  if(key==='lib/account-actions')return{accountLogoutFailure:()=>locale==='zh'?'退出失败，请检查网络后重试。':'Sign-out failed. Check your connection and retry.'};
  if(!authFixture && key==='components/sign-in-dialog')return{SignInDialog:()=>null};
  if(key==='components/markdown-content')return{MarkdownContent:()=>null};
  if(key==='components/article-language-context')return{ArticleLanguagesProvider:props=>props.children};
  if(key==='components/structured-data')return{StructuredData:()=>null};
  if(key==='lib/structured-data')return{articleStructuredData:()=>({}),breadcrumbStructuredData:()=>({})};
  if(key==='content/site')return{siteConfig:{footer:{description:{en:'A connected starting point for your product.',zh:'从清楚的起点，构建你的产品。'}},brand:{subtitle:{en:'Next.js + Go',zh:'Next.js + Go'}}},getHomeContent:()=>({})};
  if(!supportFixture && key==='components/site-shell')return{SiteShell:Frame};
  if(key.endsWith('.css')){const p='m_'+key.split('/').at(-1).replace('.module.css','').replaceAll('-','_')+'_';return{__esModule:true,default:new Proxy({},{get:(_,name)=>p+String(name)})}}
  if(cache[key])return cache[key];if(!modules[key])throw Error('Unexpected dependency '+key);
  const m={exports:{}};new Function('require','module','exports',modules[key])(name=>load(name,key),m,m.exports);cache[key]=m.exports;return m.exports;
}
const navigationFixture=[{path:'/features/authentication',kind:'features',title:'Email and OAuth login',position:1},{path:'/features/credits',kind:'features',title:'Credit ledger',position:2},{path:'/use-cases/subscription-saas',kind:'use-cases',title:'Subscription SaaS',position:1},{path:'/compare/starter-vs-from-scratch',kind:'compare',title:'Starter or scratch',position:1}];
function Frame(props){const[session,setSession]=React.useState(fixture.session);React.useEffect(()=>{const sync=()=>setSession(fixture.session);addEventListener('fixture-session',sync);return()=>removeEventListener('fixture-session',sync)},[]);
  return icon('div',{className:'app-shell compact-shell'},icon('header',{className:'topbar'},icon('div',{className:'topbar-inner'},icon('a',{className:'brand-lockup',href:'#'},'SaaS Starter'),icon(load('components/navigation-index').NavigationIndexProvider,{value:{[locale]:{entries:navigationFixture,directories:['/features','/use-cases','/compare','/docs','/blog','/resources','/about','/updates']}}},icon(load('components/public-navigation').PublicNavigation,{pathname:'/resources'})),icon('div',{className:'topbar-tools'},icon(load('components/language-picker').LanguagePicker,{locale,label:'Language',onSelect:()=>{}}),icon(load('components/account-menu').AccountMenu,{session})))),icon('main',{id:'main-content',className:'page-shell'},icon('section',{className:'hero-grid'},icon('div',{className:'hero-main-card'},icon('span',{className:'eyebrow'},props.eyebrow),icon('h1',null,props.title),icon('p',null,props.description))),props.children),icon(load('components/site-footer').SiteFooter,{name:'SaaS Starter'}));
}
const docs=load('content/docs').documentation.map(row=>({id:'docs:'+row.id,path:'/docs/'+row.id,kind:'docs',translation:{title:row.title[locale],summary:row.summary[locale]}}));
const articles=load('content/articles').articles.map(row=>({id:'blog:'+row.slug,path:'/blog/'+row.slug,kind:'blog',translation:{title:row.title[locale],summary:row.summary[locale]}}));
const entries=[...docs,...articles,{id:'feature-fixture',kind:'features',path:'/features/billing',translation:{title:'Feature page fixture',summary:'Not a guide'}}];
function SignInFixture(){const[open,setOpen]=React.useState(false);return icon(React.Fragment,null,icon('button',{id:'open-sign-in',className:'button primary',onClick:()=>setOpen(true)},'Open sign-in'),icon(load('components/sign-in-dialog').SignInDialog,{open,onClose:()=>setOpen(false)}))}
ReactDOM.createRoot(document.getElementById('root')).render(supportFixture?icon(load('components/content/contact').ContactContent):authFixture?icon(React.StrictMode,null,icon(SignInFixture)):icon(load('components/published-content').PublishedContentIndex,{kind:'resources',entries}));
fixture.open=()=>document.getElementById('open-sign-in').click();
fixture.close=()=>document.querySelector('dialog[open] button[aria-label]').click();
fixture.watchLoading=()=>{fixture.loadingSeen=false;new MutationObserver(()=>{if(document.querySelector('dialog[open] .sign-in-stack')?.textContent.match(/Loading available|正在加载可用/))fixture.loadingSeen=true}).observe(document.getElementById('root'),{subtree:true,childList:true,characterData:true})};
fixture.reset=()=>{fixture.session={...originalSession};dispatchEvent(new Event('fixture-session'))};
fixture.click=text=>{const element=[...document.querySelectorAll('button')].find(node=>node.textContent.trim()===text);if(!element)throw Error('Missing button '+text);element.click()};
fixture.search=value=>{const input=document.getElementById('published-search');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(input,value);input.dispatchEvent(new Event('input',{bubbles:true}))};
`.replace("LOCALE", () => JSON.stringify(locale)).replace("AUTH_FIXTURE", () => JSON.stringify(signIn)).replace("SUPPORT_FIXTURE", () => JSON.stringify(support)).replace("MODULES", () => JSON.stringify(modules));
  return `<!doctype html><html lang="${locale}"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>${fonts}\n${css}</style></head><body><div id="root"></div><script>${umd("react")}</script><script>${umd("react-dom")}</script><script>${js.replaceAll("</script", "<\\/script")}</script></body></html>`;
}

const profile = mkdtempSync(join(tmpdir(), "saas-public-browser-"));
const chrome = spawn(browser, ["--headless", "--no-first-run", "--no-default-browser-check", "--disable-background-networking", "--remote-debugging-pipe", `--user-data-dir=${profile}`], { stdio: ["ignore", "ignore", "ignore", "pipe", "pipe"] });
let serial=0, buffer="";
const pending=new Map();
chrome.stdio[4].on("data", chunk=>{buffer+=chunk.toString();let end;while((end=buffer.indexOf("\0"))>=0){const packet=JSON.parse(buffer.slice(0,end));buffer=buffer.slice(end+1);if(!packet.id)continue;const task=pending.get(packet.id);if(!task)continue;pending.delete(packet.id);clearTimeout(task.timer);if(packet.error)task.reject(new Error(JSON.stringify(packet.error)));else task.resolve(packet.result)}});
function send(method,params={},sessionId){return new Promise((resolve,reject)=>{const id=++serial;const timer=setTimeout(()=>{pending.delete(id);reject(new Error('Timed out: '+method))},15000);pending.set(id,{resolve,reject,timer});chrome.stdio[3].write(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})})+'\0')})}
let checks=0;
try {
  const target=await send("Target.createTarget",{url:"about:blank"});
  const {sessionId}=await send("Target.attachToTarget",{targetId:target.targetId,flatten:true});
  const command=(method,params)=>send(method,params,sessionId);
  const evaluate=async expression=>{const value=await command("Runtime.evaluate",{expression,awaitPromise:true,returnByValue:true});assert.ok(!value.exceptionDetails,JSON.stringify(value.exceptionDetails));return value.result?.value};
  const settle=()=>evaluate("new Promise(resolve=>setTimeout(resolve,70))");
  const check=async(expression,message)=>{assert.ok(await evaluate(expression),message);checks++};
  const screenshot=async name=>{await new Promise(resolve=>setTimeout(resolve,220));const {data}=await command("Page.captureScreenshot",{format:"png",captureBeyondViewport:false});writeFileSync(join(output,name+'.png'),Buffer.from(data,'base64'))};
  await command("Page.enable");
  for(const locale of ["en","zh"]){for(const width of [1440,390]){
    const tag=locale+'-'+width;
    await command("Emulation.setDeviceMetricsOverride",{width,height:1000,deviceScaleFactor:1,mobile:false});
    const file=join(output,tag+'.html');writeFileSync(file,page(locale));
    await command("Page.navigate",{url:pathToFileURL(file).href});
    for(let attempt=0;attempt<30;attempt++){if(await evaluate("Boolean(document.querySelector('#published-search'))"))break;await new Promise(resolve=>setTimeout(resolve,40))}
    await check("Boolean(document.querySelector('#published-search'))",tag+': React fixture mounted');
    await check(`JSON.stringify([...document.querySelectorAll('[data-guide-directories] a')].map(a=>a.getAttribute('href')))===JSON.stringify(${JSON.stringify(["docs","blog","updates"].map(path=>(locale==='zh'?'/zh':'')+'/'+path))})`,tag+': guide directories have real localized links');
    await evaluate("document.fonts.ready");await settle();
    await check("document.documentElement.scrollWidth<=innerWidth+1",tag+': no page overflow');
    const expected=width>1000?3:1;
    await check(`new Set([...document.querySelectorAll('main article')].slice(0,3).map(row=>row.getBoundingClientRect().top)).size===${expected===3?1:3}`,tag+': responsive card columns');
    await check("!document.querySelector('main').textContent.includes('Feature page fixture')",tag+': no repeated feature pages');
    await screenshot(tag+'-guides');
    await evaluate("document.querySelector('.language-trigger').click()");await settle();
    await check("JSON.stringify([...document.querySelectorAll('.language-options button')].map(e=>e.lang))===JSON.stringify(['en','zh-CN'])",tag+': English is always listed first');
    const other=locale==='zh'?'en':'zh-CN';
    const point=await evaluate(`(()=>{const box=document.querySelector('.language-options button[lang="${other}"]').getBoundingClientRect();return{x:box.x+box.width/2,y:box.y+box.height/2}})()`);
    await command("Input.dispatchMouseEvent",{type:"mouseMoved",...point});await settle();
    await check(`getComputedStyle(document.querySelector('.language-options [aria-pressed="true"]')).backgroundColor!==getComputedStyle(document.querySelector('.language-options button[lang="${other}"]')).backgroundColor`,tag+': selected and hovered languages differ');
    await check("(()=>{const rows=[...document.querySelectorAll('.language-options button')];return rows[1].getBoundingClientRect().top-rows[0].getBoundingClientRect().bottom>=4})()",tag+': separate language rows');
    await screenshot(tag+'-language');
    await evaluate("document.querySelector('.language-trigger').click();document.querySelector('.account-menu > button').click()");await settle();
    await check("!document.querySelector('.account-menu').textContent.match(/Account information|账户信息/)",tag+': no redundant account information');
    await check(`JSON.stringify([...document.querySelectorAll('.account-menu a[data-account-item]')].map(a=>a.getAttribute('href')))===JSON.stringify(${JSON.stringify(["/account","/billing","/credits","/referrals"])})`,tag+': account menu shares all four customer destinations');
    await screenshot(tag+'-account');
    const signout=locale==='zh'?'退出登录':'Sign out';
    await evaluate(`[...document.querySelectorAll('.account-menu button')].find(button=>button.textContent.trim()==='${signout}').focus()`);await settle();
    await evaluate(`fixture.click('${signout}');fixture.click('${signout}')`);await settle();
    await check("fixture.calls===1 && Boolean(document.querySelector('.account-menu button[disabled]')) && document.querySelector('.account-menu [id]').hidden===false",tag+': logout submits once and keeps pending visible');
    await evaluate("fixture.reject()");await settle();
    await check("Boolean(fixture.session) && Boolean(document.querySelector('[role=alert]')) && !document.querySelector('[role=alert]').textContent.includes('private raw')",tag+': failed logout is retryable and hides raw errors');
    await evaluate(`fixture.click('${signout}');fixture.resolve()`);await settle();
    await check(`!fixture.session && fixture.redirects.at(-1)==='${locale==='zh'?'/zh':'/'}'`,tag+': confirmed logout returns to localized home');
    await evaluate("fixture.reset()");await settle();await evaluate("document.querySelector('.account-menu > button').click()");await settle();
    await evaluate(`fixture.click('${signout}');fixture.session={...fixture.session,token:'newer-login'};dispatchEvent(new Event('fixture-session'));fixture.resolve()`);await settle();
    await check("fixture.session.token==='newer-login' && fixture.redirects.length===1",tag+': old logout preserves newer login');
    await check("document.querySelectorAll('.main-nav-item > .main-nav-link').length===2",tag+': saved header has Features and Use cases');
    await check("!document.querySelector('.main-nav button, .main-nav svg, .main-nav [aria-expanded]')",tag+': menus need no scripted controls');
    const prefix=locale==='zh'?'/zh':'';
    const featureLinks=JSON.stringify(["/features/authentication","/features/credits"].map(path=>prefix+path));
    await check(`JSON.stringify([...document.querySelectorAll('.main-nav-item:first-child .main-nav-menu ul a')].map(a=>a.getAttribute('href')))===${JSON.stringify(featureLinks)}`,tag+': feature menu links published pages in order');
    await check(`JSON.stringify([...document.querySelectorAll('.main-nav-item:nth-child(2) .main-nav-menu ul a')].map(a=>a.getAttribute('href')))===${JSON.stringify(JSON.stringify([prefix+"/use-cases/subscription-saas"]))}`,tag+': use-case menu is one level of use cases');
    await check(`parseFloat(getComputedStyle(document.querySelector('.main-nav')).columnGap)>=8`,tag+': navigation has breathing room');
    await check("(()=>{const links=[...document.querySelectorAll('.main-nav-item > .main-nav-link')];return links.slice(1).every((link,index)=>link.getBoundingClientRect().left-links[index].getBoundingClientRect().right>=8)})()",tag+': full link targets remain separated');
    await check("[...document.querySelectorAll('.main-nav-menu')].every(menu=>getComputedStyle(menu).visibility==='hidden')",tag+': menus start closed');
    const featurePoint=await evaluate("(()=>{const b=document.querySelector('.main-nav-item:first-child > .main-nav-link').getBoundingClientRect();return{x:b.x+b.width/2,y:b.y+b.height/2}})()");
    await command("Input.dispatchMouseEvent",{type:"mouseMoved",...featurePoint});await settle();
    if(width>860){
      await check("getComputedStyle(document.querySelector('.main-nav-item:first-child .main-nav-menu')).visibility==='visible'",tag+': hovering a directory opens its menu');
      await check("(()=>{const box=document.querySelector('.main-nav-item:first-child .main-nav-menu-card').getBoundingClientRect();return box.left>=0&&box.right<=innerWidth})()",tag+': open menu fits the viewport');
    } else await check("getComputedStyle(document.querySelector('.main-nav-menu')).display==='none' && getComputedStyle(document.querySelector('.has-menu > .main-nav-link'),'::after').display==='none'",tag+': small screens use directory pages, without menu arrows');
    await screenshot(tag+'-navigation');
    await command("Input.dispatchMouseEvent",{type:"mouseMoved",x:5,y:600});await settle();
    if(width>860){
      await evaluate("document.querySelector('.main-nav-item:nth-child(2) > .main-nav-link').focus()");await settle();
      await check("getComputedStyle(document.querySelector('.main-nav-item:nth-child(2) .main-nav-menu')).visibility==='visible'",tag+': focusing a directory opens its menu');
      await evaluate("document.querySelector('.main-nav-item:nth-child(2) .main-nav-menu a').focus()");await settle();
      await check("document.activeElement===document.querySelector('.main-nav-item:nth-child(2) .main-nav-menu a') && getComputedStyle(document.querySelector('.main-nav-item:nth-child(2) .main-nav-menu')).visibility==='visible'",tag+': the next focus stop is inside the open menu');
      await screenshot(tag+'-navigation-focus');
      await evaluate("document.activeElement.blur()");await settle();
    }
    await evaluate(`fixture.click('${locale==='zh'?'使用文档':'Documentation'}')`);await settle();
    await check(`document.querySelectorAll('main article').length>0 && [...document.querySelectorAll('main article h2 a')].every(link=>link.getAttribute('href').includes('/docs/'))`,tag+': category filters results');
    await evaluate("fixture.search('not-a-guide-match')");await settle();
    await check("document.querySelectorAll('main article').length===0",tag+': search empty state');
    await evaluate("fixture.search('')");await settle();
    await evaluate(`fixture.click('${locale==='zh'?'全部指南':'All guides'}')`);await settle();
    await evaluate("window.scrollTo({top:document.body.scrollHeight,behavior:'instant'})");await settle();
    await check("(()=>{const links=[...document.querySelectorAll('footer a')].map(link=>link.getAttribute('href'));return links.length===new Set(links).size})()",tag+': footer links do not repeat');
    await check("document.documentElement.scrollWidth<=innerWidth+1",tag+': footer fits viewport');
    await check("(()=>{const bottom=document.querySelector('[data-site-footer] > div > div:last-child');const [copyright,legal,top]=[...bottom.children].map(e=>e.getBoundingClientRect());return innerWidth>800?Math.abs(copyright.y+copyright.height/2-legal.y-legal.height/2)<1&&Math.abs(top.y+top.height/2-legal.y-legal.height/2)<1:copyright.bottom<=legal.top&&legal.bottom<=top.top})()",tag+': footer legal row aligns on desktop and stacks clearly on mobile');
    if(width>800)await check("(()=>{const bottom=document.querySelector('[data-site-footer] > div > div:last-child');const elements=[bottom.firstElementChild,...bottom.querySelectorAll('a')];const boxes=elements.map(element=>{const range=document.createRange();range.selectNodeContents(element);return range.getBoundingClientRect()});return Math.max(...boxes.map(b=>b.bottom))-Math.min(...boxes.map(b=>b.bottom))<1})()",tag+': footer text baselines align within one pixel');
    if(width>800)await check("(()=>{const bottom=document.querySelector('[data-site-footer] > div > div:last-child');const elements=[bottom.firstElementChild,...bottom.querySelectorAll('a')];const heights=elements.map(e=>e.getBoundingClientRect().height);const fonts=elements.map(e=>{const c=getComputedStyle(e);return[c.fontSize,c.fontWeight,c.lineHeight].join('/')});return Math.max(...heights)-Math.min(...heights)<1&&new Set(fonts).size===1})()",tag+': footer copyright and links share text metrics and row height');
    await check("document.querySelector('footer').getBoundingClientRect().bottom<=innerHeight+1",tag+': footer bottom is visible');
    await check("!document.querySelector('footer').textContent.includes('Next.js')",tag+': footer omits legacy architecture tagline');
    await screenshot(tag+'-footer');
    console.log(tag+': interaction and layout checks passed');
  }}
  for(const locale of ["en","zh"]){for(const width of [1440,768,390]){
    const tag=locale+'-'+width+'-support';
    await command('Emulation.setDeviceMetricsOverride',{width,height:1000,deviceScaleFactor:1,mobile:false});
    const file=join(output,tag+'.html');writeFileSync(file,page(locale,false,true));await command('Page.navigate',{url:pathToFileURL(file).href});
    for(let attempt=0;attempt<30;attempt++){if(await evaluate("!!document.querySelector('#account-help')"))break;await new Promise(resolve=>setTimeout(resolve,40))}
    await evaluate('document.fonts.ready');await settle();
    await evaluate("document.querySelectorAll('.hero-main-card').forEach(e=>e.getAnimations().forEach(a=>a.finish()))");
    await check("!!document.querySelector('#account-help')",tag+': support page mounted');
    await check("parseFloat(getComputedStyle(document.querySelector('.hero-main-card h1')).fontSize)<=48",tag+': utility page title uses compact sizing');
    await check("(()=>{const hero=document.querySelector('.hero-grid'),crumb=document.querySelector('.content-breadcrumbs'),title=document.querySelector('.hero-main-card');return title.getBoundingClientRect().top-crumb.getBoundingClientRect().bottom<=36&&hero.getBoundingClientRect().height<300})()",tag+': introduction stays close to breadcrumbs without a tall empty hero');
    await check("!document.querySelector('.hero-side-stack, .toc-card')",tag+': no sidebar stretching the title area');
    await check("(()=>{const hero=document.querySelector('.hero-grid').getBoundingClientRect(),body=document.querySelector('#account-help').getBoundingClientRect();return body.top-hero.bottom>=24&&body.top-hero.bottom<=40})()",tag+': help content follows the introduction');
    await check("document.querySelector('#account-help').querySelectorAll('a').length===4",tag+': all four account help destinations remain available');
    await check("!!document.querySelector('#support a[href=\"mailto:support@example.test\"]')",tag+': configured support email remains available');
    await check(`document.querySelector('#support').textContent.includes('${locale==='zh'?'登录验证码':'login codes'}')`,tag+': reporting guidance stays beside support channels');
    await check("document.documentElement.scrollWidth<=innerWidth+1",tag+': support page fits viewport');
    await check("(()=>{const bar=document.querySelector('.topbar-inner');return bar.scrollWidth<=bar.clientWidth+1})()",tag+': header tools fit the bar');
    await check("!document.querySelector('.topbar-tools a')",tag+': header tools hold only language and account');
    await check(`JSON.stringify([...document.querySelectorAll('.main-nav-item > .main-nav-link')].map(a=>a.getAttribute('href')))===${JSON.stringify(JSON.stringify(["", "/features", "/use-cases", "/compare", "/pricing"].map(path=>(locale==='zh'?'/zh':'')+path).map(path=>path||'/')))} && !document.querySelector('.main-nav-menu')`,tag+': default header is one level: home, features, use cases, comparisons, pricing');
    await screenshot(tag);
  }}
  for(const locale of ["en","zh"]){for(const width of [1440,390]){
    const tag=locale+'-'+width+'-sign-in';
    await command("Emulation.setDeviceMetricsOverride",{width,height:1000,deviceScaleFactor:1,mobile:false});
    const file=join(output,tag+'.html');writeFileSync(file,page(locale,true));
    await command("Page.navigate",{url:pathToFileURL(file).href});
    for(let attempt=0;attempt<30;attempt++){if(await evaluate("Boolean(document.querySelector('#open-sign-in'))"))break;await new Promise(resolve=>setTimeout(resolve,40))}
    await settle();
    await check("fixture.capabilityRequests===0 && !document.querySelector('.sign-in-stack')",tag+': closed dialog does not discover methods');
    await evaluate("fixture.open()");await settle();
    await check(`document.querySelector('dialog[open] h2').textContent==='${locale==='zh'?'登录':'Sign in'}'`,tag+': neutral sign-in heading');
    await check("fixture.capabilityRequests===1 && fixture.oauthRequests===0",tag+': Strict Mode mount starts only one discovery and no OAuth request');
    await evaluate("fixture.close()");await settle();await evaluate("fixture.open()");await settle();
    await check("fixture.capabilityRequests===1",tag+': pending discovery survives close/reopen without another request');
    await evaluate("fixture.resolveMethods({email_enabled:true,oauth_providers:['google']})");await settle();
    await check("Boolean(document.querySelector('.sign-in-google-button')) && Boolean(document.querySelector('#sign-in-email'))",tag+': API enables Google and email');
    await evaluate("document.querySelector('dialog[open]').getAnimations().forEach(animation=>animation.finish())");
    await screenshot(tag);
    await check("document.documentElement.scrollWidth<=innerWidth+1 && document.querySelector('dialog[open]').getBoundingClientRect().bottom<=innerHeight+1",tag+': responsive dialog fits viewport');
    await evaluate("fixture.close();fixture.watchLoading()");await settle();await evaluate("fixture.open()");await settle();
    await check("fixture.capabilityRequests===1 && !fixture.loadingSeen && Boolean(document.querySelector('.sign-in-google-button'))",tag+': warm reopen has no request or loading flash');
    await evaluate("fixture.close();fixture.advance(60001)");await settle();await evaluate("fixture.open()");await settle();
    await check("fixture.capabilityRequests===2 && !document.querySelector('.sign-in-google-button')",tag+': expired methods are checked again before display');
    await evaluate("fixture.rejectMethods()");await settle();
    await check("Boolean(document.querySelector('dialog[open] [role=alert] button'))",tag+': failed discovery can be retried');
    await evaluate("document.querySelector('dialog[open] [role=alert] button').click()");await settle();
    await check("fixture.capabilityRequests===3",tag+': retry makes a fresh request');
    await evaluate("fixture.resolveMethods({email_enabled:true,oauth_providers:[]})");await settle();
    await check("!document.querySelector('.sign-in-google-button') && Boolean(document.querySelector('#sign-in-email'))",tag+': disabled provider is removed');
    await evaluate("fixture.close();fixture.advance(60001)");await settle();await evaluate("fixture.open()");await settle();
    await evaluate("fixture.resolveMethods({email_enabled:true,oauth_providers:['google']})");await settle();
    await check("fixture.oauthRequests===0",tag+': opening and retrying never starts OAuth');
    await evaluate("document.querySelector('.sign-in-google-button').click()");await settle();
    await evaluate("document.querySelector('.sign-in-google-button').click()");await settle();
    await check("fixture.oauthRequests===2",tag+': each explicit OAuth attempt obtains a new authorization URL');
    console.log(tag+': sign-in lifecycle checks passed');
  }}
  console.log(`Public UI checks passed: ${checks} assertions. Screenshots: ${output}`);
} finally {
  for(const task of pending.values()){clearTimeout(task.timer);task.reject(new Error('Browser closed'))}pending.clear();
  chrome.kill();
  await new Promise(resolve=>chrome.once('exit',resolve));
  rmSync(profile,{recursive:true,force:true,maxRetries:10,retryDelay:100});
}
