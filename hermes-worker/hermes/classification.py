"""Conservative classification extraction: claims and names must occur in supplied texts."""
import math
import re

ENTITY_TYPES=('person','organization','place','product','event','other')

def source_texts(story):
    texts={}
    for item in story.get('evidence',[]):
        if isinstance(item,dict) and isinstance(item.get('url'),str) and isinstance(item.get('text'),str) and item['text'].strip():texts[item['url']]=' '.join(item['text'].split())
    if isinstance(story.get('url'),str) and isinstance(story.get('text'),str) and story['text'].strip():texts.setdefault(story['url'],' '.join(story['text'].split()))
    return texts

def validate_classification(result,story,legacy=False,categories=None):
    if not isinstance(result,dict):raise ValueError('Classification must be an object')
    score=result.get('score')
    if type(score) not in (int,float) or not 0<=score<=100 or not math.isfinite(score):raise ValueError('Classification score must be 0-100')
    for field,limit in (('categoryId',100),('region',200),('reason',4000)):
        if not isinstance(result.get(field),str) or len(result[field])>limit or (field!='region' and not result[field].strip()):raise ValueError('Invalid classification '+field)
    if categories is not None:
        allowed={category['id'] for category in categories if isinstance(category,dict) and isinstance(category.get('id'),str)}
        if result['categoryId'] not in allowed:raise ValueError('Classification category must belong to the supplied taxonomy')
    tags=result.get('tags')
    if not isinstance(tags,list) or len(tags)>30 or any(not isinstance(tag,str) or not tag.strip() or len(tag)>100 for tag in tags):raise ValueError('Invalid classification tags')
    texts=source_texts(story)
    for field in ('facts','entities'):
        # Only legacy transport callers may omit lists. New model output must provide both.
        if legacy and field not in result:result[field]=[]
        values=result.get(field)
        if not isinstance(values,list) or len(values)>50:raise ValueError('Classification '+field+' must be an array of at most 50 objects')
        for value in values:
            if not isinstance(value,dict):raise ValueError('Invalid classification '+field+' entry')
            urls=value.get('sourceUrls')
            if not isinstance(urls,list) or not 1<=len(urls)<=10 or any(not isinstance(url,str) or url not in texts for url in urls) or len(set(urls))!=len(urls):raise ValueError('Classification provenance requires supplied source text')
            if field=='facts':
                claim=value.get('claim');uncertainty=value.get('uncertainty')
                if not isinstance(claim,str) or not claim.strip() or len(claim)>1000 or not isinstance(uncertainty,str) or len(uncertainty)>1000:raise ValueError('Invalid source-backed classification fact')
                if any(' '.join(claim.split()) not in texts[url] for url in urls):raise ValueError('Classification claim must be a literal supplied source excerpt')
            else:
                name=value.get('name')
                if not isinstance(name,str) or not name.strip() or len(name)>200 or value.get('type') not in ENTITY_TYPES:raise ValueError('Invalid named entity')
                pattern=r'(?<!\w)'+re.escape(' '.join(name.split()))+r'(?!\w)'
                if any(not re.search(pattern,texts[url],re.IGNORECASE) for url in urls):raise ValueError('Named entity does not occur in supplied source text')
    return result
