"""Opt-in smoke against a running disposable, fixture-enabled Creata API."""
import json
import os
import sys
import threading
import time
import unittest
import urllib.request
import urllib.error
import http.cookiejar
import uuid
from http.server import ThreadingHTTPServer
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from hermes.api import API
from hermes.pipeline import Pipeline
from hermes.main import run_job
from test_pipeline import FixtureHandler

class FixtureWorker(API):
    def __init__(self,*args):super().__init__(*args);self.leases={}
    def post(self,path,payload):
        result=super().post(path,payload)
        if path=='/internal/jobs/claim' and result.get('job'):self.leases[result['job']['id']]=result['job']['leaseToken']
        return result
    def cleanup(self,request,source,article=None):
        if not source and not article:return
        for job in request('/admin/jobs')['items']:
            payload=job.get('payload') or {}
            owns_source=bool(source and ((payload.get('source') or {}).get('id')==source['id'] or (payload.get('story') or {}).get('sourceId')==source['id']))
            owns_article=bool(article and (payload.get('article') or {}).get('id')==article['id'])
            if not (owns_source or owns_article):continue
            if job['status']=='queued':request('/admin/jobs/'+job['id']+'/cancel',{})
            elif job['status']=='running' and job['id'] in self.leases:
                try:self.post('/internal/jobs/'+job['id']+'/fail',{'leaseToken':self.leases[job['id']],'error':'Fixture cleanup after interrupted assertion','retryable':False})
                except Exception:pass

@unittest.skipUnless(os.getenv('CREATA_TEST_API_ORIGIN'),'Set CREATA_TEST_API_ORIGIN to an explicitly disposable running API')
class LivePipelineTests(unittest.TestCase):
    def test_real_graphic_worker_persists_related_opportunity_and_numeric_research(self):
        origin=os.environ['CREATA_TEST_API_ORIGIN'].rstrip('/');base=origin+'/api/creata/v1';csrf=''
        opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        def request(path,data=None,method=None):
            headers={'Origin':os.getenv('CREATA_TEST_BROWSER_ORIGIN','http://localhost:6868'),'Content-Type':'application/json','X-CSRF-Token':csrf}
            req=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=headers,method=method)
            with opener.open(req,timeout=30) as response:return json.load(response)
        request('/auth/dev-login',{'role':'admin'});csrf=request('/auth/me')['csrfToken'];initial=request('/admin/settings')
        fixture=ThreadingHTTPServer(('127.0.0.1',0),FixtureHandler);fixture.fixture_marker='graphic-'+str(uuid.uuid4())[:8];fixture.fixture_numeric=True;fixture.fixture_counts={}
        threading.Thread(target=fixture.serve_forever,daemon=True).start();worker=FixtureWorker(origin,os.environ['CREATA_WORKER_TOKEN']);source=None;opportunity=None;article=None
        try:
            request('/admin/settings',{'scoreThreshold':70,'ai':{'endpoint':f'http://127.0.0.1:{fixture.server_port}/v1','model':'fixture','apiKey':'fixture-key','dailyBudgetUsd':5,'inputPricePerMillion':1,'outputPricePerMillion':2}},'PATCH')
            loc=lambda text:{lang:text for lang in ('ru','kk','en')}
            opportunity=request('/admin/opportunities',{'title':loc('Related research grant '+fixture.fixture_marker),'description':loc('Apply with a prototype'),'organization':'Creata','type':'grant','country':'KZ','url':'https://example.org/apply/'+fixture.fixture_marker,'deadline':'2099-01-01T00:00:00Z','status':'approved','categoryId':'ai','languages':['en']})
            source=request('/admin/sources',{'name':'Numeric fixture '+fixture.fixture_marker,'type':'rss','url':f'http://127.0.0.1:{fixture.server_port}/feed','locale':'en','topics':['ai'],'intervalMinutes':60,'active':False})
            request('/admin/sources/'+source['id']+'/run',{})
            collect_job=worker.post('/internal/jobs/claim',{'workerId':'graphic-fixture','kinds':['collect']})['job'];self.assertIsNotNone(collect_job)
            worker.post('/internal/jobs/'+collect_job['id']+'/complete',{'leaseToken':collect_job['leaseToken'],'result':Pipeline(worker).process(collect_job)})
            classification=worker.post('/internal/jobs/claim',{'workerId':'graphic-fixture','kinds':['classify']})['job'];self.assertIsNotNone(classification)
            classified=Pipeline(worker).process(classification);self.assertTrue(classified['facts']);self.assertTrue(classified['entities'])
            # A low-scoring classification keeps this fixture available for a manual graphic.
            classified['score']=10
            worker.post('/internal/jobs/'+classification['id']+'/complete',{'leaseToken':classification['leaseToken'],'result':classified})
            story=next(story for story in request('/admin/bank')['items'] if story['title']=='Actual fixture'+fixture.fixture_marker)
            self.assertEqual(story['facts'][0]['sourceUrls'],[story['url']]);self.assertEqual(story['entities'][0]['name'],'Creata Lab')
            article=request('/admin/bank/'+story['id']+'/graphic',{})
            job=worker.post('/internal/jobs/claim',{'workerId':'graphic-fixture','kinds':['draft']})['job'];self.assertEqual(job['payload']['article']['id'],article['id']);self.assertTrue(any(o['id']==opportunity['id'] for o in job['payload']['opportunities']))
            run_job(worker,job)
            updated=request('/admin/articles/'+article['id']);self.assertEqual(updated['status'],'review');self.assertFalse(updated['graphic']['verified']);self.assertEqual(updated['opportunityIds'],[opportunity['id']])
            brief=updated['researchBrief'];self.assertEqual(brief['facts'][0]['numbers'][0]['value'],'100');self.assertEqual(brief['facts'][0]['dates'][0]['value'],'2024');self.assertEqual(brief['facts'][0]['sourceUrls'],[story['url']])
            self.assertEqual(fixture.fixture_counts,{'classify':1,'research':1,'draft':1,'review':1})
        finally:
            worker.cleanup(request,source,article)
            if source:request('/admin/sources/'+source['id'],{'active':False},'PATCH')
            if opportunity:request('/admin/opportunities/'+opportunity['id'],{'status':'rejected'},'PATCH')
            initial['ai']['apiKey']='';request('/admin/settings',initial,'PATCH');fixture.shutdown();fixture.server_close()

    def test_real_worker_retry_resumes_checkpoints_and_rejects_old_lease(self):
        origin=os.environ['CREATA_TEST_API_ORIGIN'].rstrip('/');base=origin+'/api/creata/v1'
        cookie=http.cookiejar.CookieJar();opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookie));csrf=''
        def request(path,data=None,method=None):
            headers={'Content-Type':'application/json','X-CSRF-Token':csrf}
            headers['Origin']=os.getenv('CREATA_TEST_BROWSER_ORIGIN','http://localhost:6868')
            req=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=headers,method=method)
            with opener.open(req,timeout=30) as response:return json.load(response)
        request('/auth/dev-login',{'role':'admin'});csrf=request('/auth/me')['csrfToken'];initial=request('/admin/settings')
        fixture=ThreadingHTTPServer(('127.0.0.1',0),FixtureHandler);fixture.fixture_marker='retry-'+str(uuid.uuid4())[:8];fixture.fixture_counts={};fixture.fixture_fail_review=True
        threading.Thread(target=fixture.serve_forever,daemon=True).start();worker=FixtureWorker(origin,os.environ['CREATA_WORKER_TOKEN']);source=None;article=None
        try:
            request('/admin/settings',{'scoreThreshold':70,'ai':{'endpoint':f'http://127.0.0.1:{fixture.server_port}/v1','model':'fixture','apiKey':'fixture-key','dailyBudgetUsd':5,'inputPricePerMillion':1,'outputPricePerMillion':2}},'PATCH')
            source=request('/admin/sources',{'name':'Checkpoint fixture '+fixture.fixture_marker,'type':'rss','url':f'http://127.0.0.1:{fixture.server_port}/feed','locale':'en','topics':['ai'],'intervalMinutes':60,'active':False})
            request('/admin/sources/'+source['id']+'/run',{})
            for kind in ('collect','classify'):
                job=worker.post('/internal/jobs/claim',{'workerId':'checkpoint-fixture','kinds':[kind]})['job'];self.assertIsNotNone(job)
                worker.post('/internal/jobs/'+job['id']+'/complete',{'leaseToken':job['leaseToken'],'result':Pipeline(worker).process(job)})
            job=worker.post('/internal/jobs/claim',{'workerId':'before-interruption','kinds':['draft']})['job'];self.assertIsNotNone(job);article=job['payload']['article']
            run_job(worker,job)
            stopped=next(j for j in request('/admin/jobs')['items'] if j['id']==job['id'])
            self.assertEqual(stopped['checkpoints']['research']['status'],'succeeded');self.assertEqual(stopped['checkpoints']['draft']['status'],'succeeded');self.assertEqual(stopped['checkpoints']['review']['status'],'failed')
            # The fixture review error is retryable; stop and explicitly retry without waiting for backoff.
            if stopped['status']=='queued':request('/admin/jobs/'+job['id']+'/cancel',{})
            request('/admin/jobs/'+job['id']+'/retry',{})
            resumed=worker.post('/internal/jobs/claim',{'workerId':'after-interruption','kinds':['draft']})['job'];self.assertEqual(resumed['id'],job['id']);self.assertNotEqual(resumed['leaseToken'],job['leaseToken'])
            with self.assertRaises(Exception) as stale:worker.post('/internal/jobs/'+job['id']+'/checkpoint',{'leaseToken':job['leaseToken'],'phase':'review','status':'running'})
            self.assertEqual(getattr(stale.exception,'status',None),409)
            run_job(worker,resumed)
            final=next(j for j in request('/admin/jobs')['items'] if j['id']==job['id']);self.assertEqual(final['status'],'succeeded')
            self.assertEqual(fixture.fixture_counts,{'classify':1,'research':1,'draft':1,'review':2})
            current=request('/admin/articles/'+article['id']);self.assertEqual(current['status'],'review');self.assertTrue(current['researchBrief']['facts'])
            self.assertTrue(all(final['checkpoints'][phase]['status']=='succeeded' for phase in ('research','draft','review')))
            self.assertEqual(request('/admin/articles/'+article['id']+'/research')['items'][0]['id'],job['id'])
        finally:
            worker.cleanup(request,source,article)
            if source:request('/admin/sources/'+source['id'],{'active':False},'PATCH')
            initial['ai']['apiKey']='';request('/admin/settings',initial,'PATCH');fixture.shutdown();fixture.server_close()

    def test_source_through_real_worker_publication_and_digest(self):
        origin=os.environ['CREATA_TEST_API_ORIGIN'].rstrip('/')
        base=origin+'/api/creata/v1'
        self.cookie=http.cookiejar.CookieJar();self.opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.cookie));self.csrf=''
        def request(path,data=None,method=None):
            headers={'Content-Type':'application/json'}
            headers['Origin']=os.getenv('CREATA_TEST_BROWSER_ORIGIN','http://localhost:6868')
            if self.csrf:headers['X-CSRF-Token']=self.csrf
            req=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=headers,method=method)
            try:
                with self.opener.open(req,timeout=30) as res:return json.load(res)
            except urllib.error.HTTPError as err:raise AssertionError(str(err.code)+' '+err.read().decode()) from None
        request('/auth/dev-login',{'role':'admin'})
        self.csrf=request('/auth/me')['csrfToken']
        initial=request('/admin/settings')
        fixture=ThreadingHTTPServer(('127.0.0.1',0),FixtureHandler)
        threading.Thread(target=fixture.serve_forever,daemon=True).start()
        worker=FixtureWorker(origin,os.environ['CREATA_WORKER_TOKEN'])
        marker=str(uuid.uuid4())[:8]
        fixture.fixture_marker=marker
        source=None;article=None
        try:
            request('/admin/settings',{'scoreThreshold':70,'ai':{'endpoint':f'http://127.0.0.1:{fixture.server_port}/v1','model':'local-fixture','apiKey':'local-fixture-key','dailyBudgetUsd':1,'inputPricePerMillion':1,'outputPricePerMillion':2}},'PATCH')
            source=request('/admin/sources',{'name':'Integration RSS '+marker,'type':'rss','url':f'http://127.0.0.1:{fixture.server_port}/feed','locale':'kk','topics':['ai'],'intervalMinutes':60,'active':False})
            source=source.get('source',source)
            request('/admin/sources/'+source['id']+'/run',{})
            for kind in ['collect','classify','draft']:
                job=worker.post('/internal/jobs/claim',{'workerId':'integration-'+marker,'kinds':[kind]})['job']
                self.assertIsNotNone(job,kind+' must be queued')
                result=Pipeline(worker).process(job)
                worker.post('/internal/jobs/'+job['id']+'/complete',{'leaseToken':job['leaseToken'],'result':result})
            items=request('/admin/articles')['items']
            article=next(a for a in items if a['title']['en']=='Evidence-based article'+marker and a['status']=='review')
            self.assertFalse(any(a['id']==article['id'] for a in request('/public/articles')['items']))
            authors=request('/public/authors')['items'];self.assertTrue(authors)
            article=request('/admin/articles/'+article['id'],{'expectedRevision':article['revision'],'authorId':authors[0]['id'],'categoryId':'ai','region':'KZ'},'PATCH')
            article=article.get('article',article)
            for action in ['approve','publish']:
                body={'action':action,'expectedRevision':article['revision']}
                channel=os.getenv('CREATA_TEST_POSTIZ_CHANNEL')
                if channel and action=='publish':
                    body['postizTargets']=[{'channelId':channel,'locale':'kk'}]
                request('/admin/articles/'+article['id']+'/actions',body)
                article=request('/admin/articles/'+article['id'])
            public=request('/public/articles/'+article['slug'])
            self.assertEqual(public['title']['kk'],'Дереккөздерге негізделген мақала'+marker)
            self.assertEqual(public['reviewNotes'],[])
            if channel:
                deadline=time.monotonic()+30
                while True:
                    deliveries=request('/admin/articles/'+article['id']+'/postiz')['items']
                    if deliveries and deliveries[0]['status']=='draft':break
                    if time.monotonic()>=deadline:self.fail('Postiz draft was not confirmed: '+str(deliveries))
                    time.sleep(0.25)
                self.assertEqual(len(deliveries),1)
                self.assertEqual(deliveries[0]['channelId'],channel)
                self.assertEqual(deliveries[0]['revision'],article['revision'])
            settings=request('/admin/settings');self.assertGreater(settings['ai']['spentTodayUsd'],0)
            bot=API(origin,os.environ['CREATA_BOT_SERVICE_TOKEN'])
            telegram_id=str(int('91'+str(uuid.uuid4().int)[:9]))
            user=bot.post('/internal/bot/users/upsert',{'telegramId':telegram_id,'locale':'kk'})
            self.assertEqual(user['locale'],'kk')
            digest=bot.post('/internal/bot/digest',{'telegramId':telegram_id})
            self.assertTrue(any(a['id']==article['id'] for a in digest['articles']))
            bot.post('/internal/bot/digests/'+digest['id']+'/complete',{'telegramId':telegram_id,'messageId':'fixture-message'})
            second=bot.post('/internal/bot/digest',{'telegramId':telegram_id})
            self.assertFalse(any(a['id']==article['id'] for a in second['articles']))
        finally:
            worker.cleanup(request,source,article)
            if article:
                current=request('/admin/articles/'+article['id'])
                request('/admin/articles/'+article['id']+'/actions',{'action':'unpublish','expectedRevision':current['revision']})
            if source:request('/admin/sources/'+source['id'],{'active':False},'PATCH')
            initial['ai']['apiKey']=''
            request('/admin/settings',initial,'PATCH')
            fixture.shutdown();fixture.server_close()

if __name__=='__main__':unittest.main()
