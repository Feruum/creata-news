import copy
import unittest
import sys
from pathlib import Path
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from hermes.pipeline import Pipeline, validate_research, source_evidence

class CheckpointTests(unittest.TestCase):
    def test_research_evidence_is_bounded_for_persistent_checkpoint_transport(self):
        evidence,_=source_evidence({'evidence':[{'url':f'https://example.com/{i}','text':'Факты '*50000} for i in range(10)]},{},[])
        self.assertLessEqual(sum(len(e['text']) for e in evidence),120000)
        self.assertTrue(all(len(e['text'])<=20000 for e in evidence))
    def test_research_rejects_unretrieved_provenance_and_unsupported_numbers(self):
        evidence=[{'url':'https://example.com/source','text':'The team raised 12 million in 2026.'}]
        for fact in [{'claim':'A claim','sourceUrls':['https://invented.example/a']},{'claim':'A claim','sourceUrls':[{}]},{'claim':'Raised 99 million','sourceUrls':['https://example.com/source'],'numbers':[{'value':'99','sourceUrl':'https://example.com/source','quote':'Raised 99 million'}]}]:
            with self.assertRaises(ValueError):validate_research({'facts':[fact],'conflicts':[],'singleSourceClaims':[]},evidence,[])

    def test_retry_uses_successful_research_and_draft_without_provider_reinvocation(self):
        class API:
            def __init__(self):self.checkpoints={};self.calls=[]
            def post(self,path,data):
                self.calls.append((path,copy.deepcopy(data)))
                if path.endswith('/checkpoint') and data['status']=='succeeded':
                    self.checkpoints[data['phase']]={'status':'succeeded','output':copy.deepcopy(data['output'])}
                return {}
        class Provider:
            count={'research':0,'draft':0,'review':0}
            config={}
            def __init__(self,api):pass
            def complete(self,task,data,max_tokens):
                phase='research' if task.startswith('Research') else 'review' if task.startswith('Review') else 'draft'
                self.count[phase]+=1
                if phase=='research':return {'facts':[{'claim':'A finding','sourceUrls':['https://example.com/source']}],'conflicts':[],'singleSourceClaims':['A finding']}
                if phase=='review':
                    if self.count[phase]==1:raise TimeoutError('interrupted review')
                    return {'reviewNotes':[]}
                return {field:{l:'A factual draft' for l in ('ru','kk','en')} for field in ('title','lead','body','summary','actions')}
        api=API();job={'id':'job','leaseToken':'lease','kind':'draft','payload':{'story':{'url':'https://example.com/source','text':'A finding','sources':[{'url':'https://example.com/source'}]}}}
        with patch('hermes.pipeline.AIProvider',Provider):
            with self.assertRaises(TimeoutError):Pipeline(api).process(job)
            job['checkpoints']=copy.deepcopy(api.checkpoints)
            result=Pipeline(api).process(job)
        self.assertEqual(Provider.count,{'research':1,'draft':1,'review':2})
        self.assertEqual(result['researchBrief']['facts'][0]['claim'],'A finding')
        self.assertEqual(result['opportunityIds'],[])
        self.assertTrue(any(data['phase']=='review' and data['status']=='failed' for _,data in api.calls))
