"""Source-grounded chart drafting. Every plotted value needs literal source evidence."""
import math
import re
from decimal import Decimal, InvalidOperation
from .ai import validate_draft, LOCALES

def localized(value):
    return isinstance(value,dict) and all(isinstance(value.get(l),str) and value[l].strip() for l in LOCALES)

def source_number(raw):
    if not isinstance(raw,str) or len(raw)>50:raise ValueError('Invalid source number')
    value=re.sub(r'[\s\u00a0\u202f]','',raw).replace('−','-')
    if ',' in value and '.' in value:
        # The last punctuation is decimal, preceding punctuation is grouping.
        value=value.replace(',','') if value.rfind('.')>value.rfind(',') else value.replace('.','').replace(',','.')
    elif ',' in value:
        value=value.replace(',','') if re.fullmatch(r'[+-]?\d{1,3}(,\d{3})+',value) else value.replace(',','.')
    try:return Decimal(value)
    except InvalidOperation:raise ValueError('Source number is not numeric')

def normalize_quote(value):return ' '.join(value.split())

def validate_graphic_result(result,evidence):
    g=result.get('graphic')
    if not isinstance(g,dict) or g.get('type') not in ('bar','line','area','donut'):raise ValueError('Invalid chart type')
    labels,series=g.get('labels'),g.get('series')
    if not isinstance(labels,list) or not 2<=len(labels)<=120 or not all(localized(v) for v in labels):raise ValueError('Invalid translated chart labels')
    if not isinstance(series,list) or not 1<=len(series)<=6:raise ValueError('Invalid chart series')
    texts={e['url']:normalize_quote(e.get('text','')) for e in evidence}
    if not isinstance(g.get('sourceUrl'),str) or g['sourceUrl'] not in texts:raise ValueError('Chart source must be retrieved evidence')
    if not all(isinstance(g.get(k),str) and g[k].strip() for k in ('unit','period','sourceName')) or not localized(g.get('methodology')):raise ValueError('Chart needs unit, period, source and methodology')
    proofs=result.get('evidence')
    if not isinstance(proofs,list):raise ValueError('Every chart value requires source evidence')
    by_cell={}
    for proof in proofs:
        if not isinstance(proof,dict):raise ValueError('Invalid chart evidence')
        row,col=proof.get('row'),proof.get('series')
        if type(row) is not int or type(col) is not int or not 0<=row<len(labels) or not 0<=col<len(series):raise ValueError('Invalid evidence cell')
        if (row,col) in by_cell:raise ValueError('Duplicate chart evidence cell')
        quote,raw,url=proof.get('quote'),proof.get('numberText'),proof.get('sourceUrl')
        if not isinstance(quote,str) or not 1<=len(quote)<=400 or not isinstance(url,str) or url not in texts or normalize_quote(quote) not in texts[url]:raise ValueError('Chart evidence quote not found in source text')
        if not isinstance(raw,str) or raw not in quote:raise ValueError('Number text not found in source quote')
        by_cell[row,col]=source_number(raw)
    for col,s in enumerate(series):
        if not isinstance(s,dict) or not localized(s.get('name')) or not isinstance(s.get('values'),list) or len(s['values'])!=len(labels):raise ValueError('Series length or translated name is invalid')
        for row,value in enumerate(s['values']):
            if isinstance(value,bool) or not isinstance(value,(int,float)) or abs(value)>1e15 or not math.isfinite(value):raise ValueError('Chart values must be finite numbers')
            if (row,col) not in by_cell or by_cell[row,col]!=Decimal(str(value)):raise ValueError('Chart value does not match its source number; calculations and invented values are forbidden')
    if g['type']=='donut' and (len(series)!=1 or any(v<0 for v in series[0]['values']) or sum(series[0]['values'])<=0):raise ValueError('Invalid donut values')
    g['verified']=False
    return g

def graphic_research_brief(extracted,evidence,unavailable):
    facts=[]
    period=extracted['graphic']['period']
    for proof in extracted['evidence']:
        quote=normalize_quote(proof['quote']);url=proof['sourceUrl']
        numbers=[{'value':proof['numberText'],'sourceUrl':url,'quote':quote}];dates=[]
        supported=set(re.findall(r'\d+(?:[.,]\d+)*',proof['numberText']))
        for value in re.findall(r'\d+(?:[.,]\d+)*',quote):
            if value in supported:continue
            ref={'value':value,'sourceUrl':url,'quote':quote}
            (dates if re.fullmatch(r'[12]\d{3}',value) and value in period else numbers).append(ref)
        facts.append({'claim':quote,'sourceUrls':[url],'numbers':numbers,'dates':dates})
    return {'facts':facts,'conflicts':[],'singleSourceClaims':[fact['claim'] for fact in facts],'providerResult':extracted,'retrievedEvidence':evidence,'unavailableSources':unavailable}

def draft_graphic(provider,story,article,sources,evidence,note,opportunities=None,unavailable_sources=None):
    from .pipeline import validate_opportunities
    opportunities=opportunities or []
    extracted=provider.complete(
        'Extract a small useful financial, market or technology chart ONLY from explicit numbers in retrieved source text. '
        'No fabricated values, estimates, calculations, currency conversion, interpolated points or inferred totals. '
        'If fewer than two comparable source values exist, return {"error":"insufficient_data"}. '
        'Return {graphic:{type:bar|line|area|donut,unit:string,period:string,sourceUrl:string,sourceName:string,methodology:{ru,kk,en},verified:false,'
        'labels:[{ru,kk,en}],series:[{name:{ru,kk,en},values:[number]}]},evidence:[{row:zero_based_index,series:zero_based_index,'
        'sourceUrl,quote:literal_source_excerpt_max_400_characters,numberText:literal_number_within_quote}]}. '
        'Every value needs one evidence entry. Keep the original scale and unit; never rescale numbers. '
        'Prefer 2–12 rows and 1–3 series. Use identical periods and units, disclose limitations in methodology.',
        {'retrievedEvidence':evidence,'unavailableSources':unavailable_sources or [],'story':story,'previousGraphic':article.get('graphic'),'editorNote':note},8000)
    if extracted.get('error'):raise ValueError('Insufficient comparable numeric source data; editor must supply a dataset')
    graphic=validate_graphic_result(extracted,evidence)
    draft=provider.complete(
        'Write an original short Creata News explanation of this source-backed chart. '
        'Return title,lead,body,summary,actions; EACH is {ru:string,kk:string,en:string}. '
        'Return opportunityIds:string[] containing only exact supplied opportunity IDs used in actions, or []. '
        'Link practical actions only to supplied opportunities; never invent IDs or application links. '
        'Keep numbers, units, period and source attribution consistent. No forecasts or investment advice. '
        'Describe what the data shows and its limitations. Respect the editorial note. Do not invent causes.',
        {'graphic':graphic,'sourceEvidence':extracted['evidence'],'editorNote':note,'previousDraft':article,'opportunities':opportunities},9000)
    validate_opportunities(validate_draft(draft),opportunities)
    review=provider.complete(
        'Review this chart and explanation against the literal source evidence. Check comparable periods, units, currency, '
        'chart choice, dates and translation consistency. Return reviewNotes (array of strings). '
        'Flag uncertainty; never claim that external verification was performed.',
        {'graphic':graphic,'draft':draft,'sourceEvidence':extracted['evidence'],'opportunities':opportunities},3000)
    notes=review.get('reviewNotes')
    if not isinstance(notes,list) or any(not isinstance(v,str) for v in notes):raise ValueError('Invalid graphic review output')
    notes=list(notes)
    notes.insert(0,'Chart data requires human source verification before approval / График требует проверки чисел редактором / Сандарды редактор тексеруі керек')
    draft.update(graphic=graphic,sources=sources,reviewNotes=notes)
    return draft
