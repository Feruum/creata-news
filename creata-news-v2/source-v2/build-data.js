const fs=require('fs'),path=require('path');
global.window={};require('../site_v1/assets/data.js');
const S=JSON.parse(JSON.stringify(window.CN_SEED)); const C=require('./content.js');
S.version=5;
S.verticals.forEach(v=>v.name_en=C.verticalsEn[v.id]);
S.authors.forEach(a=>{const e=C.authorsEn[a.id];a.name_en=e.name;a.role_en=e.role;a.bio_en=e.bio;Object.assign(a,C.authorsExtra[a.id]||{followers:300+a.id.length*41,achievements:["first-issue"]});});
S.opportunities.forEach(o=>{const e=C.opportunitiesEn[o.id];for(const k in e)o[k+'_en']=e[k];o.photo=C.opportunityPhotos[o.id];});
for(const a of S.articles){
  const w=C.world[a.id]; if(w){a.photo=w.photo;a.photos=w.photos;for(const k in w.en)a[k+'_en']=w.en[k];if(w.title)a.title=w.title;Object.assign(a,C.worldRuPatch[a.id]||{});}
  const p=C.ecoPatch[a.id]; if(p)Object.assign(a,p);
  delete a.illo;
}
const eco=C.eco.map(e=>Object.assign({slug:e.id,layer:'eco',label:'Экосистема Creata',status:'published',sources:1,source:{name:'Creata News № 01',author:'Редакция Creata News',date:'2026-10-15',url:'https://creata.news/issues/01'},corrections:[]},e));
const i=S.articles.findIndex(a=>a.id==='e-paper'); S.articles.splice(i,0,...eco);
const more=C.worldMore.map(e=>Object.assign({slug:e.id,layer:'world',label:'Мир',status:'published',sources:1,corrections:[]},e));
S.articles.splice(0,0,...more);
S.articles.forEach(a=>{if(!a.country&&(a.layer==='eco'||/Казахстан/.test(a.place||'')||a.layer==='world'))a.country='KZ';});
const AEN={'Карлыгаш Нуржан':'Karlygash Nurzhan','Шангерей Бөкей':'Shangerei Bokei','Редакция BES.media':'BES.media editors','Айдана Усупова':'Aidana Usupova','Редакция Creata News':'Creata News editors'};
S.articles.forEach(a=>{if(a.source&&AEN[a.source.author])a.source.author_en=AEN[a.source.author];});
S.comments.forEach(c=>c.lang='ru'); S.comments.push(...C.commentsRuExtra,...C.commentsEn);
S.settings.languages={ru:"проверенный",en:"проверенный",es:"скоро",pt:"скоро",ar:"скоро",hi:"скоро",zh:"скоро"};
S.issue={number:1,month:"Ноябрь 2026",month_en:"November 2026",city:"Алматы",city_en:"Almaty",cover:"assets/photos/c01.jpg"};
fs.writeFileSync('../site/assets/data.js','/* Creata News — демо-данные v4 (RU+EN, реальные фото). Генерируется app/build-data.js */\nwindow.CN_SEED = '+JSON.stringify(S,null,1)+';\n');
// check missing photos
const all=JSON.stringify(S).match(/assets\/photos\/[\w-]+\.jpg/g);const miss=[...new Set(all)].filter(f=>!fs.existsSync('../site/'+f));console.log('articles',S.articles.length,'missing',miss);
