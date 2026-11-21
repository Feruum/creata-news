import json
import urllib.request
import urllib.error
from .ai import NoRedirect

class APIError(RuntimeError):
    def __init__(self,status,message):super().__init__(message);self.status=status

class API:
    def __init__(self,origin,token):
        self.base=origin.rstrip('/')+'/api/creata/v1';self.token=token
    def request(self,path,payload=None):
        data=None if payload is None else json.dumps(payload,ensure_ascii=False).encode()
        req=urllib.request.Request(self.base+path,data=data,headers={'Authorization':'Bearer '+self.token,'Content-Type':'application/json'})
        try:
            with urllib.request.build_opener(NoRedirect()).open(req,timeout=40) as res:
                raw=res.read(10_000_001)
                if len(raw)>10_000_000:raise ValueError('API response too large')
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as err:
            try:message=json.loads(err.read(5000)).get('error','API request failed')
            except (ValueError,AttributeError):message='API request failed'
            raise APIError(err.code,message) from None
    def get(self,path):return self.request(path)
    def post(self,path,payload):return self.request(path,payload)
