"""Find relevant Commons photos with credited, reusable source files."""

import ipaddress
import json
import logging
import re
from http.client import HTTPException
from urllib.parse import unquote, urlencode, urlsplit

from . import net
from .extract import ArticleParser


COMMONS_API = 'https://commons.wikimedia.org/w/api.php'
COMMONS_USER_AGENT = 'CreataHermes/1.0 (https://www.creata.news/; info@creata.news)'
LOGGER = logging.getLogger(__name__)
IMAGE_HOSTS = ('upload.wikimedia.org', 'thumb.wikimedia.org')
IMAGE_MIMES = ('image/jpeg', 'image/png', 'image/webp')
LICENSES = {
    '/licenses/by/3.0': 'CC BY 3.0',
    '/licenses/by/4.0': 'CC BY 4.0',
    '/licenses/by-sa/3.0': 'CC BY-SA 3.0',
    '/licenses/by-sa/4.0': 'CC BY-SA 4.0',
    '/publicdomain/zero/1.0': 'CC0 1.0',
}
ENTITY_PRIORITY = ('product', 'person', 'place', 'event', 'organization')
WORDS = re.compile(r'[^\W\d_]+|\d+')
VERSION = re.compile(r'(?<!\d)\d+(?:\.\d+)+(?!\d)')
VERSION_SUBJECT = re.compile(r'([^\W\d_][\w.-]*(?:[ \t]+[^\W\d_][\w.-]*){0,2})[ \t]+v?\d+(?:\.\d+)+\b')
LICENSE_DEED = re.compile(r'/(?:deed|legalcode)(?:\.[a-z-]+)?$')
LEADING_WORDS = {'a', 'an', 'the', 'new', 'how', 'why', 'what', 'in', 'on', 'at', 'for', 'from', 'with',
                 'announcing', 'introducing', 'release', 'released', 'dev', 'snapshot', 'version', 'beta', 'update',
                 'как', 'почему', 'новый', 'новая', 'новое', 'в', 'на', 'для', 'жаңа'}
GENERIC_IMAGES = {'emblem', 'flag', 'icon', 'logo', 'placeholder', 'splash', 'splashscreen', 'symbol'}
UNKNOWN_ARTISTS = {'unknown', 'unknown author', 'author unknown', 'anonymous', 'not provided', 'own work', 'n/a', '-'}
SOFTWARE_IMAGE_WORDS = {'application', 'debugger', 'editor', 'engine', 'interface', 'screenshot', 'software', 'visualscript'}
SOFTWARE_RELEASE_WORDS = SOFTWARE_IMAGE_WORDS | {'alpha', 'beta', 'patch', 'rc', 'release', 'snapshot', 'version'}


def _subject(story):
    title = story.get('title')
    if not isinstance(title, str):
        return ''
    title = title[:500]
    title_words = WORDS.findall(title)
    normalized_title = ' ' + ' '.join(word.casefold() for word in title_words) + ' '
    entities = story.get('entities')
    if isinstance(entities, list):
        for kind in ENTITY_PRIORITY:
            for entity in entities[:50]:
                if not isinstance(entity, dict) or entity.get('type') != kind:
                    continue
                name = entity.get('name')
                words = WORDS.findall(name[:120]) if isinstance(name, str) else []
                phrase = ' '.join(words)
                if words and 2 < len(phrase) <= 100 and f' {phrase.casefold()} ' in normalized_title:
                    return phrase

    versioned = VERSION_SUBJECT.search(title)
    if versioned:
        name = []
        for word in reversed(versioned[1].split()):
            if not any(char.isupper() for char in word) or word.casefold() in LEADING_WORDS:
                break
            name.append(word)
        if name:
            return ' '.join(WORDS.findall(' '.join(reversed(name))))[:100]

    meaningful = [word for word in title_words if word[:1].isalpha() and word.casefold() not in LEADING_WORDS]
    if not meaningful:
        return ''
    if meaningful[0][:1].isupper() and meaningful[0] == title_words[0]:
        leading_name = []
        for word in title_words[:4]:
            if not word[:1].isupper() or word.casefold() in LEADING_WORDS:
                break
            leading_name.append(word)
        if leading_name and len(leading_name[0]) >= 3:
            return ' '.join(leading_name)[:100]
    return ' '.join(meaningful[:3])[:100]


def _public_story(story):
    # Local sources can contain private editorial material and fixture headlines.
    url = story.get('url')
    if url is None:
        return True
    if not isinstance(url, str):
        return False
    try:
        parsed = urlsplit(url)
    except ValueError:
        return False
    host = (parsed.hostname or '').rstrip('.')
    if parsed.scheme not in ('https', 'http') or not host:
        return False
    try:
        return ipaddress.ip_address(host).is_global
    except ValueError:
        return '.' in host and not host.endswith(('.localhost', '.local', '.internal', '.test'))


def _commons_pages(query):
    params = urlencode({
        'action': 'query', 'generator': 'search', 'gsrnamespace': 6,
        'gsrsearch': query, 'gsrlimit': 18, 'prop': 'imageinfo',
        'iiprop': 'url|size|mime|sha1|extmetadata', 'iiurlwidth': 960,
        'iiextmetadatafilter': 'Artist|Attribution|Credit|ImageDescription|LicenseUrl',
        'format': 'json', 'formatversion': 2,
    })
    try:
        raw, _, _ = net.fetch(f'{COMMONS_API}?{params}', max_bytes=1_000_000, redirects=0, timeout=5,
                              user_agent=COMMONS_USER_AGENT)
        data = json.loads(raw)
    except (ValueError, OSError, RuntimeError, HTTPException) as err:
        LOGGER.warning('Wikimedia Commons photo search failed (%s)', type(err).__name__)
        return None
    result = data.get('query', {}) if isinstance(data, dict) and 'error' not in data else None
    pages = result.get('pages', []) if isinstance(result, dict) else None
    if not isinstance(pages, list):
        LOGGER.warning('Wikimedia Commons photo search returned an invalid response')
        return None
    return sorted((page for page in pages if isinstance(page, dict)),
                  key=lambda page: page['index'] if type(page.get('index')) is int else 1000)


def _provider_url(value, hosts, schemes=('https',)):
    if not isinstance(value, str) or not value or '\\' in value or any(char.isspace() or ord(char) < 32 for char in value):
        return None
    try:
        if len(value.encode()) > 2048:
            return None
        url = urlsplit(value)
    except ValueError:
        return None
    if url.scheme in schemes and url.netloc.lower() in hosts:
        return url._replace(scheme='https', netloc=url.netloc.lower(), query='', fragment='')
    return None


def _metadata_text(metadata, key):
    entry = metadata.get(key)
    value = entry.get('value') if isinstance(entry, dict) else None
    if not isinstance(value, str):
        return ''
    parser = ArticleParser()
    try:
        parser.feed(value[:8000])
        parser.close()
        text = ' '.join(''.join(parser.parts).split())
        text.encode()
        return text
    except (ValueError, AssertionError):
        return ''


def _relevant(subject, value):
    # Match whole words, so Godot never matches Godotian.
    return ' ' + subject.casefold() + ' ' in ' ' + ' '.join(WORDS.findall(value.casefold())) + ' '


def _photo_credit(metadata):
    custom = _metadata_text(metadata, 'Attribution')
    if custom and custom.casefold() not in UNKNOWN_ARTISTS:
        return custom
    artist = _metadata_text(metadata, 'Artist')
    if not artist or artist.casefold() in UNKNOWN_ARTISTS:
        return ''
    source = _metadata_text(metadata, 'Credit')
    if source and source.casefold() not in UNKNOWN_ARTISTS and source.casefold() not in artist.casefold():
        return f'{artist} / {source}'
    return artist


def _subject_version(subject, value):
    match = re.search(r'(?<!\w)' + re.escape(subject) + r'[ -]*v?(\d+(?:\.\d+)*)(?!\w)', value, re.IGNORECASE)
    return tuple(map(int, match[1].split('.'))) if match else ()


def _photo_from_page(page, subject, software_release):
    title = page.get('title')
    infos = page.get('imageinfo')
    if not isinstance(title, str) or len(title) > 500 or not title.startswith('File:') or not isinstance(infos, list) or not infos:
        return None
    info = infos[0]
    if not isinstance(info, dict) or info.get('mime') not in IMAGE_MIMES:
        return None
    if any(type(info.get(key)) is not int or info[key] < minimum
           for key, minimum in (('width', 640), ('height', 360), ('thumbwidth', 640), ('thumbheight', 360))):
        return None
    if max(info['thumbwidth'], info['thumbheight']) > 3 * min(info['thumbwidth'], info['thumbheight']):
        return None

    url = _provider_url(info.get('thumburl'), IMAGE_HOSTS)
    source = _provider_url(info.get('descriptionurl'), ('commons.wikimedia.org',))
    source_path = unquote(source.path) if source else ''
    if not url or not url.path.startswith('/wikipedia/commons/') or not source_path.startswith('/wiki/File:') or source_path == '/wiki/File:':
        return None
    metadata = info.get('extmetadata')
    if not isinstance(metadata, dict):
        return None
    credit = _photo_credit(metadata)
    if not credit or len(credit) > 400:
        return None
    license_url = _provider_url(_metadata_text(metadata, 'LicenseUrl'), ('creativecommons.org',), ('http', 'https'))
    license_path = LICENSE_DEED.sub('', license_url.path.rstrip('/')) if license_url else ''
    if license_path not in LICENSES:
        return None

    filename = ' '.join(title[5:].rsplit('.', 1)[0].replace('_', ' ').split())
    if not filename or GENERIC_IMAGES.intersection(WORDS.findall(filename.casefold())):
        return None
    description = _metadata_text(metadata, 'ImageDescription')
    relevant_filename = _relevant(subject, filename)
    description_excerpt = description[:600]
    relevant_description = _relevant(subject, description_excerpt)
    if not relevant_filename and not relevant_description:
        return None
    if software_release and not (
        relevant_filename and SOFTWARE_IMAGE_WORDS.intersection(WORDS.findall(filename.casefold()))
        or relevant_description and SOFTWARE_IMAGE_WORDS.intersection(WORDS.findall(description_excerpt.casefold()))
        or re.fullmatch(re.escape(subject) + r'[ -]*v?\d+(?:\.\d+)+', filename, re.IGNORECASE)
    ):
        return None
    versions = set(VERSION.findall(filename))
    file_version = _subject_version(subject, filename)
    description_version = _subject_version(subject, description_excerpt)
    use_description = (relevant_description and len(description) <= 200 and versions.issubset(VERSION.findall(description))
                       and file_version == description_version[:len(file_version)])
    caption = description if use_description else filename.encode(errors='replace').decode()
    return {
        'url': url.geturl(), 'caption': caption[:300], 'credit': credit,
        'rights': LICENSES[license_path], 'sourceUrl': source.geturl(),
        'licenseUrl': f'https://creativecommons.org{license_path}/',
    }


def find_article_photos(story):
    """Return up to three attributed images for a collected story."""
    if not _public_story(story):
        return []
    subject = _subject(story)
    if len(subject) < 3:
        return []
    title = story['title'][:500]
    release = VERSION.search(title)
    software_release = bool(release and SOFTWARE_RELEASE_WORDS.intersection(WORDS.findall(title.casefold())))
    if software_release:
        major = int(release[0].split('.')[0])
        queries = (f'"{subject}" screenshot', f'intitle:"{subject}" engine')
    else:
        queries = (f'intitle:"{subject}"', f'"{subject}"')
    photos = []
    seen = set()
    for query in queries:
        pages = _commons_pages(query)
        if pages is None:
            break
        for page in pages:
            photo = _photo_from_page(page, subject, software_release)
            if photo is None:
                continue
            sha1 = page['imageinfo'][0].get('sha1')
            keys = {photo['url'], unquote(photo['sourceUrl']).replace('_', ' ').casefold()}
            if isinstance(sha1, str) and sha1:
                keys.add(sha1)
            if seen.isdisjoint(keys):
                seen.update(keys)
                photos.append(photo)
        if len(photos) >= 3:
            break
    if software_release:
        def version_rank(photo):
            version = _subject_version(subject, photo['caption'])
            return version[:1] == (major,), version
        photos.sort(key=version_rank, reverse=True)
    return photos[:3]


def enrich_editorial_result(job, result):
    """Attach sourced images without overwriting an editor's prior decision."""
    kind = job.get('kind')
    if kind not in ('classify', 'draft', 'rework'):
        return result
    result = dict(result)
    result.pop('photos', None)
    payload = job.get('payload') or {}
    story = payload.get('story') or {}
    article = payload.get('article') or {}
    if kind == 'classify' and not (story.get('articleId') or story.get('graphicId') or story.get('mergedInto')):
        photos = find_article_photos({**story, 'entities': result.get('entities')})
    elif kind == 'draft' and article.get('format') != 'graphic' and 'photos' not in article and 'photos' not in story:
        photos = find_article_photos(story)
    else:
        return result
    if photos:
        result['photos'] = photos
    return result
