import json
import sys
import unittest
from pathlib import Path
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from hermes.main import run_job


def file_page(name, index=1, author='A. Creator', **overrides):
    stem = name.replace(' ', '_')
    info = {
        'url': f'https://upload.wikimedia.org/wikipedia/commons/9/91/{stem}',
        'thumburl': f'https://thumb.wikimedia.org/wikipedia/commons/thumb/9/91/{stem}/960px-{stem}',
        'descriptionurl': f'https://commons.wikimedia.org/wiki/File:{stem}',
        'mime': 'image/png', 'width': 1280, 'height': 720, 'size': 250_000,
        'thumbwidth': 960, 'thumbheight': 540, 'sha1': str(index),
        'extmetadata': {
            'Artist': {'value': author},
            'ImageDescription': {'value': 'An editor screenshot.'},
            'LicenseUrl': {'value': 'https://creativecommons.org/licenses/by-sa/4.0/'},
            'LicenseShortName': {'value': 'CC BY-SA 4.0'},
        },
    }
    info.update(overrides)
    return {'ns': 6, 'title': 'File:' + name, 'index': index, 'imageinfo': [info]}


def commons_response(*pages):
    return json.dumps({'query': {'pages': list(pages)}}).encode(), 'application/json', 'https://commons.wikimedia.org/w/api.php'


class CompletionAPI:
    def __init__(self):
        self.posts = {}

    def post(self, path, data):
        self.posts[path.rsplit('/', 1)[-1]] = data
        return {}


STORY = {
    'title': 'Godot 4.6 launches with editor improvements',
    'url': 'https://godotengine.org/article/godot-4-6/',
    'text': 'Non-public body fixture 42',
}
ENTITIES = [{'name': 'Godot', 'type': 'product'}, {'name': 'Shared Foundation', 'type': 'organization'}]


class WorkerPhotoTests(unittest.TestCase):
    def test_classify_worker_completes_with_source_backed_photos(self):
        api = CompletionAPI()
        job = {'id': 'job-photo-classify', 'leaseToken': 'lease', 'kind': 'classify', 'payload': {'story': STORY},
               'checkpoints': {'classification': {'status': 'succeeded', 'output': {'score': 35, 'entities': ENTITIES}}}}

        def fetch(url, *, max_bytes, redirects, timeout, user_agent):
            parsed = urlsplit(url)
            self.assertEqual(parsed.netloc, 'commons.wikimedia.org')
            query = parse_qs(parsed.query)
            self.assertIn('Godot', query['gsrsearch'][0])
            self.assertNotIn('Shared Foundation', url)
            self.assertNotIn('non-public', url.casefold())
            self.assertEqual(query['gsrnamespace'], ['6'])
            self.assertLessEqual(timeout, 8)
            self.assertEqual(redirects, 0)
            self.assertIn('info@creata.news', user_agent)
            return commons_response(file_page('Godot 3.5 screenshot.png', author='<a href="/wiki/User:Alice">Alice &amp; Bob</a>'))

        with patch('hermes.net.fetch', side_effect=fetch):
            run_job(api, job)

        self.assertEqual(api.posts['complete']['result'], {
            'score': 35, 'entities': ENTITIES,
            'photos': [{
                'caption': 'Godot 3.5 screenshot', 'credit': 'Alice & Bob', 'rights': 'CC BY-SA 4.0',
                'url': 'https://thumb.wikimedia.org/wikipedia/commons/thumb/9/91/Godot_3.5_screenshot.png/960px-Godot_3.5_screenshot.png',
                'sourceUrl': 'https://commons.wikimedia.org/wiki/File:Godot_3.5_screenshot.png',
                'licenseUrl': 'https://creativecommons.org/licenses/by-sa/4.0/',
            }],
        })

    def test_only_initial_draft_without_a_photo_decision_discovers_images(self):
        evidence = [{'url': STORY['url'], 'text': STORY['text']}]
        checkpoints = {
            'research': {'status': 'succeeded', 'output': {'retrievedEvidence': evidence, 'conflicts': [], 'singleSourceClaims': []}},
            'draft': {'status': 'succeeded', 'output': {'title': {'en': 'Godot launches'}, 'photos': ['Provider-supplied image']}},
            'review': {'status': 'succeeded', 'output': {'reviewNotes': []}},
        }
        for kind, article, story, should_find in (
            ('draft', {}, STORY, True),
            ('draft', {'photos': []}, STORY, False),
            ('draft', {}, {**STORY, 'photos': [{'url': 'Existing photo'}]}, False),
            ('draft', {}, {**STORY, 'url': 'http://127.0.0.1:12225/article'}, False),
            ('rework', {}, STORY, False),
        ):
            with self.subTest(kind=kind, article=article, story=story):
                api = CompletionAPI()
                job = {'id': 'job-photo-draft', 'leaseToken': 'lease', 'kind': kind,
                       'payload': {'story': story, 'article': article}, 'checkpoints': checkpoints}
                with patch('hermes.net.fetch', return_value=commons_response(file_page('Godot editor.png'))) as fetch:
                    run_job(api, job)
                result = api.posts['complete']['result']
                self.assertEqual(result['title']['en'], 'Godot launches')
                if should_find:
                    self.assertEqual(result['photos'][0]['caption'], 'Godot editor')
                else:
                    self.assertNotIn('photos', result)
                    fetch.assert_not_called()

    def test_provider_failure_still_completes_classification(self):
        for response in (TimeoutError('Commons unavailable'), (b'{"query": [null]}', 'application/json', None)):
            with self.subTest(response=response):
                api = CompletionAPI()
                job = {'id': 'job-photo-failure', 'leaseToken': 'lease', 'kind': 'classify', 'payload': {'story': STORY},
                       'checkpoints': {'classification': {'status': 'succeeded', 'output': {'score': 80, 'entities': ENTITIES}}}}
                with patch('hermes.net.fetch', side_effect=[response, response]), self.assertLogs('hermes.photos', 'WARNING') as logs:
                    run_job(api, job)
                self.assertEqual(api.posts['complete']['result'], {'score': 80, 'entities': ENTITIES})
                self.assertNotIn('Godot', ' '.join(logs.output))


class CommonsPhotoTests(unittest.TestCase):
    def test_discovery_canonicalizes_commons_tracking_and_legacy_license_urls(self):
        from hermes.photos import find_article_photos

        page = file_page(
            'Godot editor.png',
            thumburl='https://thumb.wikimedia.org/wikipedia/commons/thumb/9/91/Godot_editor.png/960px-Godot_editor.png?utm_source=commons',
            descriptionurl='https://commons.wikimedia.org/wiki/File:Godot_editor.png?uselang=en',
            extmetadata={'Artist': {'value': 'Actual creator'},
                         'LicenseUrl': {'value': 'http://creativecommons.org/publicdomain/zero/1.0/deed.en'}},
        )
        with patch('hermes.net.fetch', return_value=commons_response(page)):
            photos = find_article_photos({**STORY, 'entities': ENTITIES})

        self.assertEqual(photos, [{
            'caption': 'Godot editor', 'credit': 'Actual creator', 'rights': 'CC0 1.0',
            'url': 'https://thumb.wikimedia.org/wikipedia/commons/thumb/9/91/Godot_editor.png/960px-Godot_editor.png',
            'sourceUrl': 'https://commons.wikimedia.org/wiki/File:Godot_editor.png',
            'licenseUrl': 'https://creativecommons.org/publicdomain/zero/1.0/',
        }])

    def test_discovery_uses_versioned_product_from_headline_without_entities(self):
        from hermes.photos import find_article_photos

        def fetch(url, **_):
            query = parse_qs(urlsplit(url).query)['gsrsearch'][0]
            self.assertIn('Godot', query)
            self.assertNotIn('snapshot', query)
            return commons_response(file_page('Godot4-Interface-FR.png'))

        with patch('hermes.net.fetch', side_effect=fetch):
            photos = find_article_photos({'title': 'Dev snapshot: Godot 4.8 beta 4'})

        self.assertEqual([photo['caption'] for photo in photos], ['Godot4-Interface-FR'])

    def test_discovery_uses_requested_commons_attribution_and_credit(self):
        from hermes.photos import find_article_photos

        custom = file_page('Godot editor.png', 1)
        custom['imageinfo'][0]['extmetadata']['Attribution'] = {'value': '<b>Godot Community Contributors</b>'}
        custom['imageinfo'][0]['extmetadata']['Credit'] = {'value': 'A source the custom credit replaces'}
        with_source = file_page('Godot interface.png', 2)
        with_source['imageinfo'][0]['extmetadata']['Credit'] = {'value': '<a href="https://example.org/">Studio Archive</a>'}
        own_work = file_page('Godot debugger.png', 3)
        own_work['imageinfo'][0]['extmetadata']['Credit'] = {'value': '<span>Own work</span>'}

        def fetch(url, **_):
            fields = parse_qs(urlsplit(url).query)['iiextmetadatafilter'][0].split('|')
            self.assertIn('Attribution', fields)
            self.assertIn('Credit', fields)
            return commons_response(custom, with_source, own_work)

        with patch('hermes.net.fetch', side_effect=fetch):
            photos = find_article_photos({**STORY, 'entities': ENTITIES})

        self.assertEqual([photo['credit'] for photo in photos], [
            'Godot Community Contributors', 'A. Creator / Studio Archive', 'A. Creator',
        ])

    def test_software_release_prefers_newer_screenshots_and_excludes_unrelated_theater(self):
        from hermes.photos import find_article_photos

        pages = (
            file_page('Waiting for Godot, theater production.png', 1),
            file_page('Godot splash screen.png', 2),
            file_page('Godot 3.1 screenshot.png', 3),
            file_page('Godot 4.6.png', 4),
            file_page('Godot4 editor interface.png', 5),
        )

        def fetch(url, **_):
            terms = parse_qs(urlsplit(url).query)['gsrsearch'][0]
            self.assertTrue('screenshot' in terms or 'engine' in terms, terms)
            return commons_response(*pages)

        with patch('hermes.net.fetch', side_effect=fetch):
            photos = find_article_photos({'title': 'Dev snapshot: Godot 4.8 beta 4'})

        self.assertEqual([photo['caption'] for photo in photos], [
            'Godot 4.6', 'Godot4 editor interface', 'Godot 3.1 screenshot',
        ])

    def test_discovery_filters_relevance_dimensions_license_credit_and_duplicates(self):
        from hermes.photos import find_article_photos

        pages = [
            None,
            file_page('Godot icon.png', 1),
            file_page('Generic photo.png', 2),
            file_page('Godot tiny.png', 3, thumbheight=60),
            file_page('Godot vector.svg', 4, mime='image/svg+xml'),
            file_page('Godot uncredited.png', 5, author='Unknown author'),
            file_page('Godot outside.png', 6, thumburl='https://upload.wikimedia.org.example.com/godot.png'),
            file_page('Godot wrong source.png', 7, descriptionurl='https://commons.wikimedia.org/wiki/User:A'),
            file_page('Godot wrong license.png', 8, extmetadata={
                'Artist': {'value': 'Actual Artist'}, 'LicenseUrl': {'value': 'https://creativecommons.org/licenses/by-nc/4.0/'},
            }),
            file_page('Godot editor 3.5.png', 9),
            file_page('Godot editor in 3.5.png', 10, sha1='9'),
            file_page('Godot Inspector.png', 11, extmetadata={
                'Artist': {'value': '<b>Inspector Author</b>'},
                'LicenseUrl': {'value': 'https://creativecommons.org/publicdomain/zero/1.0'},
                'ImageDescription': {'value': '<p>Godot Inspector in the editor.</p>'},
            }),
            file_page('Godot debugger.png', 12),
            file_page('Godot game window.png', 13),
        ]
        with patch('hermes.net.fetch', return_value=commons_response(*reversed(pages))):
            photos = find_article_photos({**STORY, 'entities': ENTITIES})

        self.assertEqual([photo['caption'] for photo in photos], ['Godot editor 3.5', 'Godot Inspector in the editor.', 'Godot debugger'])
        self.assertEqual(photos[1]['credit'], 'Inspector Author')
        self.assertEqual(photos[1]['rights'], 'CC0 1.0')
        self.assertEqual(photos[1]['licenseUrl'], 'https://creativecommons.org/publicdomain/zero/1.0/')
