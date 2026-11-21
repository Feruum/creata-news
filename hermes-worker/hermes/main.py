import argparse
import json
import os
import signal
import threading
import time
import uuid
from .api import API, APIError
from .ai import NotConfigured
from .pipeline import Pipeline
from .accounting import reconcile_outbox

stop=threading.Event()

def run_job(api,job):
    lease=job['leaseToken'];done=threading.Event();lost=threading.Event()
    def heartbeat():
        while not done.wait(20):
            try:api.post('/internal/jobs/'+job['id']+'/heartbeat',{'leaseToken':lease})
            except Exception:lost.set();return
    thread=threading.Thread(target=heartbeat,daemon=True);thread.start()
    try:
        result=Pipeline(api).process(job)
        if lost.is_set():raise RuntimeError('Job lease was lost; result not committed')
        api.post('/internal/jobs/'+job['id']+'/complete',{'leaseToken':lease,'result':result})
        print(json.dumps({'job':job['id'],'status':'succeeded'}),flush=True)
    except Exception as err:
        configured=isinstance(err,NotConfigured)
        retryable=isinstance(err,(OSError,TimeoutError)) or (isinstance(err,APIError) and err.status in (429,502,503,504))
        try:api.post('/internal/jobs/'+job['id']+'/fail',{'leaseToken':lease,'error':str(err)[:1000],'retryable':retryable,'notConfigured':configured})
        except Exception:pass
        print(json.dumps({'job':job['id'],'status':'not_configured' if configured else 'failed','error':str(err)[:500]}),flush=True)
    finally:done.set();thread.join(timeout=1)

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--once',action='store_true');args=parser.parse_args()
    token=os.getenv('CREATA_WORKER_TOKEN','')
    if len(token)<32:raise SystemExit('CREATA_WORKER_TOKEN must contain at least 32 characters')
    api=API(os.getenv('CREATA_API_ORIGIN','http://localhost:5555'),token)
    worker_id=os.getenv('CREATA_WORKER_ID','hermes-'+str(uuid.uuid4())[:8])
    for sig in (signal.SIGINT,signal.SIGTERM):signal.signal(sig,lambda *_:stop.set())
    while not stop.is_set():
        try:
            reconcile_outbox(api)
            job=api.post('/internal/jobs/claim',{'workerId':worker_id}).get('job')
            if job:run_job(api,job)
            elif not args.once:stop.wait(3)
        except Exception as err:
            print(json.dumps({'status':'api_unavailable','error':str(err)[:300]}),flush=True)
            if not args.once:stop.wait(5)
        if args.once:break

if __name__=='__main__':main()
