import json
import sys
import unittest
import threading
import os
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
from unittest.mock import patch
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from hermes.extract import normalize_url, extract_html, parse_feed
from hermes.net import public_addresses, UnsafeURL
from hermes.ai import validate_draft, NotConfigured, AIProvider
from hermes.pipeline import Pipeline,source_evidence
from hermes.collectors import collect

class ExtractionTests(unittest.TestCase):
    def test_canonical_url_deduplicates_tracking_and_fragment(self):
        self.assertEqual(normalize_url('https://EXAMPLE.com:443/story?utm_source=x&b=2&a=1#hello'), 'https://example.com/story?a=1&b=2')

    def test_html_discards_navigation_scripts_and_keeps_article(self):
        got = extract_html('<html><head><title>Story</title></head><body><nav>Menu</nav><article><h1>Headline</h1><p>Real paragraph.</p><script>attack()</script></article><footer>Footer</footer></body></html>', 'https://example.com/a')
        self.assertEqual(got['title'], 'Headline')
        self.assertIn('Real paragraph.', got['text'])
        self.assertNotIn('Menu', got['text'])
        self.assertNotIn('attack', got['text'])

    def test_rss_and_atom_preserve_dates_and_links(self):
        rss='<rss><channel><item><title>New</title><link>https://example.com/a</link><description>Facts</description><pubDate>Fri, 13 Nov 2026 10:00:00 GMT</pubDate></item></channel></rss>'
        items=parse_feed(rss.encode(),'https://example.com/feed','kk')
        self.assertEqual(items[0]['locale'],'kk')
        self.assertEqual(items[0]['publishedAt'],'2026-11-13T10:00:00Z')
        atom='<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Atom</title><link href="https://example.com/b"/><summary>Actual body</summary></entry></feed>'
        self.assertEqual(parse_feed(atom.encode(),'https://example.com/feed','ru')[0]['url'],'https://example.com/b')

    def test_website_metadata_preserves_publication_and_author(self):
        result=extract_html('<meta property="article:published_time" content="2026-11-13T10:00:00Z"><meta name="author" content="Editor"><article><h1>Story</h1><p>Facts.</p></article>','https://example.com/story')
        self.assertEqual(result['publishedAt'],'2026-11-13T10:00:00Z')
        self.assertEqual(result['author'],'Editor')

    def test_invalid_link_does_not_drop_valid_feed_items(self):
        rss='<rss><channel><item><title>Bad</title><link>javascript:alert(1)</link></item><item><title>Valid</title><link>https://example.com/a</link><description>Actual text</description></item></channel></rss>'
        self.assertEqual(parse_feed(rss.encode(),'https://example.com/feed','en')[0]['title'],'Valid')

    def test_network_rejects_credentials_private_and_mapped_ipv6(self):
        for url in ['file:///etc/passwd','http://u:p@example.com','http://127.0.0.1/a','http://169.254.169.254/latest','http://[::ffff:127.0.0.1]/']:
            with self.subTest(url=url), patch.dict(os.environ,{'CREATA_ALLOW_PRIVATE_SOURCES':'0'}), self.assertRaises(UnsafeURL): public_addresses(url)

class DraftTests(unittest.TestCase):
    def test_missing_translation_cannot_become_article(self):
        with self.assertRaises(ValueError): validate_draft({'title': {'ru':'X'}})

    def test_draft_requires_real_three_locale_text(self):
        value={f:{l:'A' for l in ['ru','kk','en']} for f in ['title','lead','body','summary','actions']}
        self.assertEqual(validate_draft(value)['body']['kk'],'A')

    def test_unconfigured_ai_does_not_produce_fake_success(self):
        class API:
            def get(self,path): return {'endpoint':'','model':'','apiKey':'','dailyBudgetUsd':0}
        with self.assertRaises(NotConfigured): Pipeline(API()).process({'kind':'classify','payload':{'story':{'title':'X'}}})

    def test_second_url_without_retrieved_text_is_not_corroboration(self):
        evidence,unavailable=source_evidence({'url':'https://example.com/a','text':'Actual evidence'}, {},[{'url':'https://example.com/a'},{'url':'http://127.0.0.1/blocked'}])
        self.assertEqual(len(evidence),1)
        self.assertEqual(unavailable,['http://127.0.0.1/blocked'])

    def test_both_token_prices_must_be_configured_before_reserving(self):
        class API:
            def get(self,path):return {'endpoint':'https://example.com/v1','model':'fixture','apiKey':'fixture','dailyBudgetUsd':1,'inputPricePerMillion':0,'outputPricePerMillion':1}
        with self.assertRaises(NotConfigured):AIProvider(API())

class FixtureHandler(BaseHTTPRequestHandler):
    def log_message(self,*args): pass
    def do_GET(self):
        self.send_response(200);self.send_header('Content-Type','text/plain' if self.path=='/robots.txt' else 'application/rss+xml');self.end_headers()
        if self.path=='/robots.txt':self.wfile.write(b'User-agent: *\nDisallow: /private');return
        marker=getattr(self.server,'fixture_marker','')
        content='Creata Lab reported '+('Evidence. '*80)
        if getattr(self.server,'fixture_numeric',False):content+='Revenue in 2024 was 100 million USD. Revenue in 2025 was 120 million USD. '
        self.wfile.write(('<rss><channel><item><title>Actual fixture'+marker+'</title><link>http://127.0.0.1:'+str(self.server.server_port)+'/article</link><description>'+content+marker+'</description></item></channel></rss>').encode())
    def do_POST(self):
        payload=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        instruction=payload['messages'][1]['content']
        phase='research' if instruction.startswith(('Research','Extract')) else 'review' if instruction.startswith('Review') else 'classify' if instruction.startswith('Classify') else 'draft'
        if hasattr(self.server,'fixture_counts'):self.server.fixture_counts[phase]=self.server.fixture_counts.get(phase,0)+1
        if phase=='review' and getattr(self.server,'fixture_fail_review',False):
            self.server.fixture_fail_review=False
            self.send_response(503);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(b'{"error":"Interrupted fixture review"}');return
        if instruction.startswith('Extract'):
            source_data=json.loads(instruction.split('<SOURCE_DATA>\n',1)[1].rsplit('\n</SOURCE_DATA>',1)[0]);source_url=source_data['retrievedEvidence'][0]['url'];loc=lambda text:{lang:text for lang in ('ru','kk','en')}
            value={'graphic':{'type':'bar','unit':'million USD','period':'2024–2025','sourceUrl':source_url,'sourceName':'Fixture report','methodology':loc('Reported annual revenue'),'verified':False,'labels':[loc('2024'),loc('2025')],'series':[{'name':loc('Revenue'),'values':[100,120]}]},'evidence':[{'row':0,'series':0,'sourceUrl':source_url,'quote':'Revenue in 2024 was 100 million USD.','numberText':'100'},{'row':1,'series':0,'sourceUrl':source_url,'quote':'Revenue in 2025 was 120 million USD.','numberText':'120'}]}
        elif instruction.startswith('Research'):
            source_data=json.loads(instruction.split('<SOURCE_DATA>\n',1)[1].rsplit('\n</SOURCE_DATA>',1)[0])
            source_url=source_data['retrievedEvidence'][0]['url']
            value={'facts':[{'claim':'Fixture','sourceUrls':[source_url]}],'conflicts':[],'singleSourceClaims':['Fixture']}
            if getattr(self.server,'fixture_malformed_research',False):
                self.server.fixture_malformed_research=False;value['facts'][0]['sourceUrls']=[{}]
        elif instruction.startswith('Review'):value={'reviewNotes':[]}
        elif instruction.startswith('Classify'):
            source_data=json.loads(instruction.split('<SOURCE_DATA>\n',1)[1].rsplit('\n</SOURCE_DATA>',1)[0])
            source_url=source_data['story']['url']
            value={'categoryId':'ai','region':'KZ','tags':['ai'],'score':85,'reason':'Research','facts':[{'claim':'Evidence.','sourceUrls':[source_url],'uncertainty':'Single source fixture'}],'entities':[{'name':'Creata Lab','type':'organization','sourceUrls':[source_url]}]}
            if getattr(self.server,'fixture_unknown_category',False):
                self.server.fixture_unknown_category=False;value['categoryId']='invented-category'
        else:
            titles = {'ru': 'Материал по источникам', 'kk': 'Дереккөздерге негізделген мақала', 'en': 'Evidence-based article'}
            marker = getattr(self.server, 'fixture_marker', '')
            value = {field: {locale: text + marker for locale, text in titles.items()} for field in ('title', 'lead', 'body', 'summary', 'actions')}
            source_data=json.loads(instruction.split('<SOURCE_DATA>\n',1)[1].rsplit('\n</SOURCE_DATA>',1)[0])
            value['opportunityIds']=[source_data['opportunities'][0]['id']] if source_data.get('opportunities') else []
        result={'choices':[{'message':{'content':json.dumps(value)}}],'usage':{'prompt_tokens':100,'completion_tokens':100}}
        self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(json.dumps(result).encode())

class RealTransportTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server=ThreadingHTTPServer(('127.0.0.1',0),FixtureHandler)
        cls.thread=threading.Thread(target=cls.server.serve_forever,daemon=True);cls.thread.start()
    @classmethod
    def tearDownClass(cls):cls.server.shutdown();cls.server.server_close()
    def test_actual_rss_transport_requires_explicit_fixture_network_flag(self):
        source={'type':'rss','url':f'http://127.0.0.1:{self.server.server_port}/feed','locale':'kk'}
        with patch.dict(os.environ,{'CREATA_ALLOW_PRIVATE_SOURCES':'0'}),self.assertRaises(UnsafeURL):collect(source,check=True)
        with patch.dict(os.environ,{'CREATA_ALLOW_PRIVATE_SOURCES':'1','CREATA_ENV':'test'}):
            result=collect(source,check=True)
        self.assertEqual(result['items'][0]['title'],'Actual fixture')
    def test_drafting_reserves_each_request_and_retains_source_evidence(self):
        endpoint=f'http://127.0.0.1:{self.server.server_port}/v1'
        class API:
            def __init__(self):self.calls=[]
            def get(self,path):return {'endpoint':endpoint,'model':'fixture','apiKey':'fixture','dailyBudgetUsd':5,'inputPricePerMillion':1,'outputPricePerMillion':2}
            def post(self,path,value):self.calls.append((path,value));return {'reservationId':str(len(self.calls))}
        api=API();result=Pipeline(api).process({'kind':'draft','payload':{'story':{'url':'https://example.com/source','text':'Fixture evidence','sources':[{'url':'https://example.com/source','title':'Source'}]}}})
        self.assertEqual(result['sources'][0]['url'],'https://example.com/source')
        self.assertEqual(len([p for p,_ in api.calls if p.endswith('/reserve')]),3)
        self.assertEqual(len([p for p,_ in api.calls if p.endswith('/commit')]),3)
        self.assertIn('Single source',result['reviewNotes'][0])

    def test_provider_redirect_never_forwards_authorization(self):
        seen=[]
        class Redirect(BaseHTTPRequestHandler):
            def log_message(self,*args):pass
            def do_POST(self):
                self.rfile.read(int(self.headers.get('Content-Length','0')))
                self.send_response(302);self.send_header('Location',f'http://127.0.0.1:{self.server.server_port}/leak');self.send_header('Content-Length','0');self.end_headers()
            def do_GET(self):seen.append(self.headers.get('Authorization'));self.send_response(200);self.end_headers()
        server=ThreadingHTTPServer(('127.0.0.1',0),Redirect)
        threading.Thread(target=server.serve_forever,daemon=True).start()
        class API:
            def get(self,path):return {'endpoint':f'http://127.0.0.1:{server.server_port}/v1','model':'fixture','apiKey':'private-fixture-key','dailyBudgetUsd':1,'inputPricePerMillion':1,'outputPricePerMillion':1}
            def post(self,path,payload):return {'reservationId':'r'}
        try:
            import urllib.error
            with self.assertRaises(urllib.error.HTTPError):AIProvider(API()).complete('Classify',{},100)
            self.assertEqual(seen,[])
        finally:server.shutdown();server.server_close()

if __name__ == '__main__': unittest.main()
