const { chromium } = require('playwright');
const assert = require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({headless:true,...(process.env.BROWSER_CHROME?{executablePath:process.env.BROWSER_CHROME}:{}),args:['--no-sandbox']});
 const page=await browser.newPage();const errors=[];
 page.on('pageerror',error=>errors.push(error.message));
 await page.addInitScript(token=>sessionStorage.setItem('example-token',token),process.env.EXAMPLE_TOKEN);
 const base=process.env.BROWSER_BASE||'http://127.0.0.1:18082';
 try{
  await page.goto(base+'/admin/example',{waitUntil:'networkidle'});
  await page.getByRole('textbox',{name:'Message',exact:true}).fill('Browser field round trip');
  const create=page.getByRole('button',{name:'Создать форму примера'});
  if(await create.count())await create.click();
  await page.getByRole('textbox',{name:'Text',exact:true}).fill('Browser element round trip');
  await page.getByRole('button',{name:'Сохранить',exact:true}).click();
  await page.getByRole('status').filter({hasText:'Сохранено'}).waitFor();
  await page.reload({waitUntil:'networkidle'});
  await page.getByRole('textbox',{name:'Text',exact:true}).waitFor();
  assert.equal(await page.getByRole('textbox',{name:'Message',exact:true}).inputValue(),'Browser field round trip');
  assert.equal(await page.getByRole('textbox',{name:'Text',exact:true}).inputValue(),'Browser element round trip');
  await page.getByLabel('Сайт',{exact:true}).selectOption({label:'plain.example.test'});
  await page.getByText('Модуль недоступен на выбранном сайте.').waitFor();
  assert.equal(await page.getByRole('textbox',{name:'Message',exact:true}).count(),0);
  await page.getByLabel('Сайт',{exact:true}).selectOption({label:'extended.example.test'});
  await page.getByRole('textbox',{name:'Message',exact:true}).waitFor();
  assert.equal(await page.getByRole('textbox',{name:'Message',exact:true}).inputValue(),'Browser field round trip');
  await page.goto(base+'/admin/example?missing=1',{waitUntil:'networkidle'});
  await page.getByText('Редактор «example.text» недоступен.',{exact:true}).first().waitFor();
  let mutations=0;page.on('request',request=>{if(['POST','PUT','PATCH','DELETE'].includes(request.method()))mutations++});
  await page.getByRole('button',{name:'Сохранить',exact:true}).click();
  await page.getByRole('alert').filter({hasText:'message: Редактор'}).waitFor();
  assert.equal(mutations,0,'missing editor must block all writes');
  assert.deepEqual(errors,[]);
  console.log('PASS: external SDK page, field + Forms element persistence, site switch, missing editor blocks writes; no page errors');
 }finally{await browser.close()}
})().catch(error=>{console.error(error);process.exit(1)});
