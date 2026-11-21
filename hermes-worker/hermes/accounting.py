"""Durable paid-response outbox; accounting retries never require another AI call."""
import hashlib
import json
import os
import tempfile
import time
from pathlib import Path

def retry_transport(operation,attempts=3):
    for attempt in range(attempts):
        try:return operation()
        except Exception as error:
            status=getattr(error,'status',getattr(error,'code',None))
            retryable=status in (429,502,503,504) if status is not None else isinstance(error,(OSError,TimeoutError))
            if not retryable or attempt==attempts-1:raise
            time.sleep(0.05*(attempt+1))

def state_dir():return Path(os.getenv('CREATA_WORKER_STATE_DIR',str(Path.cwd()/'.hermes-state')))
def record_path(job_id,phase):return state_dir()/(hashlib.sha256((job_id+':'+phase).encode()).hexdigest()+'.json')
def load_record(job_id,phase):
    path=record_path(job_id,phase)
    try:return json.loads(path.read_text(encoding='utf-8'))
    except FileNotFoundError:return None

def save_record(record):
    directory=state_dir();directory.mkdir(parents=True,exist_ok=True)
    fd,temporary=tempfile.mkstemp(prefix='pending-',suffix='.tmp',dir=directory)
    try:
        with os.fdopen(fd,'w',encoding='utf-8') as file:
            json.dump(record,file,ensure_ascii=False);file.flush();os.fsync(file.fileno())
        os.replace(temporary,record_path(record['jobId'],record['phase']))
    finally:
        try:os.unlink(temporary)
        except FileNotFoundError:pass

def clear_record(job_id,phase):
    try:record_path(job_id,phase).unlink()
    except FileNotFoundError:pass

def commit_usage(api,usage):
    seen=set()
    for item in usage:
        if item['reservationId'] in seen:continue
        retry_transport(lambda:api.post('/internal/ai/commit',item));seen.add(item['reservationId'])

def reconcile_phase(api,job_id,phase,usage=(),completed=False):
    record=load_record(job_id,phase) if job_id else None
    combined=list(usage)
    if record and not record.get('accounted'):combined.append(record['usage'])
    commit_usage(api,combined)
    if record:
        record['accounted']=True
        if completed or record.get('rejected'):clear_record(job_id,phase)
        else:save_record(record)

def reconcile_outbox(api):
    directory=state_dir()
    if not directory.exists():return
    for path in sorted(directory.glob('*.json')):
        record=json.loads(path.read_text(encoding='utf-8'))
        reconcile_phase(api,record['jobId'],record['phase'],completed=record.get('checkpointed',False))
