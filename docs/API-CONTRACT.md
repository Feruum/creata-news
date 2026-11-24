# Creata v1 implementation contract

All dates ISO8601 UTC; Locale = ru|kk|en; localized fields = {ru:string,kk:string,en:string}. Responses JSON, errors {error:string,code?:string}. Lists {items:[],total?:number}. Browser credentials include cookies, mutations send X-CSRF-Token from /auth/me. Base /api/creata/v1. API port 5555, web 3002, editorial 6868, Hermes admin 5174. Vite proxies /api to API. Public web rewrites /api. Admin app location is / (separate origins locally). Go API owns persistence and authorization. API permits only configured exact frontend origins.

Article fields: id,slug,title,lead,body,summary,actions (all localized text; body plain paragraphs separated by newline),layer(world|eco),categoryId,region,tags:string[],authorId,author?:Author,sources:SourceRef[],reviewNotes:string[],media?:{url,caption,credit,rights},status(new|writing|review|rework|approved|published|rejected),revision:number,approvedRevision?:number,publishedRevision?:number,publishedAt?,scheduledAt?,createdAt,updatedAt. Article.actions is localized text (one action per newline). SourceRef={url,title,publishedAt?}. Author={id,name,bio:localized,avatarUrl?,role?}. Category={id,slug,name:localized}. Opportunity={id,title:localized,description:localized,organization,type,url,country,deadline,status(pending|approved|rejected),categoryId?}. Public articles use last published revision snapshot; drafts/withdrawn never public. Changing head after approval clears approval; changing already-published article preserves last live snapshot until revised publication. Public corrections list comes from publication revision notes.

User={id,name,email?,locale,stage,topics:string[],frequency(daily|weekly),paused:boolean,roles:string[],verticals:string[],telegramLinked:boolean}. Roles reader,author,editor,chief_editor,admin. Author edits own; editor edits/reviews/publishes assigned verticals; chief all editorial + eco publication; admin users/settings/sources; admin needs chief role for eco. No client-asserted roles. Dev bootstrap only with explicit CREATA_DEV_AUTH=1 and loopback origin; never production.

## Public and account

The public article list accepts `limit` (1–100) and the returned `nextCursor` to page through the same filters. Set `view=summary` for cards, which carry localized `readingMinutes` and omit the article body. Set `translated=1` to require a title in the requested locale and `includeTotal=0` on paginated requests that only need a continuation cursor. The default full snapshot remains available. `GET /public/regions` returns the regions used by navigation. `GET /public/opportunities?ids=id1,id2` selects the listed active, approved opportunities.

Article list, article detail, and region responses support `ETag` and conditional `If-None-Match` requests. Publishing, correcting, withdrawing, and editing an author invalidate these validators as part of the database update. Draft edits keep the current published validator. Public SSR uses a bounded memory cache and validates with Go before reuse.
GET /public/articles?locale&layer&category&region&q&authorId
GET /public/articles/{slug}; GET /public/categories; GET /public/authors; GET /public/authors/{id}; GET /public/opportunities; GET /public/opportunities/{id}
POST /public/events {type:view|save|opportunity_click,targetId,eventId?}
GET /auth/me => {user:User|null,csrfToken?:string,providers:{google:boolean,telegram:boolean},devAuth?:boolean}
GET /auth/{google|telegram}/start?returnTo=relative-path; POST /auth/logout
POST /auth/dev-login {role?:reader|author|editor|chief_editor|admin} only explicit local development
PATCH /account {locale,stage,topics,frequency,paused,name?} => User
GET /account/saved => {articles:Article[],opportunities:Opportunity[]}
POST|DELETE /account/saved/{article|opportunity}/{id}
POST /account/telegram-link => {url,expiresAt}

## Editorial
GET|POST /admin/articles; GET|PATCH /admin/articles/{id}; PATCH body article editable fields + expectedRevision
POST /admin/articles/{id}/actions {action:submit|approve|publish|schedule|unpublish|reject|rework,expectedRevision,scheduledAt?,note?}
GET /admin/articles/{id}/revisions => {items:Revision[]} where Revision={revision,title,lead,body,summary,actions,sources,reviewNotes,createdAt,actorId?,note?}
GET /admin/bank => {items:Story[]}; POST /admin/bank/{id}/draft
Story={id,title,text,url,sourceId,sourceType,locale,categoryId?,region?,tags:string[],score:number,reason?,status:string,sources:SourceRef[],createdAt}
GET|POST /admin/opportunities; PATCH /admin/opportunities/{id}
GET|POST /admin/authors; PATCH /admin/authors/{id}
GET /admin/users; PATCH /admin/users/{id} {roles,verticals}
GET /admin/analytics => {published:number,stories:number,subscribers:number,views:number,saves:number,opportunityClicks:number,statuses:Record<string,number>,daily:[],aiCostUsd:number}
GET /admin/audit => {items:[{id,actorId,action,targetId,createdAt,detail?}]}
POST /admin/media multipart file,credit,rights,caption => {url,credit,rights,caption}
GET /admin/digests; POST /admin/digests/preview {locale,kind:'channel'} => Digest
POST /admin/digests/{id}/approve; POST /admin/digests/{id}/send
Digest={id,locale,kind,status,articles:Article[],opportunities:Opportunity[],outsideInterest?:Article,createdAt,text?}

## Hermes control
GET|POST /admin/sources; PATCH /admin/sources/{id}; POST /admin/sources/{id}/{check|run}
Source={id,name,type:rss|website|telegram,url,locale,topics:string[],intervalMinutes:number,active:boolean,lastRunAt?,lastError?,nextRunAt?}
GET /admin/jobs; GET /admin/runs (same job list, source job filter); POST /admin/jobs/{id}/{retry|cancel}
Job={id,kind:check|collect|classify|draft|rework,status:queued|running|succeeded|failed|cancelled|not_configured,payload:object,result?:object,error?:string,attempts:number,createdAt,updatedAt,leaseToken?,leaseUntil?}
GET|PATCH /admin/settings => Settings (no secrets returned)
Settings={editorialInstructions:string,scoreThreshold:number,ai:{endpoint:string,model:string,keyConfigured:boolean,dailyBudgetUsd:number,spentTodayUsd:number,inputPricePerMillion:number,outputPricePerMillion:number},digestHour:number,timezone:string}. PATCH accepts ai.apiKey write-only.

## Worker service (Authorization Bearer CREATA_WORKER_TOKEN)
POST /internal/jobs/claim {workerId,kinds?:string[]} => {job:Job|null}; claimed payload contains source for collect/check, story for classify, and article+story+expectedRevision+note for draft/rework
POST /internal/jobs/{id}/heartbeat {leaseToken}
POST /internal/jobs/{id}/complete {leaseToken,result}
POST /internal/jobs/{id}/fail {leaseToken,error,retryable:boolean,notConfigured?:boolean}
collect/check result {items:[{externalId?,url,title,text,locale,publishedAt?,author?}],checked?:boolean}; collector normalizes and restricts network targets; backend also validates fields
classify result {categoryId,region,tags,score,reason,entities?,facts?}
draft/rework result {title,lead,body,summary,actions,sources,reviewNotes} localized fields, all three locales required. Complete checks expectedRevision before writing new revision and moves to review, never publication.
GET /internal/ai-config => Settings.ai + {apiKey,editorialInstructions,categories:Category[]}
POST /internal/ai/reserve {requestId,estimatedCostUsd} => {reservationId}; atomic daily budget reservation; POST /internal/ai/commit {reservationId,costUsd,inputTokens,outputTokens}; no API calls without configured endpoint/model/key/budget/prices

## Postiz

| Operation under `/api/creata/v1` | Purpose |
|---|---|
| `GET /admin/postiz/channels` | Read configuration state, draft/now mode and connected accounts with support status. |
| `GET /admin/articles/{id}/postiz/preview?provider=threads&locale=kk` | Preview the exact localized text and permanent article or chart link. |
| `POST /admin/articles/{id}/actions` | For publish and schedule, optional `postizTargets` contains account and language pairs, optional library `mediaId` and optional YouTube `visibility`. |
| `GET /admin/articles/{id}/postiz` | Read each version's delivery status, attempts, Postiz ID, error and published link when available. |

The API persists these deliveries separately from public article data. [Operation and verification](PIPELINE.md)

## Bot service (Authorization Bearer CREATA_BOT_SERVICE_TOKEN)
POST /internal/bot/users/upsert {telegramId:string,username?,locale?} => User
PATCH /internal/bot/users/{telegramId} {locale?,topics?,frequency?,paused?} => User
POST /internal/bot/link {token,telegramId} => User
GET /internal/bot/topics => {items:Category[]}
POST /internal/bot/digest {telegramId} => Digest
POST /internal/deliveries/claim {workerId} => {delivery:null|{id,chatId,text,links:[{label,url}],leaseToken}}
POST /internal/deliveries/{id}/complete {leaseToken,messageId}
POST /internal/deliveries/{id}/fail {leaseToken,error,retryAfterSeconds?,unknown?:boolean}

## Setup and safety
No production demo seed. Explicit seed command imports reference data as development-only records. Public empty/error states are real. Google and Telegram configuration absent => provider marked unavailable. Callback uses state+nonce+PKCE where supported, validates signature/audience/issuer, ties provider subject to user; no automatic merging by contact email. Source content is untrusted data, never executable instructions. URLs restricted to http/https public hosts; reject private/loopback/link-local targets including redirects (explicit test-only local fixture allow flag). Hermes never has publish rights. Channel digest needs editorial approval. Confirmed deliveries are not retried; unknown Telegram outcomes require manual resolution. API refuses production dev auth and missing service/session secrets. New database schema only; no historical event migrations.
