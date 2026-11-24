import copy
import io
import json
import os
import socket
import sys
import tempfile
import unittest
from email.message import Message
from pathlib import Path
from unittest.mock import patch
from urllib.error import HTTPError
from urllib.response import addinfourl

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from hermes.accounting import save_record
from hermes.collectors import page
from hermes.net import UnsafeURL
from hermes.pipeline import Pipeline


STORY_URL = 'https://project.example/release'
HEADLINE = 'Godot releases Vulkan editor update'
FIRST_RESULT = 'https://news.example/godot?utm_source=exa#top'
FIRST_FINAL = 'https://news.example/godot'
SECOND_RESULT = 'https://journal.example/news'
PUBLIC_ORIGINAL = {'url': STORY_URL, 'title': HEADLINE, 'text': 'The public original article.'}


def public_dns(host, port, **kwargs):
    addresses = {'bridge.example': '127.0.0.1'}
    return [(socket.AF_INET, socket.SOCK_STREAM, 6, '', (addresses.get(host, '1.1.1.1'), port))]


class FixtureAPI:
    def __init__(self, exa_key='fixture-exa-secret'):
        self.config = {} if exa_key is None else {'exaApiKey': exa_key}
        self.posts = []

    def get(self, path):
        if path != '/internal/ai-config':
            raise AssertionError('Unexpected config endpoint')
        return self.config

    def post(self, path, data):
        self.posts.append((path, data))
        if path == '/internal/ai/reserve':
            return {'reservationId': 'fixture-reservation-'+str(len(self.posts))}
        return {}


class FixtureProvider:
    def __init__(self, api):
        self.config = api.get('/internal/ai-config')

    def complete(self, task, data, max_tokens):
        if task.startswith('Research'):
            return {'facts': [{'claim': 'Godot released a Vulkan editor update',
                               'sourceUrls': [entry['url'] for entry in data['retrievedEvidence']]}],
                    'conflicts': [], 'singleSourceClaims': []}
        if task.startswith('Review'):
            return {'reviewNotes': []}
        return {field: dict.fromkeys(('ru', 'kk', 'en'), 'Fixture draft')
                for field in ('title', 'lead', 'body', 'summary', 'actions')}


class ExaResearchTests(unittest.TestCase):
    def setUp(self):
        self.job = {'kind': 'draft', 'payload': {
            'story': {'url': STORY_URL, 'title': HEADLINE, 'text': 'Full source text is never sent to search.',
                      'sourceId': 'collector', 'sourceType': 'rss', 'sources': [{'url': STORY_URL}]},
            'article': {'title': {'en': 'Private editorial draft'}}, 'note': 'Private editorial note',
        }}
        self.api = FixtureAPI()

    def run_job(self, response=None, pages=None, *, error=None, job=None):
        available_pages = {STORY_URL: PUBLIC_ORIGINAL, **(pages or {})}

        def get_page(url, **kwargs):
            value = available_pages[url]
            if isinstance(value, Exception):
                raise value
            return value

        with patch('hermes.pipeline.AIProvider', FixtureProvider), \
             patch('hermes.net.socket.getaddrinfo', side_effect=public_dns), \
             patch('hermes.pipeline.page', side_effect=get_page) as crawl, \
             patch('urllib.request.OpenerDirector.open') as request:
            if error:
                request.side_effect = error
            else:
                request.return_value = io.BytesIO(json.dumps(response).encode())
            result = Pipeline(self.api).process(copy.deepcopy(self.job) if job is None else job)
        return result, request, crawl

    def test_searches_once_by_public_headline_then_cites_only_crawled_evidence(self):
        self.job['kind'] = 'rework'
        story = self.job['payload']['story']
        retry_url = 'https://www.project.example/release?utm_medium=rss'
        story['sources'].append({'url': retry_url})
        self.job['payload']['article']['sources'] = story['sources']
        result, request, crawl = self.run_job(
            {'results': [{'url': FIRST_RESULT, 'title': 'Exa title is ignored', 'text': 'Exa snippet is ignored'},
                         {'url': 'https://www.news.example/second'}, {'url': SECOND_RESULT},
                         {'url': 'https://extra.example/godot'}]},
            {retry_url: {'url': STORY_URL, 'text': story['text']},
             FIRST_FINAL: {'url': FIRST_FINAL, 'title': HEADLINE, 'text': 'The Godot editor Vulkan update is now out.',
                           'publishedAt': '2026-11-20T00:00:00Z'},
             SECOND_RESULT: {'url': SECOND_RESULT, 'title': 'Vulkan comes to Godot',
                             'text': 'The Godot Vulkan editor update is now available.'}},
        )
        self.assertEqual(request.call_count, 1)
        sent = request.call_args.args[0]
        self.assertEqual(sent.full_url, 'https://api.exa.ai/search')
        self.assertEqual(sent.get_method(), 'POST')
        self.assertEqual(sent.get_header('X-api-key'), 'fixture-exa-secret')
        self.assertEqual(sent.get_header('Content-type'), 'application/json')
        self.assertLessEqual(request.call_args.kwargs['timeout'], 10)
        body = json.loads(sent.data)
        self.assertEqual(body, {'query': HEADLINE, 'type': 'auto', 'numResults': 5,
                                'excludeDomains': ['project.example']})
        self.assertEqual([c.args[0] for c in crawl.call_args_list], [retry_url, STORY_URL, FIRST_FINAL, SECOND_RESULT])
        partner_evidence = [entry for entry in result['researchBrief']['retrievedEvidence']
                            if entry.get('discoveredBy') == 'exa']
        self.assertEqual(partner_evidence, [
            {'url': FIRST_FINAL, 'title': HEADLINE, 'text': 'The Godot editor Vulkan update is now out.',
             'publishedAt': '2026-11-20T00:00:00Z', 'discoveredBy': 'exa'},
            {'url': SECOND_RESULT, 'title': 'Vulkan comes to Godot',
             'text': 'The Godot Vulkan editor update is now available.', 'discoveredBy': 'exa'},
        ])
        self.assertEqual(result['researchBrief']['facts'][0]['sourceUrls'][-2:], [FIRST_FINAL, SECOND_RESULT])
        self.assertEqual(result['sources'], story['sources'])

    def test_missing_exa_key_preserves_single_source_research(self):
        self.api = FixtureAPI(exa_key=None)
        result, request, _ = self.run_job()
        request.assert_not_called()
        self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [STORY_URL])

    def test_private_or_manual_story_is_not_sent_to_exa(self):
        with patch.dict(os.environ, {'CREATA_ALLOW_PRIVATE_SOURCES': '1', 'CREATA_ENV': 'test'}):
            for change in ({'url': 'http://127.0.0.1/a'}, {'url': 'https://bridge.example/release'},
                           {'url': 'https://user:password@project.example/release'}, {'sourceId': ''},
                           {'title': None}, {'sourceType': 'manual'},
                           {'sourceType': 'telegram', 'url': 'https://t.me/c/1234/42'}):
                with self.subTest(change=change):
                    job = copy.deepcopy(self.job)
                    story = job['payload']['story']
                    story.update(change)
                    story['sources'][0]['url'] = story['url']
                    result, request, _ = self.run_job(job=job)
                    request.assert_not_called()
                    self.assertEqual(len(result['researchBrief']['retrievedEvidence']), 1)

    def test_existing_second_domain_skips_exa(self):
        self.job['payload']['story']['evidence'] = [
            {'url': SECOND_RESULT, 'title': HEADLINE, 'text': 'Godot Vulkan editor update confirmed.'},
        ]
        _, request, _ = self.run_job()
        request.assert_not_called()

    def test_private_rss_headline_on_public_domain_is_not_sent_for_search(self):
        private_sources = [
            (STORY_URL, {**PUBLIC_ORIGINAL, 'title': 'The public page has a different title'}),
            (STORY_URL, AssertionError('Invalid HTML declaration')),
            (STORY_URL+'?access=private-fixture-token', PUBLIC_ORIGINAL),
        ]
        for original, response in private_sources:
            with self.subTest(original=original, response=response):
                job = copy.deepcopy(self.job)
                job['payload']['story']['url'] = original
                job['payload']['story']['sources'] = [{'url': original}]
                result, request, crawl = self.run_job(pages={original: response}, job=job)
                request.assert_not_called()
                if '?' in original:
                    crawl.assert_not_called()
                self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [original])

    def test_rework_uses_editor_sources_instead_of_old_collected_sources(self):
        self.job['kind'] = 'rework'
        self.job['payload']['article']['sources'] = [{'url': SECOND_RESULT}]
        result, request, crawl = self.run_job(pages={SECOND_RESULT: {
            'url': SECOND_RESULT, 'title': HEADLINE, 'text': 'The editor’s selected source.',
        }})
        request.assert_not_called()
        self.assertEqual([c.args[0] for c in crawl.call_args_list], [SECOND_RESULT])
        self.assertEqual(result['sources'], [{'url': SECOND_RESULT}])
        self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [SECOND_RESULT])

    def test_rework_respects_a_deliberately_empty_editor_source_list(self):
        self.job['kind'] = 'rework'
        self.job['payload']['article']['sources'] = []
        with self.assertRaisesRegex(ValueError, 'No retrievable source evidence'):
            self.run_job({'results': []})

    def test_provider_failure_cannot_disclose_key_or_replace_original_evidence(self):
        cases = (HTTPError('https://api.exa.ai/search', 429, 'fixture-exa-secret', {}, io.BytesIO()),
                 TimeoutError('fixture-exa-secret'))
        for case in cases:
            with self.subTest(case=type(case)), self.assertLogs(level='WARNING') as warnings:
                result, request, _ = self.run_job(error=case)
                self.assertEqual(request.call_count, 1)
                self.assertNotIn('fixture-exa-secret', '\n'.join(warnings.output) + json.dumps(result))
                self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [STORY_URL])

    def test_provider_redirect_never_forwards_credentials(self):
        headers = Message()
        headers['Location'] = 'https://redirect.example/collect'

        def redirect(request):
            response = addinfourl(io.BytesIO(), headers, request.full_url, code=307)
            response.msg = 'Temporary Redirect'
            return response

        with patch('hermes.pipeline.AIProvider', FixtureProvider), \
             patch('hermes.net.socket.getaddrinfo', side_effect=public_dns), \
             patch('hermes.pipeline.page', return_value=PUBLIC_ORIGINAL), \
             patch('urllib.request.HTTPSHandler.https_open', side_effect=redirect) as request, \
             self.assertLogs(level='WARNING'):
            result = Pipeline(self.api).process(self.job)
        self.assertEqual([c.args[0].full_url for c in request.call_args_list], ['https://api.exa.ai/search'])
        self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [STORY_URL])

    def test_strict_page_validates_dns_at_each_redirect(self):
        with patch.dict(os.environ, {'CREATA_ALLOW_PRIVATE_SOURCES': '1', 'CREATA_ENV': 'test'}), \
             patch('hermes.net.socket.getaddrinfo', side_effect=public_dns), \
             patch('hermes.net.socket.create_connection') as connect, \
             patch('hermes.net.http.client.HTTPConnection') as connection:
            conn = connection.return_value
            conn.getresponse.side_effect = [conn.response_robots, conn.response_redirect]
            conn.response_robots.status = 200
            conn.response_robots.read.return_value = b'User-agent: *\nAllow: /'
            conn.response_redirect.status = 302
            conn.response_redirect.getheader.return_value = 'http://bridge.example/unsafe'
            with self.assertRaises(UnsafeURL):
                page('http://redirect.example/article', strict_public=True)
        self.assertEqual([call.args[0] for call in connect.call_args_list], [('1.1.1.1', 80)] * 2)

    def test_drops_private_redirect_duplicates_unreadable_and_unrelated_results(self):
        result, _, crawl = self.run_job(
            {'results': [{'url': 'file:///tmp/record'}, {'url': 'http://127.0.0.1/a'},
                         {'url': 'https://redirect.example/godot'}, {'url': FIRST_FINAL, 'text': 'Godot Vulkan'},
                         {'url': SECOND_RESULT, 'text': 'Godot Vulkan'}]},
            {'https://redirect.example/godot': {'url': STORY_URL+'#dup', 'title': HEADLINE, 'text': 'Godot Vulkan'},
             FIRST_FINAL: {'url': FIRST_FINAL, 'title': HEADLINE, 'text': ' '},
             SECOND_RESULT: {'url': SECOND_RESULT, 'title': 'Baking recipes', 'text': 'Mix flour and butter.'}},
        )
        self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [STORY_URL])
        self.assertEqual([c.args[0] for c in crawl.call_args_list],
                         [STORY_URL, 'https://redirect.example/godot', FIRST_FINAL, SECOND_RESULT])
        self.assertTrue(all(c.kwargs.get('strict_public') for c in crawl.call_args_list))

    def test_malformed_and_null_containing_candidate_pages_are_optional(self):
        result, _, _ = self.run_job({'results': [{'url': FIRST_FINAL}, {'url': SECOND_RESULT}]}, {
            FIRST_FINAL: AssertionError('Invalid HTML declaration'),
            SECOND_RESULT: {'url': SECOND_RESULT, 'title': HEADLINE, 'text': 'The Godot Vulkan update\x00 is out.'},
        })
        self.assertEqual([entry['url'] for entry in result['researchBrief']['retrievedEvidence']], [STORY_URL])

    def test_long_raw_story_does_not_hide_exa_evidence_in_provider_inputs(self):
        self.api.config.update(endpoint='https://ai.example/v1', model='fixture', apiKey='model-key', dailyBudgetUsd=50,
                               inputPricePerMillion=1, outputPricePerMillion=2)
        self.job['payload']['story'].update(text='RAW DUPLICATE ' * 24_000, evidence=[PUBLIC_ORIGINAL])
        candidate = {'url': FIRST_FINAL, 'title': HEADLINE, 'text': 'The Godot editor Vulkan update is now out.'}
        model_inputs = {}

        def provider_reply(request, timeout):
            if request.full_url == 'https://api.exa.ai/search':
                return io.BytesIO(json.dumps({'results': [{'url': FIRST_FINAL}]}).encode())
            instruction = json.loads(request.data)['messages'][1]['content']
            phase = instruction.split()[0]
            content = instruction.split('<SOURCE_DATA>\n', 1)[1].rsplit('\n</SOURCE_DATA>', 1)[0]
            if phase in ('Research', 'Write'):
                model_inputs[phase] = json.loads(content)
            data = model_inputs.get(phase, {})
            result = FixtureProvider(self.api).complete(instruction, data, 0)
            return io.BytesIO(json.dumps({'choices': [{'message': {'content': json.dumps(result)}}]}).encode())

        with patch('hermes.net.socket.getaddrinfo', side_effect=public_dns), \
             patch('hermes.pipeline.page', side_effect=lambda url, **kwargs: {
                 STORY_URL: PUBLIC_ORIGINAL, FIRST_FINAL: candidate,
             }[url]), \
             patch('urllib.request.OpenerDirector.open', side_effect=provider_reply):
            Pipeline(self.api).process(self.job)
        self.assertEqual(model_inputs['Research']['retrievedEvidence'][1]['url'], FIRST_FINAL)
        self.assertEqual(model_inputs['Write']['evidence']['retrievedEvidence'][1]['url'], FIRST_FINAL)
        self.assertNotIn('RAW DUPLICATE', json.dumps(model_inputs))

    def test_reuses_successful_and_recoverable_paid_research(self):
        original_evidence = {'url': STORY_URL, 'title': HEADLINE, 'text': 'Previously retrieved text'}
        prior_result = {'facts': [{'claim': 'A saved fact', 'sourceUrls': [STORY_URL]}],
                        'conflicts': [], 'singleSourceClaims': []}
        job = copy.deepcopy(self.job)
        job['checkpoints'] = {'research': {'status': 'succeeded', 'output': {
            **prior_result, 'retrievedEvidence': [original_evidence],
        }}}
        result, request, crawl = self.run_job(job=job)
        request.assert_not_called()
        crawl.assert_not_called()
        self.assertEqual(result['researchBrief']['retrievedEvidence'], [original_evidence])

        with tempfile.TemporaryDirectory() as temporary, patch.dict(os.environ, {'CREATA_WORKER_STATE_DIR': temporary}):
            job = {**self.job, 'id': 'paid-research', 'leaseToken': 'fixture-lease'}
            save_record({'jobId': job['id'], 'phase': 'research', 'accounted': False,
                         'output': prior_result, 'usage': {'reservationId': 'fixture-reservation'},
                         'inputData': {'retrievedEvidence': [original_evidence]}})
            result, request, crawl = self.run_job(job=job)
            request.assert_not_called()
            crawl.assert_not_called()
            self.assertEqual(result['researchBrief']['retrievedEvidence'], [original_evidence])
            self.assertIn(('/internal/ai/commit', {'reservationId': 'fixture-reservation'}), self.api.posts)


if __name__ == '__main__':
    unittest.main()
