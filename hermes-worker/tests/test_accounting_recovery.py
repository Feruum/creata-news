import copy
import os
import tempfile
import threading
import unittest
import sys
from pathlib import Path
from http.server import ThreadingHTTPServer
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from hermes.api import APIError
from hermes.pipeline import Pipeline
from test_pipeline import FixtureHandler

class AccountingRecoveryTests(unittest.TestCase):
    def setUp(self):
        self.directory=tempfile.TemporaryDirectory();self.environment=patch.dict(os.environ,{'CREATA_WORKER_STATE_DIR':self.directory.name});self.environment.start()
        self.server=ThreadingHTTPServer(('127.0.0.1',0),FixtureHandler);self.server.fixture_counts={};threading.Thread(target=self.server.serve_forever,daemon=True).start()
    def tearDown(self):self.server.shutdown();self.server.server_close();self.environment.stop();self.directory.cleanup()
    def api(self):
        endpoint=f'http://127.0.0.1:{self.server.server_port}/v1'
        class API:
            def __init__(self):self.checkpoints={};self.fail_commit=True;self.fail_checkpoint=0;self.reservations=0;self.commits=[];self.configured=True;self.checkpoint_status=503
            def get(self,path):return {'endpoint':endpoint,'model':'fixture','apiKey':'fixture' if self.configured else '','dailyBudgetUsd':5,'inputPricePerMillion':1,'outputPricePerMillion':2,'categories':[{'id':'ai'}]}
            def post(self,path,data):
                if path.endswith('/reserve'):self.reservations+=1;return {'reservationId':'reservation-'+str(self.reservations)}
                if path.endswith('/commit'):
                    if self.fail_commit:raise APIError(503,'Accounting transport unavailable')
                    self.commits.append(copy.deepcopy(data));return {'ok':True}
                if path.endswith('/checkpoint'):
                    if data['status']=='succeeded' and self.fail_checkpoint:
                        self.fail_checkpoint-=1;raise APIError(self.checkpoint_status,'Checkpoint transport unavailable')
                    self.checkpoints[data['phase']]={key:copy.deepcopy(data[key]) for key in ('status','output','usage') if key in data};return {'ok':True}
                return {}
        return API()
    def job(self):return {'id':'paid-classification','leaseToken':'lease','kind':'classify','payload':{'story':{'url':'https://example.com/source','text':'Creata Lab reported Evidence.'}}}
    def test_commit_failure_preserves_paid_phase_and_retry_does_not_recall_provider(self):
        api=self.api();job=self.job()
        with self.assertRaises(APIError):Pipeline(api).process(job)
        self.assertEqual(api.checkpoints['classification']['status'],'succeeded')
        self.assertEqual(self.server.fixture_counts,{'classify':1})
        retry=self.job();retry['checkpoints']=copy.deepcopy(api.checkpoints);api.fail_commit=False
        result=Pipeline(api).process(retry)
        self.assertEqual(self.server.fixture_counts,{'classify':1});self.assertEqual(api.reservations,1);self.assertTrue(result['facts']);self.assertEqual(api.commits[-1]['reservationId'],'reservation-1')
    def test_lost_checkpoint_transport_reuses_durable_output_after_new_pipeline(self):
        api=self.api();api.fail_commit=False;api.fail_checkpoint=10;job=self.job()
        with self.assertRaises(APIError):Pipeline(api).process(job)
        self.assertEqual(self.server.fixture_counts,{'classify':1})
        api.fail_checkpoint=0
        result=Pipeline(api).process(self.job())
        self.assertTrue(result['facts']);self.assertEqual(self.server.fixture_counts,{'classify':1});self.assertEqual(api.reservations,1)
    def test_transient_success_checkpoint_failure_retries_same_output(self):
        api=self.api();api.fail_commit=False;api.fail_checkpoint=1
        self.assertTrue(Pipeline(api).process(self.job())['facts'])
        self.assertEqual(self.server.fixture_counts,{'classify':1});self.assertEqual(api.reservations,1)
    def test_changed_source_after_checkpoint_outage_does_not_repeat_paid_research(self):
        api=self.api();api.fail_commit=False;api.fail_checkpoint=10
        job={'id':'paid-research','leaseToken':'lease','kind':'draft','payload':{'article':{'sources':[{'url':'https://example.com/source'}]}}}
        with patch('hermes.pipeline.page',side_effect=[{'url':'https://example.com/source','title':'Source','text':'Fixture evidence T1'},{'url':'https://example.com/source','title':'Source','text':'Completely changed source T2'}]) as fetch:
            with self.assertRaises(APIError):Pipeline(api).process(copy.deepcopy(job))
            api.fail_checkpoint=0
            result=Pipeline(api).process(copy.deepcopy(job))
        self.assertEqual(fetch.call_count,1)
        self.assertEqual(result['researchBrief']['retrievedEvidence'][0]['text'],'Fixture evidence T1')
        self.assertEqual(self.server.fixture_counts,{'research':1,'draft':1,'review':1})
    def test_paid_output_recovers_without_provider_key_after_checkpoint_outage(self):
        api=self.api();api.fail_commit=False;api.fail_checkpoint=10
        with self.assertRaises(APIError):Pipeline(api).process(self.job())
        api.fail_checkpoint=0;api.configured=False
        self.assertTrue(Pipeline(api).process(self.job())['facts'])
        self.assertEqual(self.server.fixture_counts,{'classify':1})
    def test_lost_lease_after_paid_output_still_reconciles_usage(self):
        from hermes.accounting import load_record
        api=self.api();api.fail_commit=False;api.fail_checkpoint=1;api.checkpoint_status=409
        with self.assertRaises(APIError) as error:Pipeline(api).process(self.job())
        self.assertEqual(error.exception.status,409)
        self.assertEqual(api.commits[-1]['reservationId'],'reservation-1')
        self.assertTrue(load_record('paid-classification','classification')['accounted'])
        self.assertEqual(self.server.fixture_counts,{'classify':1})
    def test_malformed_paid_output_is_accounted_and_does_not_poison_manual_retry(self):
        api=self.api();api.fail_commit=False;self.server.fixture_malformed_research=True
        job={'id':'invalid-research','leaseToken':'lease','kind':'draft','payload':{'story':{'url':'https://example.com/source','text':'Fixture evidence'}}}
        with self.assertRaises(ValueError):Pipeline(api).process(copy.deepcopy(job))
        retry=copy.deepcopy(job);retry['checkpoints']=copy.deepcopy(api.checkpoints)
        self.assertTrue(Pipeline(api).process(retry)['researchBrief']['facts'])
        self.assertEqual(self.server.fixture_counts,{'research':2,'draft':1,'review':1})
        self.assertTrue(any(usage['reservationId']=='reservation-1' for usage in api.commits))
    def test_unknown_paid_category_is_rejected_before_immutable_checkpoint_and_retry_recovers(self):
        api=self.api();api.fail_commit=False;self.server.fixture_unknown_category=True
        with self.assertRaises(ValueError):Pipeline(api).process(self.job())
        self.assertEqual(api.checkpoints['classification']['status'],'failed')
        retry=self.job();retry['checkpoints']=copy.deepcopy(api.checkpoints)
        self.assertEqual(Pipeline(api).process(retry)['categoryId'],'ai')
        self.assertEqual(api.checkpoints['classification']['status'],'succeeded')
        self.assertEqual(self.server.fixture_counts,{'classify':2})
