// Generated from backend/creata/openapi.json. Run bun run contracts.
export interface schemas {
  "Locale": "ru" | "kk" | "en"
  "LocalizedText": { "ru": string; "kk": string; "en": string; }
  "ActionGuide": { "audience": schemas["LocalizedText"]; "steps": schemas["LocalizedText"]; "sourceUrl": string; }
  "StoryUpdate": { "id": string; "occurredAt": string; "title": schemas["LocalizedText"]; "body": schemas["LocalizedText"]; "sourceUrl": string; "articleSlug"?: string; }
  "FollowedStories": { "items": Array<schemas["Article"]>; "unreadUpdateIds": { [key: string]: Array<string>; }; }
  "StoryReadInput": { "updateIds": Array<string>; }
  "StoryFollowUpInput": { "dueDate": string; "assigneeId": string; }
  "StoryFollowUp": { "articleId": string; "title": schemas["LocalizedText"]; "assigneeId": string; "assigneeName": string; "dueDate": string; "updatedAt": string; }
  "SourceRef": { "url": string; "title": string; "publishedAt"?: string; }
  "Media": { "url": string; "caption": string; "credit": string; "rights": string; }
  "ArticlePhoto": { "url": string; "caption": string; "credit": string; "rights": string; "sourceUrl": string; "licenseUrl": string; }
  "ArticlePhotos": Array<schemas["ArticlePhoto"]>
  "Author": { "id": string; "name": string; "bio": schemas["LocalizedText"]; "avatarUrl"?: string; "role"?: string; }
  "Category": { "id": string; "slug": string; "name": schemas["LocalizedText"]; }
  "Article": schemas["ArticleFields"] & { "body": schemas["LocalizedText"]; }
  "ArticleSummary": schemas["ArticleFields"] & { "readingMinutes": { "ru": number; "kk": number; "en": number; }; }
  "ArticleFields": { "id": string; "slug": string; "categoryId": string; "region": string; "authorId": string; "createdAt": string; "updatedAt": string; "publishedAt"?: string; "scheduledAt"?: string; "title": schemas["LocalizedText"]; "lead": schemas["LocalizedText"]; "summary": schemas["LocalizedText"]; "actions": schemas["LocalizedText"]; "actionGuide"?: schemas["ActionGuide"]; "storyUpdates"?: Array<schemas["StoryUpdate"]>; "layer": "world" | "eco"; "tags": Array<string>; "sources": Array<schemas["SourceRef"]>; "reviewNotes": Array<string>; "author"?: schemas["Author"]; "media"?: schemas["Media"]; "photos"?: schemas["ArticlePhotos"]; "status": "new" | "writing" | "review" | "rework" | "approved" | "scheduled" | "published" | "unpublished" | "rejected"; "revision": number; "approvedRevision"?: number; "publishedRevision"?: number; "corrections"?: Array<{ "revision": number; "publishedAt": string; "note": string; }>; "editorId"?: string; "format"?: "article" | "graphic"; "graphic"?: schemas["GraphicData"]; "opportunityIds"?: Array<string>; "researchBrief"?: schemas["ResearchBrief"]; }
  "Revision": { "id"?: string; "slug"?: string; "categoryId"?: string; "region"?: string; "authorId"?: string; "createdAt": string; "updatedAt"?: string; "publishedAt"?: string; "scheduledAt"?: string; "title": schemas["LocalizedText"]; "lead": schemas["LocalizedText"]; "body": schemas["LocalizedText"]; "summary"?: schemas["LocalizedText"]; "actions"?: schemas["LocalizedText"]; "actionGuide"?: schemas["ActionGuide"]; "storyUpdates"?: Array<schemas["StoryUpdate"]>; "layer"?: "world" | "eco"; "tags"?: Array<string>; "sources"?: Array<schemas["SourceRef"]>; "reviewNotes"?: Array<string>; "author"?: schemas["Author"]; "media"?: schemas["Media"]; "status"?: "new" | "writing" | "review" | "rework" | "approved" | "scheduled" | "published" | "unpublished" | "rejected"; "revision": number; "approvedRevision"?: number; "publishedRevision"?: number; "actorId"?: string; "note"?: string; }
  "Opportunity": { "id": string; "title": schemas["LocalizedText"]; "description": schemas["LocalizedText"]; "organization": string; "type": string; "url": string; "country": string; "deadline": string; "status": "pending" | "approved" | "rejected"; "categoryId"?: string; "photo"?: string; "forWhom"?: schemas["LocalizedText"]; "photoRights"?: { "credit": string; "rights": string; }; "languages"?: Array<schemas["Locale"]>; "needsRightsReview"?: boolean; }
  "User": { "adminTheme"?: "light" | "dark" | "cats"; "id": string; "name": string; "email"?: string; "locale": "ru" | "kk" | "en"; "stage": "exploring" | "trying" | "building"; "topics": Array<string>; "frequency": "daily" | "weekly"; "paused": boolean; "roles": Array<string>; "verticals": Array<string>; "telegramLinked": boolean; }
  "Source": { "id": string; "name": string; "type": "rss" | "website" | "telegram"; "url": string; "locale": "ru" | "kk" | "en"; "topics": Array<string>; "intervalMinutes": number; "active": boolean; "lastRunAt"?: string; "lastError"?: string; "nextRunAt"?: string; }
  "Story": { "id": string; "title": string; "text": string; "url": string; "sourceId": string; "sourceType": string; "locale": "ru" | "kk" | "en"; "categoryId"?: string; "region"?: string; "tags": Array<string>; "score": number; "reason"?: string; "status": string; "sources": Array<schemas["SourceRef"]>; "createdAt": string; "evidence"?: Array<{ "url": string; "title": string; "text": string; "publishedAt"?: string; "author"?: string; "sourceId"?: string; }>; "graphicId"?: string; "photos"?: schemas["ArticlePhotos"]; "facts"?: Array<schemas["ClassificationFact"]>; "entities"?: Array<schemas["NamedEntity"]>; }
  "Job": { "id": string; "kind": "check" | "collect" | "classify" | "draft" | "rework"; "status": "queued" | "running" | "succeeded" | "failed" | "cancelled" | "not_configured"; "payload": { [key: string]: unknown; }; "result"?: { [key: string]: unknown; }; "error"?: string; "attempts": number; "createdAt": string; "updatedAt": string; "leaseToken"?: string; "leaseUntil"?: string; "checkpoints"?: { [key: string]: schemas["JobCheckpoint"]; }; "progress"?: { "phase": string; "status": string; "updatedAt": string; }; }
  "AISettings": { "endpoint": string; "model": string; "keyConfigured": boolean; "exaKeyConfigured"?: boolean; "dailyBudgetUsd": number; "spentTodayUsd": number; "inputPricePerMillion": number; "outputPricePerMillion": number; }
  "Settings": { "editorialInstructions": string; "scoreThreshold": number; "ai": schemas["AISettings"]; "digestHour": number; "morningDigestHour"?: number; "timezone": string; }
  "Digest": { "id": string; "locale": "ru" | "kk" | "en"; "kind": string; "status": string; "articles": Array<schemas["Article"]>; "opportunities": Array<schemas["Opportunity"]>; "outsideInterest"?: schemas["Article"]; "createdAt": string; "morningDate"?: string; "text"?: string; }
  "MorningDigestStatus": { "edition"?: { "id": string; "date": string; "status": string; "articleCount": number; "opportunityCount": number; }; "lastCollectionAt"?: string; }
  "Delivery": { "id": string; "chatId": string; "text": string; "links": Array<{ "label": string; "url": string; }>; "leaseToken": string; }
  "AuthState": { "user": schemas["User"] | null; "csrfToken"?: string; "providers": { "google": boolean; "telegram": boolean; }; "devAuth"?: boolean; }
  "Error": { "error": string; "code"?: string; }
  "OK": { "ok": boolean; }
  "Launch": { "id": string; "slug": string; "name": string; "tagline": schemas["LocalizedText"]; "description": schemas["LocalizedText"]; "team": schemas["LocalizedText"]; "url": string; "launchDate": string; "country": string; "categoryId": string; "photo"?: string; "status": "draft" | "published" | "archived"; "revision": number; "createdAt": string; "updatedAt": string; "logoUrl"?: string; "tags"?: Array<string>; "votes"?: number; "commentsCount"?: number; }
  "LaunchCreate": { "slug": string; "name": string; "tagline": schemas["LocalizedText"]; "description": schemas["LocalizedText"]; "team": schemas["LocalizedText"]; "url": string; "launchDate": string; "country": string; "categoryId": string; "photo"?: string; "status"?: "draft" | "published" | "archived"; "logoUrl"?: string; "tags"?: Array<string>; }
  "LaunchUpdate": { "slug"?: string; "name"?: string; "tagline"?: schemas["LocalizedText"]; "description"?: schemas["LocalizedText"]; "team"?: schemas["LocalizedText"]; "url"?: string; "launchDate"?: string; "country"?: string; "categoryId"?: string; "photo"?: string; "status"?: "draft" | "published" | "archived"; "expectedRevision": number; "logoUrl"?: string; "tags"?: Array<string>; }
  "GraphicData": { "type": "bar" | "line" | "area" | "donut"; "unit": string; "period": string; "sourceUrl": string; "sourceName": string; "methodology": schemas["LocalizedText"]; "verified": boolean; "labels": Array<schemas["LocalizedText"]>; "series": Array<{ "name": schemas["LocalizedText"]; "values": Array<number>; }>; }
  "NewspaperItem": { "id": string; "articleId": string; "photo"?: schemas["Media"]; }
  "NewspaperCreate": { "slug": string; "number": string; "date": string; "title": string; "description"?: string; "locale": schemas["Locale"]; "cover"?: schemas["Media"]; "items": Array<schemas["NewspaperItem"]>; }
  "NewspaperUpdate": { "slug"?: string; "number"?: string; "date"?: string; "title"?: string; "description"?: string; "locale"?: schemas["Locale"]; "cover"?: schemas["Media"]; "items"?: Array<schemas["NewspaperItem"]>; "expectedRevision": number; }
  "NewspaperIssue": { "slug": string; "number": string; "date": string; "title": string; "description": string; "locale": schemas["Locale"]; "cover": schemas["Media"]; "items": Array<schemas["NewspaperItem"]>; "id": string; "revision": number; "status": "draft" | "published"; "createdAt": string; "updatedAt": string; "creatorId": string; "publishedAt"?: string; "publishedRevision"?: number; }
  "NewspaperPublication": { "slug": string; "number": string; "date": string; "title": string; "description": string; "locale": schemas["Locale"]; "cover": schemas["Media"]; "id": string; "revision": number; "pages": number; "publishedAt"?: string; "publishedRevision": number; "entries": Array<{ "id": string; "articleId": string; "article": schemas["Article"]; "photo": schemas["Media"]; }>; "format"?: "original" | "managed"; "html"?: string; "pdf"?: string; }
  "LaunchComment": { "id": string; "launchId": string; "authorName": string; "body": string; "createdAt": string; "canDelete": boolean; }
  "LaunchVoteState": { "voted": boolean; "votes": number; }
  "AboutContent": { "ru": { "title": string; "body": string; "second": string; "values": Array<Array<string>>; }; "kk": { "title": string; "body": string; "second": string; "values": Array<Array<string>>; }; "en": { "title": string; "body": string; "second": string; "values": Array<Array<string>>; }; }
  "EcosystemContent": { "copy": { "ru": { "title": string; "description": string; "eyebrow": string; "hero": Array<string>; "intro": string; "explore": string; "meet": string; "photoCredit": string; "who": string; "whoTitle": string; "whoBody": string; "whoSecond": string; "group": string; "companies": string; "layers": string; "route": string; "model": string; "modelTitle": string; "modelLead": string; "products": string; "productsTitle": string; "productsLead": string; "filterLabel": string; "filters": { "all": string; "education": string; "games": string; "business": string; "media": string; }; "productCount": string; "openProduct": string; "stories": string; "storiesTitle": string; "issueTitle": string; "issueBody": string; "readIssue": string; "next": string; "nextTitle": string; "nextBody": string; "opportunities": string; "contact": string; "backToProducts": string; "storiesFilter": string; "allTopics": string; "allRegions": string; "noFiltered": string; }; "kk": { "title": string; "description": string; "eyebrow": string; "hero": Array<string>; "intro": string; "explore": string; "meet": string; "photoCredit": string; "who": string; "whoTitle": string; "whoBody": string; "whoSecond": string; "group": string; "companies": string; "layers": string; "route": string; "model": string; "modelTitle": string; "modelLead": string; "products": string; "productsTitle": string; "productsLead": string; "filterLabel": string; "filters": { "all": string; "education": string; "games": string; "business": string; "media": string; }; "productCount": string; "openProduct": string; "stories": string; "storiesTitle": string; "issueTitle": string; "issueBody": string; "readIssue": string; "next": string; "nextTitle": string; "nextBody": string; "opportunities": string; "contact": string; "backToProducts": string; "storiesFilter": string; "allTopics": string; "allRegions": string; "noFiltered": string; }; "en": { "title": string; "description": string; "eyebrow": string; "hero": Array<string>; "intro": string; "explore": string; "meet": string; "photoCredit": string; "who": string; "whoTitle": string; "whoBody": string; "whoSecond": string; "group": string; "companies": string; "layers": string; "route": string; "model": string; "modelTitle": string; "modelLead": string; "products": string; "productsTitle": string; "productsLead": string; "filterLabel": string; "filters": { "all": string; "education": string; "games": string; "business": string; "media": string; }; "productCount": string; "openProduct": string; "stories": string; "storiesTitle": string; "issueTitle": string; "issueBody": string; "readIssue": string; "next": string; "nextTitle": string; "nextBody": string; "opportunities": string; "contact": string; "backToProducts": string; "storiesFilter": string; "allTopics": string; "allRegions": string; "noFiltered": string; }; }; "photos": Array<{ "src": string; "label": string; "caption": { "ru": string; "kk": string; "en": string; }; }>; "layers": Array<{ "id": string; "number": string; "title": { "ru": string; "kk": string; "en": string; }; "subtitle": { "ru": string; "kk": string; "en": string; }; "description": { "ru": string; "kk": string; "en": string; }; "photo": string; "caption": { "ru": string; "kk": string; "en": string; }; "companies": Array<{ "name": string; "href": string; "note": { "ru": string; "kk": string; "en": string; }; }>; }>; "products": Array<{ "id": string; "name": string; "category": string; "href": string; "audience": { "ru": string; "kk": string; "en": string; }; "description": { "ru": string; "kk": string; "en": string; }; }>; "issueCover": string; "groupUrl": string; "contactUrl": string; "groupLabel": string; "issueSlug": string; "issueNumber": string; }
  "OriginalIssueContent": { "slug": string; "number": string; "date": string; "title": string; "description": string; "locale": schemas["Locale"]; "pages": number; "cover": schemas["Media"]; "pdf": string; "html": string; }
  "SitePage": { "id": string; "slug": "about" | "ecosystem" | "newspaper-01"; "content": schemas["AboutContent"] | schemas["EcosystemContent"] | schemas["OriginalIssueContent"]; "revision": number; "status": "draft" | "published"; "publishedRevision"?: number; "createdAt"?: string; "updatedAt"?: string; }
  "MediaAsset": { "url": string; "caption"?: string; "credit": string; "rights": string; "id": string; "key"?: string; "mime"?: string; "bytes"?: number; "revision": number; "createdAt"?: string; "legacyPath"?: string; "needsRightsReview"?: boolean; "referenced"?: boolean; "references"?: Array<{ "kind": string; "id": string; }>; "width"?: number; "height"?: number; "verifiedMp4"?: boolean; }
  "ResearchBrief": { "facts": Array<{ "claim": string; "sourceUrls": Array<string>; "numbers"?: Array<{ "value": string; "sourceUrl": string; "quote": string; }>; "dates"?: Array<{ "value": string; "sourceUrl": string; "quote": string; }>; }>; "conflicts": Array<string>; "singleSourceClaims": Array<string>; "retrievedEvidence": Array<{ "url": string; "title"?: string; "text": string; "publishedAt"?: string; "author"?: string; "discoveredBy"?: "exa"; }>; "unavailableSources": Array<string>; }
  "JobCheckpoint": { "status": "running" | "succeeded" | "failed"; "output"?: { [key: string]: unknown; }; "error"?: string; "completedAt"?: string; "usage"?: Array<{ "reservationId": string; "costUsd": number; "inputTokens": number; "outputTokens": number; }>; "updatedAt"?: string; }
  "StorySimilarity": { "leftId": string; "rightId": string; "score": number; "reasons": Array<string>; "leftTitle"?: string; "rightTitle"?: string; "leftUpdatedAt"?: string; "rightUpdatedAt"?: string; }
  "StoryEvidence": { "story": schemas["Story"]; "rawItems": Array<{ [key: string]: unknown; }>; "linkedArticleIds": Array<string>; "version": string; }
  "StoryMerge": { "id": string; "leftId": string; "rightId": string; "status": "merged" | "undone" | "split"; "createdAt": string; "undoneAt"?: string; }
  "StageMetrics": { "durations": Array<{ "stage": string; "completedVisits": number; "averageSeconds": number; "medianSeconds": number; "p90Seconds": number; "currentCount": number; "averageCurrentAgeSeconds": number; }>; "funnel": Array<{ "stage": string; "count": number; "rate": number; }>; "transitions": Array<{ "from": string; "to": string; "count": number; }>; "articleCount": number; "measuredAt": string; }
  "ClassificationFact": { "claim": string; "sourceUrls": Array<string>; "uncertainty": string; }
  "NamedEntity": { "name": string; "type": "person" | "organization" | "place" | "product" | "event" | "other"; "sourceUrls": Array<string>; }
  "PostizTarget": { "channelId": string; "locale": schemas["Locale"]; "visibility"?: "private" | "unlisted" | "public"; "mediaId"?: string; }
  "PostizChannel": { "id": string; "name": string; "provider": string; "supported": boolean; "reason"?: string; }
  "PostizChannels": { "configured": boolean; "mode": "draft" | "now"; "items": Array<schemas["PostizChannel"]>; }
  "PostizDelivery": { "id": string; "channelId": string; "channelName": string; "provider": string; "locale": schemas["Locale"]; "revision": number; "status": "scheduled" | "queued" | "sending" | "accepted" | "draft" | "published" | "failed" | "uncertain" | "cancelled" | "missing_media"; "mode": "draft" | "now"; "content": string; "articleUrl": string; "attempts": number; "postId"?: string; "releaseUrl"?: string; "error"?: string; "updatedAt": string; }
  "PostizPreview": { "content": string; "articleUrl": string; }
  "LaunchSubmission": { "name": string; "tagline": schemas["LocalizedText"]; "description": schemas["LocalizedText"]; "team": schemas["LocalizedText"]; "url": string; "launchDate": string; "country": string; "categoryId": string; "photo": string; "id": string; "status": "draft" | "in_review" | "changes_requested" | "published" | "unpublished"; "revision": number; "createdAt": string; "updatedAt": string; "reviewNote"?: string; "launchSlug"?: string; }
  "LaunchSubmissionCreate": { "name"?: string; "tagline"?: schemas["LocalizedText"]; "description"?: schemas["LocalizedText"]; "team"?: schemas["LocalizedText"]; "url"?: string; "launchDate"?: string; "country"?: string; "categoryId"?: string; "photo"?: string; }
  "LaunchSubmissionUpdate": { "name"?: string; "tagline"?: schemas["LocalizedText"]; "description"?: schemas["LocalizedText"]; "team"?: schemas["LocalizedText"]; "url"?: string; "launchDate"?: string; "country"?: string; "categoryId"?: string; "photo"?: string; "expectedRevision": number; }
  "LaunchSubmissionSubmit": { "expectedRevision": number; }
  "LaunchSubmissionReview": { "expectedRevision": number; "action": "publish" | "request_changes"; "note"?: string; }
}
export interface endpoints {
  "GET /api/creata/v1/public/articles": { request: unknown; response: { "items": Array<schemas["Article"] | schemas["ArticleSummary"]>; "total"?: number; "nextCursor"?: string; } }
  "GET /api/creata/v1/public/articles/{slug}": { request: unknown; response: schemas["Article"] }
  "GET /api/creata/v1/public/regions": { request: unknown; response: { "items": Array<string>; } }
  "GET /api/creata/v1/public/categories": { request: unknown; response: { "items": Array<schemas["Category"]>; "total"?: number; } }
  "GET /api/creata/v1/public/authors": { request: unknown; response: { "items": Array<schemas["Author"]>; "total"?: number; } }
  "GET /api/creata/v1/public/authors/{id}": { request: unknown; response: schemas["Author"] }
  "GET /api/creata/v1/public/opportunities": { request: unknown; response: { "items": Array<schemas["Opportunity"]>; "total"?: number; } }
  "GET /api/creata/v1/public/opportunities/{id}": { request: unknown; response: schemas["Opportunity"] }
  "POST /api/creata/v1/public/events": { request: { "type": "view" | "save" | "opportunity_click"; "targetId": string; "eventId"?: string; }; response: schemas["OK"] }
  "GET /api/creata/v1/auth/me": { request: unknown; response: schemas["AuthState"] }
  "POST /api/creata/v1/auth/dev-login": { request: { "role"?: string; }; response: { "user": schemas["User"]; } }
  "POST /api/creata/v1/auth/logout": { request: unknown; response: schemas["OK"] }
  "GET /api/creata/v1/auth/{provider}/start": { request: unknown; response: { [key: string]: unknown; } }
  "GET /api/creata/v1/auth/{provider}/callback": { request: unknown; response: { [key: string]: unknown; } }
  "PATCH /api/creata/v1/account": { request: { "adminTheme"?: "light" | "dark" | "cats"; [key: string]: unknown; }; response: schemas["User"] }
  "GET /api/creata/v1/account/saved": { request: unknown; response: { "articles": Array<schemas["Article"]>; "opportunities": Array<schemas["Opportunity"]>; } }
  "POST /api/creata/v1/account/telegram-link": { request: unknown; response: { "url": string; "expiresAt": string; } }
  "POST /api/creata/v1/account/saved/{kind}/{id}": { request: unknown; response: schemas["OK"] }
  "DELETE /api/creata/v1/account/saved/{kind}/{id}": { request: unknown; response: schemas["OK"] }
  "GET /api/creata/v1/account/followed-stories": { request: unknown; response: schemas["FollowedStories"] }
  "POST /api/creata/v1/account/followed-stories/{id}": { request: unknown; response: { "ok": boolean; } }
  "DELETE /api/creata/v1/account/followed-stories/{id}": { request: unknown; response: { "ok": boolean; } }
  "POST /api/creata/v1/account/followed-stories/{id}/read": { request: schemas["StoryReadInput"]; response: { "ok": boolean; } }
  "GET /api/creata/v1/admin/story-follow-ups": { request: unknown; response: { "items": Array<schemas["StoryFollowUp"]>; "total": number; } }
  "GET /api/creata/v1/admin/articles": { request: unknown; response: { "items": Array<schemas["Article"]>; "total"?: number; } }
  "POST /api/creata/v1/admin/articles": { request: { "actionGuide"?: schemas["ActionGuide"] | null; "storyUpdates"?: Array<schemas["StoryUpdate"]>; "photos"?: schemas["ArticlePhotos"]; [key: string]: unknown; }; response: schemas["Article"] }
  "PATCH /api/creata/v1/admin/articles/{id}": { request: { "expectedRevision": number; "actionGuide"?: schemas["ActionGuide"] | null; "storyUpdates"?: Array<schemas["StoryUpdate"]>; "photos"?: schemas["ArticlePhotos"]; [key: string]: unknown; }; response: schemas["Article"] }
  "GET /api/creata/v1/admin/articles/{id}": { request: unknown; response: schemas["Article"] }
  "GET /api/creata/v1/admin/articles/{id}/follow-up": { request: unknown; response: { "item": schemas["StoryFollowUp"] | null; "eligible": boolean; } }
  "PUT /api/creata/v1/admin/articles/{id}/follow-up": { request: schemas["StoryFollowUpInput"]; response: { "item": schemas["StoryFollowUp"]; } }
  "DELETE /api/creata/v1/admin/articles/{id}/follow-up": { request: unknown; response: { "ok": boolean; } }
  "GET /api/creata/v1/admin/opportunities": { request: unknown; response: { "items": Array<schemas["Opportunity"]>; "total"?: number; } }
  "POST /api/creata/v1/admin/opportunities": { request: { "photo"?: string; "forWhom"?: schemas["LocalizedText"]; [key: string]: unknown; }; response: schemas["Opportunity"] }
  "PATCH /api/creata/v1/admin/opportunities/{id}": { request: { "photo"?: string; "forWhom"?: schemas["LocalizedText"]; [key: string]: unknown; }; response: schemas["Opportunity"] }
  "GET /api/creata/v1/admin/authors": { request: unknown; response: { "items": Array<schemas["Author"]>; "total"?: number; } }
  "POST /api/creata/v1/admin/authors": { request: { [key: string]: unknown; }; response: schemas["Author"] }
  "PATCH /api/creata/v1/admin/authors/{id}": { request: { [key: string]: unknown; }; response: schemas["Author"] }
  "GET /api/creata/v1/admin/sources": { request: unknown; response: { "items": Array<schemas["Source"]>; "total"?: number; } }
  "POST /api/creata/v1/admin/sources": { request: { [key: string]: unknown; }; response: schemas["Source"] }
  "PATCH /api/creata/v1/admin/sources/{id}": { request: { [key: string]: unknown; }; response: schemas["Source"] }
  "POST /api/creata/v1/admin/articles/{id}/actions": { request: { "action": "submit" | "approve" | "publish" | "schedule" | "unpublish" | "reject" | "rework"; "expectedRevision": number; "scheduledAt"?: string; "note"?: string; "postizTargets"?: Array<schemas["PostizTarget"]>; }; response: schemas["Article"] }
  "GET /api/creata/v1/admin/articles/{id}/revisions": { request: unknown; response: { "items": Array<schemas["Revision"]>; } }
  "GET /api/creata/v1/admin/bank": { request: unknown; response: { "items": Array<schemas["Story"]>; } }
  "POST /api/creata/v1/admin/bank/{id}/draft": { request: unknown; response: schemas["Article"] }
  "GET /api/creata/v1/admin/jobs": { request: unknown; response: { "items": Array<schemas["Job"]>; } }
  "GET /api/creata/v1/admin/runs": { request: unknown; response: { "items": Array<schemas["Job"]>; } }
  "GET /api/creata/v1/admin/users": { request: unknown; response: { "items": Array<schemas["User"]>; } }
  "GET /api/creata/v1/admin/digests": { request: unknown; response: { "items": Array<schemas["Digest"]>; "channelDeliveryStatus"?: "available" | "channel_missing" | "bot_missing"; } }
  "GET /api/creata/v1/admin/audit": { request: unknown; response: { "items": Array<{ [key: string]: unknown; }>; } }
  "GET /api/creata/v1/admin/analytics": { request: unknown; response: { "published": number; "stories": number; "subscribers": number; "views": number; "saves": number; "opportunityClicks": number; "statuses": { [key: string]: number; }; "daily": Array<{ [key: string]: unknown; }>; "aiCostUsd": number; } }
  "PATCH /api/creata/v1/admin/users/{id}": { request: { "roles"?: Array<string>; "verticals"?: Array<string>; }; response: schemas["User"] }
  "GET /api/creata/v1/admin/settings": { request: unknown; response: schemas["Settings"] }
  "PATCH /api/creata/v1/admin/settings": { request: { [key: string]: unknown; }; response: schemas["Settings"] }
  "POST /api/creata/v1/admin/sources/{id}/check": { request: unknown; response: unknown }
  "POST /api/creata/v1/admin/sources/{id}/run": { request: unknown; response: unknown }
  "POST /api/creata/v1/admin/jobs/{id}/retry": { request: unknown; response: schemas["Job"] }
  "POST /api/creata/v1/admin/jobs/{id}/cancel": { request: unknown; response: schemas["Job"] }
  "GET /api/creata/v1/admin/digests/morning": { request: unknown; response: schemas["MorningDigestStatus"] }
  "POST /api/creata/v1/admin/digests/preview": { request: { "locale": "ru" | "kk" | "en"; "kind"?: string; }; response: schemas["Digest"] }
  "POST /api/creata/v1/admin/digests/{id}/approve": { request: unknown; response: schemas["Digest"] }
  "POST /api/creata/v1/admin/digests/{id}/send": { request: unknown; response: schemas["Digest"] }
  "POST /api/creata/v1/admin/media": { request: unknown; response: schemas["Media"] }
  "POST /api/creata/v1/internal/jobs/claim": { request: { "workerId": string; "kinds"?: Array<string>; }; response: { "job": schemas["Job"] | null; } }
  "POST /api/creata/v1/internal/jobs/{id}/heartbeat": { request: { [key: string]: unknown; }; response: schemas["OK"] }
  "POST /api/creata/v1/internal/jobs/{id}/complete": { request: { [key: string]: unknown; }; response: schemas["OK"] }
  "POST /api/creata/v1/internal/jobs/{id}/fail": { request: { [key: string]: unknown; }; response: schemas["OK"] }
  "GET /api/creata/v1/internal/ai-config": { request: unknown; response: { [key: string]: unknown; } }
  "POST /api/creata/v1/internal/ai/reserve": { request: { "requestId": string; "estimatedCostUsd": number; }; response: { "reservationId": string; } }
  "POST /api/creata/v1/internal/ai/commit": { request: { "reservationId": string; "costUsd": number; "inputTokens": number; "outputTokens": number; }; response: schemas["OK"] }
  "POST /api/creata/v1/internal/bot/users/upsert": { request: { "telegramId": string; "username"?: string; "locale"?: "ru" | "kk" | "en"; }; response: schemas["User"] }
  "PATCH /api/creata/v1/internal/bot/users/{telegramId}": { request: { [key: string]: unknown; }; response: schemas["User"] }
  "POST /api/creata/v1/internal/bot/link": { request: { "token": string; "telegramId": string; }; response: schemas["User"] }
  "GET /api/creata/v1/internal/bot/topics": { request: unknown; response: { "items": Array<schemas["Category"]>; } }
  "POST /api/creata/v1/internal/bot/digest": { request: { "telegramId": string; }; response: schemas["Digest"] }
  "POST /api/creata/v1/internal/deliveries/claim": { request: { "workerId": string; }; response: { "delivery": schemas["Delivery"] | null; } }
  "POST /api/creata/v1/internal/deliveries/{id}/complete": { request: { [key: string]: unknown; }; response: schemas["OK"] }
  "POST /api/creata/v1/internal/deliveries/{id}/fail": { request: { [key: string]: unknown; }; response: schemas["OK"] }
  "POST /api/creata/v1/internal/bot/digests/{id}/complete": { request: { "telegramId": string; "messageId"?: number; "error"?: string; "unknown"?: boolean; }; response: schemas["OK"] }
  "POST /api/creata/v1/internal/bot/digests/{id}/fail": { request: { "telegramId": string; "messageId"?: number; "error"?: string; "unknown"?: boolean; }; response: schemas["OK"] }
  "GET /api/creata/v1/admin/deliveries": { request: unknown; response: { "items": Array<{ [key: string]: unknown; }>; } }
  "POST /api/creata/v1/admin/deliveries/{id}/resolve": { request: { "action": "retry" | "mark_sent"; "messageId"?: number; }; response: { [key: string]: unknown; } }
  "GET /api/creata/v1/admin/staff": { request: unknown; response: { "items": Array<{ "id": string; "name": string; "roles": Array<string>; "verticals": Array<string>; }>; } }
  "POST /api/creata/v1/admin/bank/collect": { request: { "url": string; "locale": schemas["Locale"]; "topics"?: Array<string>; }; response: unknown }
  "GET /api/creata/v1/public/popular": { request: unknown; response: { "items": Array<schemas["Article"]>; "total"?: number; } }
  "GET /api/creata/v1/public/launches": { request: unknown; response: { "items": Array<schemas["Launch"]>; "total": number; } }
  "GET /api/creata/v1/public/launches/{slug}": { request: unknown; response: schemas["Launch"] }
  "GET /api/creata/v1/admin/launches": { request: unknown; response: { "items": Array<schemas["Launch"]>; "total": number; } }
  "POST /api/creata/v1/admin/launches": { request: schemas["LaunchCreate"]; response: schemas["Launch"] }
  "PATCH /api/creata/v1/admin/launches/{id}": { request: schemas["LaunchUpdate"]; response: schemas["Launch"] }
  "POST /api/creata/v1/admin/articles/{id}/graphic-draft": { request: { "expectedRevision": number; "note"?: string; }; response: unknown }
  "POST /api/creata/v1/admin/bank/{id}/graphic": { request: unknown; response: schemas["Article"] }
  "GET /api/creata/v1/admin/newspapers": { request: unknown; response: { "items": Array<schemas["NewspaperIssue"]>; "total": number; } }
  "POST /api/creata/v1/admin/newspapers": { request: schemas["NewspaperCreate"]; response: schemas["NewspaperIssue"] }
  "GET /api/creata/v1/admin/newspapers/{id}": { request: unknown; response: schemas["NewspaperIssue"] }
  "PATCH /api/creata/v1/admin/newspapers/{id}": { request: schemas["NewspaperUpdate"]; response: schemas["NewspaperIssue"] }
  "POST /api/creata/v1/admin/newspapers/{id}/publish": { request: { "expectedRevision": number; }; response: schemas["NewspaperIssue"] }
  "POST /api/creata/v1/admin/newspapers/{id}/unpublish": { request: { "expectedRevision": number; }; response: schemas["NewspaperIssue"] }
  "GET /api/creata/v1/admin/newspapers/{id}/preview": { request: unknown; response: schemas["NewspaperPublication"] }
  "GET /api/creata/v1/public/newspapers": { request: unknown; response: { "items": Array<schemas["NewspaperPublication"]>; "total": number; } }
  "GET /api/creata/v1/public/newspapers/{slug}": { request: unknown; response: schemas["NewspaperPublication"] }
  "GET /api/creata/v1/account/launch-votes": { request: unknown; response: { "items": Array<string>; "total": number; } }
  "POST /api/creata/v1/account/launches/{id}/vote": { request: { "voted": boolean; }; response: schemas["LaunchVoteState"] }
  "GET /api/creata/v1/public/launches/{slug}/comments": { request: unknown; response: { "items": Array<schemas["LaunchComment"]>; "total": number; } }
  "POST /api/creata/v1/account/launches/{id}/comments": { request: { "body": string; "clientId": string; }; response: schemas["LaunchComment"] }
  "DELETE /api/creata/v1/account/launches/{id}/comments/{commentId}": { request: unknown; response: { "ok": boolean; } }
  "GET /public/pages/{slug}": { request: unknown; response: schemas["SitePage"] }
  "GET /admin/pages": { request: unknown; response: { "items": Array<schemas["SitePage"]>; "total": number; } }
  "GET /admin/pages/{slug}": { request: unknown; response: schemas["SitePage"] }
  "PATCH /admin/pages/{slug}": { request: { "expectedRevision": number; "content": schemas["AboutContent"] | schemas["EcosystemContent"] | schemas["OriginalIssueContent"]; }; response: schemas["SitePage"] }
  "GET /admin/pages/{slug}/revisions": { request: unknown; response: { "items": Array<schemas["SitePage"]>; "total": number; } }
  "POST /admin/pages/{slug}/publish": { request: { "expectedRevision": number; }; response: schemas["SitePage"] }
  "POST /admin/pages/{slug}/unpublish": { request: { "expectedRevision": number; }; response: schemas["SitePage"] }
  "GET /admin/media": { request: unknown; response: { "items": Array<schemas["MediaAsset"]>; "total": number; "acceptedTypes": Array<string>; } }
  "PATCH /admin/media/{id}": { request: { "expectedRevision": number; "credit"?: string; "rights"?: string; "caption"?: string; "confirmRights"?: boolean; }; response: schemas["MediaAsset"] }
  "DELETE /admin/media/{id}": { request: { "expectedRevision": number; }; response: { "deleted": boolean; } }
  "POST /admin/media/cleanup": { request: unknown; response: { "removed": number; "remaining": number; } }
  "POST /internal/jobs/{id}/checkpoint": { request: { "leaseToken": string; "phase": "classification" | "research" | "draft" | "review"; "status": "running" | "succeeded" | "failed"; "output"?: { [key: string]: unknown; }; "error"?: string; "usage"?: Array<{ "reservationId": string; "costUsd": number; "inputTokens": number; "outputTokens": number; }>; }; response: { "ok": boolean; "job": schemas["Job"]; } }
  "GET /admin/articles/{id}/research": { request: unknown; response: { "items": Array<schemas["Job"]>; } }
  "GET /admin/bank/similar": { request: unknown; response: { "items": Array<schemas["StorySimilarity"]>; } }
  "POST /admin/bank/{id}/merge": { request: { "otherId": string; "expectedUpdatedAt": string; "otherExpectedUpdatedAt": string; }; response: { [key: string]: unknown; } }
  "POST /admin/bank/merges/{id}/undo": { request: unknown; response: schemas["StoryMerge"] }
  "GET /admin/bank/merges": { request: unknown; response: { "items": Array<schemas["StoryMerge"]>; } }
  "GET /admin/bank/{id}/evidence": { request: unknown; response: schemas["StoryEvidence"] }
  "POST /admin/bank/merges/{id}/split": { request: { "expectedLeftVersion": string; "expectedRightVersion": string; }; response: schemas["StoryMerge"] }
  "GET /api/creata/v1/admin/postiz/channels": { request: unknown; response: schemas["PostizChannels"] }
  "GET /api/creata/v1/admin/articles/{id}/postiz": { request: unknown; response: { "items": Array<schemas["PostizDelivery"]>; } }
  "GET /api/creata/v1/admin/articles/{id}/postiz/preview": { request: unknown; response: schemas["PostizPreview"] }
  "GET /api/creata/v1/account/launch-submissions": { request: unknown; response: { "items": Array<schemas["LaunchSubmission"]>; "total": number; } }
  "POST /api/creata/v1/account/launch-submissions": { request: schemas["LaunchSubmissionCreate"]; response: schemas["LaunchSubmission"] }
  "PATCH /api/creata/v1/account/launch-submissions/{id}": { request: schemas["LaunchSubmissionUpdate"]; response: schemas["LaunchSubmission"] }
  "POST /api/creata/v1/account/launch-submissions/{id}/submit": { request: schemas["LaunchSubmissionSubmit"]; response: schemas["LaunchSubmission"] }
  "GET /api/creata/v1/admin/launch-submissions": { request: unknown; response: { "items": Array<schemas["LaunchSubmission"]>; "total": number; } }
  "POST /api/creata/v1/admin/launch-submissions/{id}/review": { request: schemas["LaunchSubmissionReview"]; response: schemas["LaunchSubmission"] }
}
