import re
from datetime import datetime, timezone
from email.utils import parsedate_to_datetime
from html import unescape
from html.parser import HTMLParser
from urllib.parse import urlsplit, urlunsplit, parse_qsl, urlencode, urljoin
import xml.etree.ElementTree as ET

def normalize_url(url):
    p=urlsplit(url.strip())
    if p.scheme not in ('http','https') or not p.hostname or p.username or p.password: raise ValueError('Invalid source URL')
    host=p.hostname.lower().encode('idna').decode()
    if ':' in host: host='['+host+']'
    if p.port and p.port != (443 if p.scheme=='https' else 80): host+=':'+str(p.port)
    query=sorted((k,v) for k,v in parse_qsl(p.query,keep_blank_values=True) if not k.lower().startswith('utm_') and k.lower() not in ('fbclid','gclid'))
    return urlunsplit((p.scheme.lower(),host,p.path or '/',urlencode(query),''))

class ArticleParser(HTMLParser):
    ignored={'script','style','nav','footer','header','aside','noscript','svg','form','button'}
    breaks={'p','div','section','h1','h2','h3','li','br','blockquote','article','main'}
    void={'br','img','meta','link','hr','input','source','wbr','area','base','embed','param','track','col'}
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.stack=[];self.parts=[];self.article=[];self.title=[];self.heading=[];self.links=[];self.description='';self.metadata={}
    def handle_starttag(self,tag,attrs):
        attrs=dict(attrs)
        if tag=='a' and attrs.get('href'):self.links.append(attrs['href'])
        if tag=='meta' and attrs.get('name')=='description':self.description=attrs.get('content','')
        if tag=='meta':
            key=attrs.get('name') or attrs.get('property') or attrs.get('itemprop','')
            if key:self.metadata[key.lower()]=attrs.get('content','')
        if tag=='time' and attrs.get('datetime') and 'datepublished' not in self.metadata:self.metadata['datepublished']=attrs['datetime']
        if tag not in self.void:self.stack.append(tag)
        if tag in self.breaks:self._append('\n')
    def handle_endtag(self,tag):
        if tag in self.breaks:self._append('\n')
        if tag in self.stack:
            pos=len(self.stack)-1-self.stack[::-1].index(tag)
            del self.stack[pos:]
    def _append(self,text):
        if any(t in self.ignored for t in self.stack):return
        if 'title' in self.stack:self.title.append(text);return
        self.parts.append(text)
        if 'article' in self.stack or 'main' in self.stack:self.article.append(text)
        if 'h1' in self.stack:self.heading.append(text)
    def handle_data(self,data):self._append(data)

def clean_text(value):
    return '\n\n'.join(re.sub(r'\s+',' ',part).strip() for part in re.split(r'\n+',value) if part.strip()).strip()

def extract_html(html,url):
    parser=ArticleParser();parser.feed(html)
    text=clean_text(''.join(parser.article or parser.parts))
    title=clean_text(''.join(parser.heading or parser.title))
    links=[]
    for href in parser.links:
        try:
            target=normalize_url(urljoin(url,href))
            if urlsplit(target).netloc==urlsplit(normalize_url(url)).netloc and target not in links:links.append(target)
        except ValueError:continue
    result={'title':title[:500], 'text':text[:100000], 'links':links[:100], 'url':normalize_url(url), 'description':parser.description}
    date=parse_date(parser.metadata.get('article:published_time') or parser.metadata.get('datepublished') or parser.metadata.get('pubdate'))
    if date:result['publishedAt']=date
    author=parser.metadata.get('author') or parser.metadata.get('article:author')
    if author:result['author']=author[:500]
    return result

def parse_date(value):
    if not value:return None
    try:
        try:d=datetime.fromisoformat(value.replace('Z','+00:00'))
        except ValueError:d=parsedate_to_datetime(value)
        if d.tzinfo is None:d=d.replace(tzinfo=timezone.utc)
        return d.astimezone(timezone.utc).isoformat().replace('+00:00','Z')
    except (ValueError,TypeError,OverflowError):return None

def parse_feed(data,url,locale):
    if len(data)>5_000_000:raise ValueError('Feed too large')
    root=ET.fromstring(data)
    def local(tag):return tag.rsplit('}',1)[-1]
    out=[]
    for entry in root.iter():
        if local(entry.tag) not in ('item','entry'):continue
        fields={};link=''
        for child in entry:
            name=local(child.tag)
            value=''.join(child.itertext()).strip()
            if name=='link' and child.attrib.get('rel','alternate')=='alternate':link=child.attrib.get('href',value)
            fields[name]=value
        link=link or fields.get('link','')
        if not link:continue
        raw=fields.get('encoded') or fields.get('content') or fields.get('description') or fields.get('summary','')
        try:
            extracted=extract_html(raw,urljoin(url,link))
            item={'url':normalize_url(urljoin(url,link)),'title':unescape(fields.get('title',''))[:500],'text':extracted['text'],'locale':locale}
        except ValueError:continue
        date=parse_date(fields.get('pubDate') or fields.get('published') or fields.get('updated'))
        if date:item['publishedAt']=date
        if fields.get('author') or fields.get('creator'):item['author']=fields.get('author') or fields.get('creator')
        out.append(item)
        if len(out)>=100:break
    return out
