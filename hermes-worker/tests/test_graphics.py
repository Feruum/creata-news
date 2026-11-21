import sys
import unittest
from pathlib import Path
from copy import deepcopy
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from hermes.graphics import validate_graphic_result,source_number,draft_graphic

def loc(v):return {l:v for l in ('ru','kk','en')}
def fixture():
    url='https://example.org/report'
    evidence=[{'url':url,'text':'Revenue in 2024 was 100 million USD. Revenue in 2025 was 120 million USD.'}]
    result={'graphic':{'type':'bar','unit':'million USD','period':'2024–2025','sourceUrl':url,'sourceName':'Report','methodology':loc('Reported annual revenue.'),'labels':[loc('2024'),loc('2025')],'series':[{'name':loc('Revenue'),'values':[100,120]}],'verified':True},'evidence':[{'row':0,'series':0,'quote':'Revenue in 2024 was 100 million USD.','numberText':'100','sourceUrl':url},{'row':1,'series':0,'quote':'Revenue in 2025 was 120 million USD.','numberText':'120','sourceUrl':url}]}
    return result,evidence

class GraphicTests(unittest.TestCase):
    def test_graphic_pipeline_links_supplied_opportunities_and_exposes_numeric_brief(self):
        from hermes.pipeline import Pipeline
        extracted,evidence=fixture();calls=[]
        class Provider:
            def __init__(self,api):pass
            def complete(self,task,data,*args):
                calls.append(data)
                if len(calls)==1:return deepcopy(extracted)
                if len(calls)==2:return {**{key:loc('Source-backed explanation') for key in ('title','lead','body','summary','actions')},'opportunityIds':['grant-one']}
                return {'reviewNotes':[]}
        unavailable='https://unavailable.example/source'
        job={'kind':'draft','payload':{'article':{'format':'graphic'},'story':{'evidence':evidence,'sources':[{'url':evidence[0]['url']},{'url':unavailable}]},'opportunities':[{'id':'grant-one','title':loc('Apply')}]}}
        with patch('hermes.pipeline.AIProvider',Provider),patch('hermes.pipeline.page',side_effect=OSError('Source unavailable')):result=Pipeline(object()).process(job)
        self.assertEqual(calls[0]['unavailableSources'],[unavailable])
        self.assertEqual(result['researchBrief']['unavailableSources'],[unavailable])
        self.assertEqual(calls[1]['opportunities'][0]['id'],'grant-one')
        self.assertEqual(result['opportunityIds'],['grant-one'])
        self.assertTrue(result['researchBrief']['facts'])
        self.assertEqual(result['researchBrief']['facts'][0]['numbers'][0]['value'],'100')
        self.assertEqual(result['researchBrief']['facts'][0]['sourceUrls'],[evidence[0]['url']])

    def test_graphic_rejects_invented_opportunities_before_review(self):
        extracted,evidence=fixture()
        class Provider:
            calls=0
            def complete(self,*args):
                self.calls+=1
                if self.calls==1:return deepcopy(extracted)
                if self.calls==2:return {**{key:loc('Explanation') for key in ('title','lead','body','summary','actions')},'opportunityIds':['invented']}
                raise AssertionError('Invalid opportunity identifiers reached review')
        with self.assertRaises(ValueError):draft_graphic(Provider(),{}, {},[{'url':evidence[0]['url']}],evidence,'',opportunities=[{'id':'grant-one'}])
    def test_each_value_must_match_literal_retrieved_source(self):
        original,evidence=fixture()
        for mode in ('invented_number','fabricated_quote','missing_cell','wrong_url','object_url','object_proof_url','non_numeric','duplicate_cell'):
            result=deepcopy(original)
            if mode=='invented_number':result['graphic']['series'][0]['values'][1]=150
            if mode=='fabricated_quote':result['evidence'][1]['quote']='Revenue was 150 million USD.'
            if mode=='missing_cell':result['evidence'].pop()
            if mode=='wrong_url':result['graphic']['sourceUrl']='https://unretrieved.example/report'
            if mode=='object_url':result['graphic']['sourceUrl']={}
            if mode=='object_proof_url':result['evidence'][0]['sourceUrl']={}
            if mode=='non_numeric':result['graphic']['series'][0]['values'][0]=True
            if mode=='duplicate_cell':result['evidence'].append(result['evidence'][0])
            with self.subTest(mode=mode),self.assertRaises(ValueError):validate_graphic_result(result,evidence)
        self.assertFalse(validate_graphic_result(original,evidence)['verified'])

    def test_numeric_grouping_preserves_source_scale(self):
        from decimal import Decimal
        self.assertEqual(source_number('1,234.5'),Decimal('1234.5'))
        self.assertEqual(source_number('1 234,5'),Decimal('1234.5'))
        self.assertEqual(source_number('2.5'),Decimal('2.5'))

    def test_graphic_pipeline_preserves_review_gate_and_three_translations(self):
        extracted,evidence=fixture()
        class Provider:
            def __init__(self):self.calls=0
            def complete(self,*args):
                self.calls+=1
                if self.calls==1:return extracted
                if self.calls==2:return {key:loc('Source-backed explanation') for key in ('title','lead','body','summary','actions')}
                return {'reviewNotes':['Check period comparability']}
        provider=Provider();out=draft_graphic(provider,{}, {},[{'url':evidence[0]['url']}],evidence,'')
        self.assertEqual(provider.calls,3);self.assertFalse(out['graphic']['verified']);self.assertIn('kk',out['body']);self.assertIn('human',out['reviewNotes'][0])

if __name__=='__main__':unittest.main()
