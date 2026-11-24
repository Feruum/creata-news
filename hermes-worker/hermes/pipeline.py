from .ai import AIProvider, validate_draft
from .collectors import collect, page
from .classification import validate_classification
from .accounting import retry_transport, reconcile_phase, load_record, save_record
from .exa import discover_evidence
from .extract import normalize_url
import re

MAX_EVIDENCE_ENTRIES = 10
MAX_EVIDENCE_CHARACTERS = 120000
MAX_SOURCE_CHARACTERS = 20000

def validate_research(brief,evidence,unavailable):
    if not isinstance(brief,dict) or not isinstance(brief.get('facts'),list):raise ValueError('Research facts must be an array')
    texts={e['url']:e['text'] for e in evidence}
    single=[]
    for fact in brief['facts']:
        if not isinstance(fact,dict) or not isinstance(fact.get('claim'),str) or not fact['claim'].strip():raise ValueError('Research claim is required')
        urls=fact.get('sourceUrls')
        if not isinstance(urls,list) or not urls or any(not isinstance(u,str) or u not in texts for u in urls):raise ValueError('Research claim cites unavailable evidence')
        if len(set(urls))<2:single.append(fact['claim'])
        supported=set()
        for field in ('numbers','dates'):
            refs=fact.get(field,[])
            if not isinstance(refs,list):raise ValueError('Research numeric and date evidence must be an array')
            for ref in refs:
                if not isinstance(ref,dict):raise ValueError('Invalid research source quote')
                value,quote,url=ref.get('value'),ref.get('quote'),ref.get('sourceUrl')
                if not isinstance(value,str) or not value or not isinstance(quote,str) or not quote or url not in urls or quote not in texts.get(url,'') or value not in quote:raise ValueError('Research number or date is not supported by source text')
                supported.update(re.findall(r'\d+(?:[.,]\d+)*',value))
        if any(v not in supported for v in re.findall(r'\d+(?:[.,]\d+)*',fact['claim'])):raise ValueError('Every research number and date requires a source quote')
    for field in ('conflicts','singleSourceClaims'):
        if not isinstance(brief.get(field),list) or any(not isinstance(v,str) for v in brief[field]):raise ValueError('Research warnings must be text arrays')
    brief['singleSourceClaims']=list(dict.fromkeys(brief['singleSourceClaims']+single))
    brief['retrievedEvidence']=evidence
    brief['unavailableSources']=unavailable
    return brief

def validate_opportunities(draft,opportunities):
    selected=draft.setdefault('opportunityIds',[])
    allowed={o['id'] for o in opportunities}
    if not isinstance(selected,list) or any(not isinstance(oid,str) or oid not in allowed for oid in selected) or len(set(selected))!=len(selected):raise ValueError('Draft selected unknown opportunity identifiers')
    return draft

def _source_key(url):
    if isinstance(url, str):
        try:
            return normalize_url(url)
        except ValueError:
            return None
    return None

def _story_metadata(story, evidence):
    metadata = {field: story[field][:500]
                for field in ('title', 'locale', 'categoryId', 'region', 'publishedAt')
                if isinstance(story.get(field), str)}
    original_url = _source_key(story.get('url'))
    if original_url and any(_source_key(entry['url']) == original_url for entry in evidence):
        metadata['url'] = original_url[:2048]
    return metadata

def source_evidence(story,article,sources,*,current_sources_only=False):
    current = {_source_key(source.get('url')) for source in sources if isinstance(source, dict)} - {None} if current_sources_only else None

    def selected(url):
        return current is None or _source_key(url) in current

    evidence=[e for e in story.get('evidence',[]) if isinstance(e,dict) and e.get('url') and str(e.get('text','')).strip() and selected(e['url'])]
    if story.get('text') and selected(story.get('url')) and story.get('url') and not any(e['url']==story['url'] for e in evidence):
        evidence.append({'url':story['url'],'title':story.get('title',''),'text':story['text']})
    available={e['url'] for e in evidence}
    unavailable=[]
    for source in sources[:8]:
        if source['url'] in available:continue
        try:
            retrieved=page(source['url'])
            if not retrieved.get('text','').strip():raise ValueError('No source text')
            evidence.append({k:retrieved[k] for k in ('url','title','text','author','publishedAt') if k in retrieved})
            available.add(source['url'])
        except (ValueError,RuntimeError,OSError):unavailable.append(source['url'])
    return bound_evidence(evidence),unavailable

def bound_evidence(evidence):
    # Keep persisted research within the API body limit and the provider context.
    # Immutable raw items retain the complete collected text for editorial review.
    bounded=[];remaining=MAX_EVIDENCE_CHARACTERS
    for item in evidence[:MAX_EVIDENCE_ENTRIES]:
        if remaining<=0:break
        text=item['text'][:min(MAX_SOURCE_CHARACTERS,remaining)];remaining-=len(text)
        bounded.append({**item,'text':text})
    return bounded

class Pipeline:
    def __init__(self,api):self.api=api;self.provider=None;self.current_phase=None
    def stage(self,job,phase,operation):
        existing=(job.get('checkpoints') or {}).get(phase) or {}
        job_id=job.get('id')
        reconcile_phase(self.api,job_id,phase,existing.get('usage',[]),completed=existing.get('status')=='succeeded')
        if existing.get('status')=='succeeded':return existing['output']
        persistent=bool(job.get('id') and job.get('leaseToken'))
        def save(status,**extra):
            if persistent:retry_transport(lambda:self.api.post('/internal/jobs/'+job['id']+'/checkpoint',{'leaseToken':job['leaseToken'],'phase':phase,'status':status,**extra}))
        def usage():
            values=list(self.provider.usage_for(phase)) if self.provider and hasattr(self.provider,'usage_for') else []
            pending=load_record(job_id,phase) if job_id else None
            if pending and not any(item['reservationId']==pending['usage']['reservationId'] for item in values):values.append(pending['usage'])
            return values
        self.current_phase=phase
        save('running')
        try:
            value=operation()
        except Exception as err:
            record=load_record(job_id,phase) if job_id else None
            if record and isinstance(err,ValueError):record['rejected']=True;save_record(record)
            try:save('failed',error=str(err)[:1000],usage=usage())
            finally:reconcile_phase(self.api,job_id,phase,usage())
            raise
        paid_usage=usage()
        try:save('succeeded',output=value,usage=paid_usage)
        except Exception:
            # Retain the paid output in the outbox; reconcile even if a lease was lost.
            reconcile_phase(self.api,job_id,phase,paid_usage)
            raise
        job.setdefault('checkpoints',{})[phase]={'status':'succeeded','output':value,'usage':paid_usage}
        record=load_record(job_id,phase) if job_id else None
        if record:record['checkpointed']=True;save_record(record)
        reconcile_phase(self.api,job_id,phase,paid_usage,completed=True)
        return value
    def process(self,job):
        kind=job['kind'];payload=job.get('payload') or {}
        if kind in ('collect','check'):
            return collect(payload.get('source') or payload,kind=='check')
        provider=None
        def get_provider():
            nonlocal provider
            if provider is None:
                provider=AIProvider.for_context(self.api,job,self.current_phase) if hasattr(AIProvider,'for_context') else AIProvider(self.api)
                self.provider=provider
            if hasattr(provider,'set_context'):provider.set_context(job,self.current_phase)
            return provider
        def complete(task,data,max_tokens):
            return get_provider().complete(task,data,max_tokens)
        story=payload.get('story') or {}
        if kind=='classify':
            def classify():
                result=complete('Classify the story for young creators. Return categoryId (choose supplied category), region:string, tags:string[], score:number (0-100), reason:string, entities:[{name:string,type:person|organization|place|product|event|other,sourceUrls:string[]}], facts:[{claim:string,sourceUrls:string[],uncertainty:string}]. Both entities and facts arrays are mandatory; use [] if no supported extraction exists. Every fact claim must be a verbatim excerpt (at most 1000 characters) occurring in each cited supplied source text, not a paraphrase. Every complete entity name must occur in each cited source text. A URL without source text is not evidence. Never invent sources, entities, numbers or dates.',{'story':story,'categories':get_provider().config.get('categories',[])},3500)
                return validate_classification(result,story,categories=get_provider().config.get('categories',[]))
            return self.stage(job,'classification',classify)
        if kind not in ('draft','rework'):raise ValueError('Unsupported job kind')
        article=payload.get('article') or {}
        use_current_sources = kind == 'rework' and 'sources' in article
        if use_current_sources:
            sources = article['sources'] or []
        else:
            sources = story.get('sources') or article.get('sources') or []
            if not sources and story.get('url'):
                sources = [{'url':story['url'],'title':story.get('title','')}]
        cached=(job.get('checkpoints') or {}).get('research') or {}
        pending=load_record(job['id'],'research') if job.get('id') else None
        recovering = pending and not pending.get('rejected') and pending.get('output') is not None
        if cached.get('status')=='succeeded':
            evidence=cached['output']['retrievedEvidence'];unavailable=cached['output'].get('unavailableSources',[])
        elif recovering and pending.get('inputData',{}).get('retrievedEvidence'):
            evidence=pending['inputData']['retrievedEvidence'];unavailable=pending['inputData'].get('unavailableSources',[])
        else:evidence,unavailable=source_evidence(story,article,sources,current_sources_only=use_current_sources)
        if not evidence:raise ValueError('No retrievable source evidence; editor must add a readable source')
        if article.get('format')=='graphic':
            from .graphics import draft_graphic
            pipeline=self
            class CheckpointProvider:
                phase=0
                def complete(self,task,data,max_tokens):
                    phase=('research','draft','review')[self.phase];self.phase+=1
                    def invoke():
                        result=complete(task,data,max_tokens)
                        if phase=='research':
                            from .graphics import validate_graphic_result, graphic_research_brief
                            if result.get('error'):raise ValueError('Insufficient comparable source numbers')
                            validate_graphic_result(result,evidence)
                            return validate_research(graphic_research_brief(result,evidence,unavailable),[dict(e,text=' '.join(e['text'].split())) for e in evidence],unavailable)
                        if phase=='draft':validate_opportunities(validate_draft(result),payload.get('opportunities',[]))
                        if phase=='review' and (not isinstance(result.get('reviewNotes'),list) or any(not isinstance(n,str) for n in result['reviewNotes'])):raise ValueError('Invalid graphic review output')
                        return result
                    output=pipeline.stage(job,phase,invoke)
                    return output['providerResult'] if phase=='research' else output
            result=draft_graphic(CheckpointProvider(),story,article,sources,evidence,payload.get('note',''),opportunities=payload.get('opportunities',[]),unavailable_sources=unavailable)
            result['researchBrief']=job['checkpoints']['research']['output']
            return result
        story_metadata = _story_metadata(story, evidence)

        def research():
            retrieved = evidence
            if not recovering and sum(len(entry['text']) for entry in evidence) < MAX_EVIDENCE_CHARACTERS:
                limit = min(2, MAX_EVIDENCE_ENTRIES - len(evidence))
                additional = discover_evidence(get_provider().config.get('exaApiKey'), story, evidence, sources, page, limit)
                if additional:
                    retrieved = bound_evidence(evidence + additional)
            data = {'retrievedEvidence':retrieved,'story':story_metadata,'unavailableSources':unavailable}
            result = complete('Research only retrieved source texts. A URL alone is not corroboration. Return facts (array of {claim,sourceUrls,numbers:[{value,sourceUrl,quote}],dates:[{value,sourceUrl,quote}]}), conflicts (array of strings), singleSourceClaims (array of strings). Every number and date in a claim needs an exact literal source quote containing its value. Keep article chronology and distinguish publication date from event date. Cite discovered sources only when their text supports the same fact about this story.',data,3500)
            return validate_research(result,retrieved,unavailable)
        brief = self.stage(job,'research',research)
        evidence = brief['retrievedEvidence']
        draft=self.stage(job,'draft',lambda:validate_opportunities(validate_draft(complete('Write an original Creata News article, factual and actionable. Return title (at most 8 words), lead, body (plain paragraphs), summary, actions (one practical action per newline). EACH field is {ru:string,kk:string,en:string}; all three language versions mandatory. Return opportunityIds: an array of exact supplied opportunity IDs used in actions, or []. Link actions only to supplied opportunities. No HTML. Respect editorial rework note; preserve corrections. Do not copy source text.',{'story':story_metadata,'evidence':brief,'previousDraft':article,'editorNote':payload.get('note',''),'opportunities':payload.get('opportunities',[])},11000)),payload.get('opportunities',[])))
        def review_draft():
            review=complete('Review the draft against retrieved source texts, comparing numbers, dates, names and translations. A URL alone is not evidence. Return reviewNotes (array of concise strings) identifying contradictions and single-source claims. Return no claims of external verification.',{'draft':draft,'brief':brief,'retrievedEvidence':evidence,'unavailableSources':unavailable},3000)
            if not isinstance(review.get('reviewNotes'),list) or any(not isinstance(n,str) for n in review['reviewNotes']):raise ValueError('Invalid review result')
            return review
        review=self.stage(job,'review',review_draft)
        notes=list(review.get('reviewNotes'))
        if not isinstance(notes,list) or any(not isinstance(n,str) for n in notes):raise ValueError('Invalid review result')
        if len({e['url'] for e in evidence})<2:notes.insert(0,'Single source: editorial verification required / Один источник / Бір дереккөз')
        if unavailable:notes.append('Source text unavailable; not corroborated: '+', '.join(unavailable))
        notes.extend('Conflict: '+v for v in brief.get('conflicts',[]))
        notes.extend('Single-source claim: '+v for v in brief.get('singleSourceClaims',[]))
        draft=dict(draft)
        draft['researchBrief']=brief
        draft['sources']=sources
        draft['reviewNotes']=notes
        return draft
