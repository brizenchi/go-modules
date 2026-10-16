const fs=require('fs');const vm=require('vm');const path=require('path');const ts=require(process.cwd()+'/templates/quickstart-nextjs/node_modules/typescript');
const frontendRoot=path.resolve('templates/quickstart-nextjs');const loaded=new Map();
function readFile(filename){
  if(!filename.startsWith(frontendRoot+path.sep))throw new Error('Seed imports must stay inside the frontend');
  if(loaded.has(filename))return loaded.get(filename);
  const code=ts.transpileModule(fs.readFileSync(filename,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
  const sandbox={exports:{},require:specifier=>{
    if(!specifier.startsWith('.'))throw new Error('Seed imports must be local');
    return readFile(path.resolve(path.dirname(filename),specifier)+'.ts');
  }};
  loaded.set(filename,sandbox.exports);vm.runInNewContext(code,sandbox,{filename});return sandbox.exports;
}
function read(name){return readFile(path.join(frontendRoot,'content',name+'.ts'));}
const {articles}=read('articles'),{documentation}=read('docs'),{marketingPages}=read('marketing');
const entries=[];
for(const a of articles){const translations={};for(const l of ['en','zh'])translations[l]={title:a.title[l],summary:a.summary[l],body:a.sections.map(s=>'## '+s.title[l]+'\n\n'+s.paragraphs.map(p=>p[l]).join('\n\n')).join('\n\n'),seo_title:a.title[l],seo_description:a.summary[l]};entries.push({path:'/blog/'+a.slug,kind:'blog',translations});}
for(const a of documentation){const translations={};for(const l of ['en','zh'])translations[l]={title:a.title[l],summary:a.summary[l],body:a.paragraphs.map(p=>p[l]).join('\n\n')+(a.items?'\n\n'+a.items.map(i=>'## '+i.title[l]+'\n\n'+i.body[l]).join('\n\n'):'')+(a.code?'\n\n```\n'+a.code+'\n```':'')+(a.links?'\n\n'+a.links.map(i=>'['+i.label[l]+']('+i.href+')').join('\n\n'):''),seo_title:a.title[l],seo_description:a.summary[l]};entries.push({path:'/docs/'+a.id,kind:'docs',translations});}
for(const a of marketingPages){const translations={};for(const l of ['en','zh']){let parts=[a.intro[l],'## '+(l==='zh'?'如何使用':'How it works'),...a.steps.map(x=>'### '+x.title[l]+'\n\n'+x.body[l])];for(const s of a.sections)parts.push('## '+s.title[l]+'\n\n'+s.body[l]+(s.bullets?'\n\n'+s.bullets.map(b=>'- '+b[l]).join('\n'):''));if(a.limitations?.length)parts.push('## '+(l==='zh'?'注意事项':'Limitations')+'\n\n'+a.limitations.map(x=>'- '+x[l]).join('\n'));if(a.faq?.length)parts.push('## '+(l==='zh'?'常见问题':'Questions')+'\n\n'+a.faq.map(x=>'### '+x.question[l]+'\n\n'+x.answer[l]).join('\n\n'));translations[l]={title:a.title[l],summary:a.description[l],body:parts.join('\n\n'),seo_title:a.title[l],seo_description:a.description[l]};}entries.push({path:a.path,kind:a.kind,translations});}
fs.writeFileSync('templates/quickstart/internal/feature/content/seed.json',JSON.stringify(entries,null,2)+'\n');console.log('Seed entries:',entries.length);
