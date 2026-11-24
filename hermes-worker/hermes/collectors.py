import asyncio
import os
from pathlib import Path
from urllib.parse import urlsplit
from urllib.robotparser import RobotFileParser
from .net import fetch
from .extract import extract_html, parse_feed, normalize_url
from .ai import NotConfigured

def allowed(url, *, strict_public=False, timeout=20, redirects=5):
    p=urlsplit(url)
    robots=p.scheme+'://'+p.netloc+'/robots.txt'
    try:
        raw,_,_=fetch(robots,max_bytes=500000,strict_public=strict_public,timeout=timeout,redirects=redirects)
    except RuntimeError as err:
        if '404' in str(err):return True
        raise
    parser=RobotFileParser();parser.parse(raw.decode('utf-8',errors='replace').splitlines())
    return parser.can_fetch('CreataHermes',url)

def page(url, *, strict_public=False, timeout=20, redirects=5):
    if not allowed(url,strict_public=strict_public,timeout=timeout,redirects=redirects):raise ValueError('Source disallows crawling in robots.txt')
    raw,kind,final=fetch(url,strict_public=strict_public,timeout=timeout,redirects=redirects)
    if 'html' not in kind and kind and not kind.startswith('text/'):raise ValueError('Source is not an HTML page')
    return extract_html(raw.decode('utf-8',errors='replace'),final)

def collect(source,check=False):
    kind=source['type'];url=source['url'];locale=source.get('locale','ru')
    if kind=='telegram':return asyncio.run(collect_telegram(source,check))
    if kind=='rss':
        if not allowed(url):raise ValueError('Feed disallows crawling in robots.txt')
        raw,_,final=fetch(url)
        items=parse_feed(raw,final,locale)
        if not check:
            for item in items[:20]:
                if len(item.get('text',''))<500:
                    try:
                        detail=page(item['url'])
                        if len(detail['text'])>len(item['text']):item['text']=detail['text']
                    except (ValueError,RuntimeError,OSError):pass
        rejected=[{'url':i['url'],'error':'Missing title or source text'} for i in items if not i.get('title','').strip() or not i.get('text','').strip()]
        return {'items':[i for i in items if i.get('title','').strip() and i.get('text','').strip()],'checked':check,'rejected':rejected}
    if kind!='website':raise ValueError('Unsupported source type')
    root=page(url)
    items=[]
    if root['title'] and len(root['text'])>100:items.append({**{k:root[k] for k in ('url','title','text','publishedAt','author') if k in root},'locale':locale})
    if not check and not source.get('singlePage',False):
        for link in root['links'][:20]:
            if link==normalize_url(url):continue
            try:
                item=page(link)
                if item['title'] and len(item['text'])>200:items.append({**{k:item[k] for k in ('url','title','text','publishedAt','author') if k in item},'locale':locale})
            except (ValueError,RuntimeError,OSError):continue
    return {'items':items,'checked':check}

async def collect_telegram(source,check):
    session=os.getenv('TELEGRAM_COLLECTOR_SESSION','')
    session_file=os.getenv('TELEGRAM_COLLECTOR_SESSION_FILE','')
    if not session and session_file:session=Path(session_file).read_text().strip()
    api_id=os.getenv('TELEGRAM_API_ID','');api_hash=os.getenv('TELEGRAM_API_HASH','')
    if not session or not api_id or not api_hash:raise NotConfigured('Telegram collector session/API credentials are not configured')
    try:
        from telethon import TelegramClient
        from telethon.sessions import StringSession
    except ImportError:raise NotConfigured('Install the hermes-worker Telegram dependencies') from None
    p=urlsplit(source['url']);username=p.path.strip('/').removeprefix('s/').split('/')[0]
    if p.hostname not in ('t.me','telegram.me') or not username or username.startswith('+'):raise ValueError('A public Telegram channel URL is required')
    client=TelegramClient(StringSession(session),int(api_id),api_hash)
    await client.connect()
    try:
        if not await client.is_user_authorized():raise NotConfigured('Telegram collector session requires operator authorization')
        entity=await client.get_entity(username)
        items=[]
        async for msg in client.iter_messages(entity,limit=1 if check else 100):
            if not msg.message:continue
            text=msg.message[:100000]
            items.append({'externalId':str(entity.id)+':'+str(msg.id),'url':f'https://t.me/{username}/{msg.id}','title':text.splitlines()[0][:500],'text':text,'locale':source.get('locale','ru'),'publishedAt':msg.date.isoformat()})
        return {'items':items,'checked':check}
    finally:await client.disconnect()
