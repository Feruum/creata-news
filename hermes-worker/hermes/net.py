"""Bounded public HTTP fetching with DNS pinning and redirect validation."""
import http.client
import ipaddress
import os
import socket
import ssl
from urllib.parse import urlsplit, urljoin

class UnsafeURL(ValueError):
    pass

def public_addresses(url, *, strict_public=False):
    parsed = urlsplit(url)
    if parsed.scheme not in ('http', 'https') or not parsed.hostname or parsed.username is not None or parsed.password is not None:
        raise UnsafeURL('Only public HTTP(S) URLs without credentials are allowed')
    allow_private = not strict_public and os.getenv('CREATA_ALLOW_PRIVATE_SOURCES') == '1' and os.getenv('CREATA_ENV') != 'production'
    addresses = socket.getaddrinfo(parsed.hostname, parsed.port or (443 if parsed.scheme == 'https' else 80), type=socket.SOCK_STREAM)
    result = []
    for info in addresses:
        addr = info[4][0]
        ip = ipaddress.ip_address(addr)
        if isinstance(ip, ipaddress.IPv6Address) and ip.ipv4_mapped: ip = ip.ipv4_mapped
        if (not ip.is_global or ip.is_multicast) and not allow_private:
            raise UnsafeURL('Private, loopback and link-local network targets are blocked')
        if addr not in result: result.append(addr)
    if not result: raise UnsafeURL('Source hostname has no usable public address')
    return result

def fetch(url, max_bytes=5_000_000, redirects=5, *, timeout=20, user_agent='CreataHermes/1.0 (+news research)', strict_public=False):
    for hop in range(redirects + 1):
        parsed = urlsplit(url)
        addresses = public_addresses(url, strict_public=strict_public)
        port = parsed.port or (443 if parsed.scheme == 'https' else 80)
        # Connect to the validated address, never resolve the hostname a second time.
        conn = http.client.HTTPConnection(parsed.hostname, port, timeout=timeout)
        sock = socket.create_connection((addresses[0], port), timeout=timeout)
        if parsed.scheme == 'https': sock = ssl.create_default_context().wrap_socket(sock, server_hostname=parsed.hostname)
        conn.sock = sock
        try:
            path = (parsed.path or '/') + ('?' + parsed.query if parsed.query else '')
            conn.request('GET', path, headers={'User-Agent':user_agent, 'Accept':'text/html,application/rss+xml,application/atom+xml,application/xml;q=0.9,*/*;q=0.5','Accept-Encoding':'identity'})
            res = conn.getresponse()
            if res.status in (301,302,303,307,308):
                if hop == redirects: raise UnsafeURL('Too many redirects')
                target = res.getheader('Location')
                if not target: raise ValueError('Redirect without a target')
                url = urljoin(url,target)
                continue
            if res.status >= 400: raise RuntimeError('Source HTTP status ' + str(res.status))
            raw = res.read(max_bytes+1)
            if len(raw)>max_bytes: raise ValueError('Source response exceeds size limit')
            return raw, res.getheader('Content-Type',''), url
        finally: conn.close()
    raise UnsafeURL('Redirect limit exceeded')
