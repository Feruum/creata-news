"""Discover article sources through Exa and retrieve their text independently."""

import ipaddress
import json
import logging
import re
from http.client import HTTPException
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import Request, build_opener

from .ai import NoRedirect
from .extract import normalize_url
from .net import public_addresses


LOGGER = logging.getLogger(__name__)
SEARCH_URL = 'https://api.exa.ai/search'
RESULT_LIMIT = 5
RESPONSE_LIMIT = 256_000
PAGE_TIMEOUT = 5
COMMON_WORDS = set('and the how why what for from with this that new will about как почему что чтобы для при новый новая новое жаңа үшін туралы'.split())


def _search_urls(api_key, headline, excluded_domains):
    body = {'query': headline, 'type': 'auto', 'numResults': RESULT_LIMIT, 'excludeDomains': sorted(excluded_domains)}
    try:
        request = Request(SEARCH_URL, data=json.dumps(body).encode(), method='POST',
                          headers={'x-api-key': api_key, 'Content-Type': 'application/json', 'Accept': 'application/json'})
        with build_opener(NoRedirect()).open(request, timeout=8) as response:
            raw = response.read(RESPONSE_LIMIT + 1)
        if len(raw) > RESPONSE_LIMIT:
            raise ValueError('Exa response exceeds size limit')
        value = json.loads(raw)
        results = value.get('results') if isinstance(value, dict) else None
        if not isinstance(results, list):
            raise ValueError('Invalid Exa response')
    except (ValueError, OSError, RuntimeError, HTTPException) as error:
        if isinstance(error, HTTPError):
            error.close()
        LOGGER.warning('Exa search failed; continuing with collected sources')
        return []
    return [item.get('url') for item in results[:RESULT_LIMIT] if isinstance(item, dict)]


def _canonical_public_url(value):
    if not isinstance(value, str) or not value or '\\' in value or any(c.isspace() or ord(c) < 32 for c in value):
        return None
    try:
        if len(value.encode()) > 2048:
            return None
        parsed = urlsplit(value)
        if parsed.username is not None or parsed.password is not None or not parsed.hostname:
            return None
        host = parsed.hostname.rstrip('.')
        try:
            address = ipaddress.ip_address(host)
        except ValueError:
            if '.' not in host or host.endswith(('.localhost', '.local', '.internal', '.test')):
                return None
        else:
            if isinstance(address, ipaddress.IPv6Address) and address.ipv4_mapped:
                address = address.ipv4_mapped
            if not address.is_global or address.is_multicast:
                return None
        return normalize_url(value)
    except ValueError:
        return None


def _domain(url):
    return urlsplit(url).hostname.rstrip('.').removeprefix('www.')


def _resolves_publicly(url):
    try:
        public_addresses(url, strict_public=True)
        return True
    except (ValueError, OSError):
        return False


def _terms(text):
    words = re.findall(r'[^\W\d_]{3,}', text.casefold())
    return set(words) - COMMON_WORDS


def _read_public_page(crawl, url, **options):
    try:
        return crawl(url, strict_public=True, timeout=PAGE_TIMEOUT, **options)
    except (ValueError, OSError, RuntimeError, HTTPException, AssertionError):
        return None


def discover_evidence(api_key, story, evidence, sources, crawl, limit):
    if not isinstance(api_key, str) or not api_key.strip() or limit < 1:
        return []
    if (not isinstance(story.get('sourceId'), str) or not story['sourceId'].strip()
            or story.get('sourceType') not in ('rss', 'website', 'telegram')):
        return []
    title = story.get('title')
    original_url = _canonical_public_url(story.get('url'))
    if not isinstance(title, str) or not original_url:
        return []
    telegram_hosts = ('t.me', 'telegram.me')
    original = urlsplit(original_url)
    if original.query:
        return []
    if story['sourceType'] == 'telegram' or original.hostname in telegram_hosts:
        if original.hostname not in telegram_hosts or not re.fullmatch(r'/(?:s/)?[a-zA-Z0-9_]{3,32}/[0-9]+/?', original.path):
            return []
    collected_headline = ' '.join(title[:500].split())
    if not _terms(collected_headline):
        return []

    evidence_urls = {_canonical_public_url(entry['url']) for entry in evidence} - {None}
    if original_url not in evidence_urls or len({_domain(url) for url in evidence_urls}) >= 2:
        return []
    if not _resolves_publicly(original_url):
        return []
    # A collected feed can carry a private headline even if its link has public DNS.
    # Confirm the heading using a public request without a query, redirect or credentials.
    original_page = _read_public_page(crawl, original_url, redirects=0)
    headline = ' '.join(original_page.get('title', '').split()) if original_page else ''
    if '\x00' in headline or headline.casefold() != collected_headline.casefold():
        return []
    terms = _terms(headline)
    source_urls = {_canonical_public_url(source.get('url')) for source in sources if isinstance(source, dict)} - {None}
    known_urls = source_urls | evidence_urls
    known_domains = {_domain(url) for url in known_urls}
    excluded_domains = {_domain(original_url)}
    for url in sorted(known_urls):
        domain = _domain(url)
        if domain not in excluded_domains and _resolves_publicly(url):
            excluded_domains.add(domain)

    seen = set(known_urls)
    discovered = []
    for url in _search_urls(api_key.strip(), headline, excluded_domains):
        url = _canonical_public_url(url)
        if not url or url in seen or _domain(url) in known_domains:
            continue
        seen.add(url)
        retrieved = _read_public_page(crawl, url)
        if not retrieved:
            continue
        final_url = _canonical_public_url(retrieved.get('url'))
        if not final_url or (final_url != url and final_url in seen) or _domain(final_url) in known_domains:
            continue
        seen.add(final_url)
        text = retrieved.get('text', '').strip()
        title = retrieved.get('title', '').strip()
        if not text or not title or '\x00' in title or '\x00' in text:
            continue
        if len(terms & _terms(title + ' ' + text[:20000])) < min(2, len(terms)):
            continue
        entry = {'url': final_url, 'title': title, 'text': text, 'discoveredBy': 'exa'}
        if retrieved.get('publishedAt'):
            entry['publishedAt'] = retrieved['publishedAt']
        discovered.append(entry)
        known_domains.add(_domain(final_url))
        if len(discovered) == limit:
            break
    return discovered
