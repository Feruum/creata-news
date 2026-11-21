import copy
import unittest
import sys
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from hermes.pipeline import validate_classification

URL='https://example.org/source'
STORY={'url':URL,'text':'Creata Lab launched Orbit One in 2026. The prototype reached 120 users.'}
def classification():
    return {'categoryId':'ai','region':'KZ','tags':['research'],'score':85,'reason':'Relevant research','facts':[{'claim':'The prototype reached 120 users.','sourceUrls':[URL],'uncertainty':'Reported by one source'}],'entities':[{'name':'Creata Lab','type':'organization','sourceUrls':[URL]}]}

class ClassificationTests(unittest.TestCase):
    def test_category_must_belong_to_supplied_taxonomy_before_checkpoint(self):
        with self.assertRaises(ValueError):validate_classification(classification(),STORY,categories=[{'id':'science'}])
        self.assertEqual(validate_classification(classification(),STORY,categories=[{'id':'ai'}])['categoryId'],'ai')
    def test_classification_rejects_malformed_or_unsupported_claims_and_entities(self):
        for change in (
            {'facts':'not an array'}, {'entities':['Creata Lab']},
            {'facts':[{'claim':'The prototype reached 999 users.','sourceUrls':[URL],'uncertainty':''}]},
            {'facts':[{'claim':'The prototype reached 120 users.','sourceUrls':['https://invented.example/source'],'uncertainty':''}]},
            {'facts':[{'claim':'The prototype reached 120 users.','sourceUrls':[URL]}]},
            {'entities':[{'name':'Invented Labs','type':'organization','sourceUrls':[URL]}]},
            {'entities':[{'name':'Creat','type':'organization','sourceUrls':[URL]}]},
            {'entities':[{'name':'Creata Lab','type':'unknown','sourceUrls':[URL]}]},
            {'tags':[{'bad':'type'}]}, {'score':True}):
            with self.subTest(change=change),self.assertRaises(ValueError):validate_classification({**classification(),**change},STORY)
        self.assertEqual(validate_classification(classification(),STORY)['entities'][0]['name'],'Creata Lab')

    def test_legacy_omitted_lists_are_explicitly_normalized(self):
        result=classification();del result['facts'];del result['entities']
        with self.assertRaises(ValueError):validate_classification(copy.deepcopy(result),STORY)
        normalized=validate_classification(result,STORY,legacy=True)
        self.assertEqual(normalized['facts'],[]);self.assertEqual(normalized['entities'],[])
