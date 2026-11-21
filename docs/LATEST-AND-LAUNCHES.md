# Latest and Creata launches

User-approved addition, 14 November 2026.

## Navigation and Latest

- Keep the existing public Shell, header, primary navigation, article cards, themes and Skiper TextRoll. Extend their existing owners.
- The news destination is `/:locale/latest`, labelled **Latest**. Preserve old `/:locale/world` links through a redirect, including filters.
- Open the existing native dialog as a wide top panel: short utility row and search, then visible columns for topics, regions and sections; project/account links and theme/language controls below. No right-side drawer or nested topic accordions.
- Latest uses editorial rows: a heading/link on the left, up to three existing article cards on the right. Mobile stacks these rows.
- Latest user correction: remove Clubs and Volunteer teams for now, including their empty placeholder rows. Startups remains a group. Merge overlapping editorial topics into Technology, Science & education, Creative industries, and Impact. Do not treat internal Creata departments as public topics. Only render populated rows; never invent material to fill a group.
- `/:locale/latest?section=recent` is a collection view. Reuse the archived ClubProfileScreen cover/identity/content structure and existing ArticleCard, with a large cover, overlapping introduction and chronological stories; the main Latest route retains thematic rows. This does not restore club listings.
- Further user correction: CollectionScreen is the default for every opened topic, category, region, ecosystem and author collection. Use each collection's own title, description and first available cover. Preserve author biography/avatar and existing non-Latest filters through composition; no topic-specific replacement layouts.
- Animate menu entry/exit and staggered links using the existing Motion dependency, with the free Skiper TextRoll retained. Direction references: https://skiper-ui.com/v1/skiper38 and https://skiper-ui.com/v1/skiper58. Pro source is not bundled. Honor reduced motion, Escape and focus restoration.
- Reuse published articles and their category/tag metadata. Preserve category and region deep links; keep nonmatching and unavailable states explicit. Latest has no visible filters, dropdowns or topic tabs (explicit user correction).
- Add a violet Most popular carousel using the existing useCarousel and CarouselActions. Rank public, translated snapshots by accepted view events in the last seven days through `/api/creata/v1/public/popular`. The locale layout loads this feed once and Shell renders the sole PopularStories instance on every localized page. Individual pages never import, duplicate or fetch it separately. Hide when no ranked stories are available; no fabricated ranks or fallback fixtures.
- Replace the decorative arrow in Most popular with the existing CreataBrand and an original dimensional graphic derived from its three-bar ≡ mark. Its shared CreataEmblem adapts the free Skiper52 viewport entry and hover expansion: a staggered three-strip reveal, spring separation on fine-pointer hover and a static reduced-motion variant. Hover motion is disabled on touch. It remains decorative and creates no extra tab stop.

## Creata launches

Founders submit a product with its name, pitch, website, screenshots, category, stage, team and optional public GitHub repository. They can edit a draft, submit it, respond to review, choose a launch date and see their launches. Staff moderates before public display. Editing approved content returns it to review.

Flow: draft → review → changes requested → review → scheduled / published; rejected is separate. Publication requires an editor, chief editor or administrator. Hermes never publishes.

Public product pages and the launch feed include community votes, a website button and an optional GitHub button. One account has one vote per product; the founder cannot vote for their own product. Repository stars remain separate from Creata votes. Visitors star with their own GitHub account. Star counts come from GitHub, with collection time and period snapshots retained internally; unavailable counts are never replaced by invented numbers. A change in stars is a change during the launch period, not a claim that Creata caused every star.

Minimum initial slice: submission and review, founder/staff workspace, public feed/profile, votes and GitHub count. The weekly view considers launches in the selected period, not lifetime vote totals for all products. Account sessions and CSRF checks use the existing API. Nonpublic submissions are visible only to their owner and authorized moderators.

Next slices: richer logo/demo uploads, verified team ownership, visit/click analytics and weekly awards, editorial launch announcements and digest inclusion. The agreed end state connects launches to the newsroom and Telegram; no direct founder broadcast is allowed. Comments and social profiles remain separate future work.

## Validation

Check dialog focus, Escape, outside dismissal, route dismissal, search and topic/region navigation. Exercise Latest at 360, 768 and 1440 pixels, reduced motion, both themes and RU/kk/en. Verify source category aliases, no repeated sections, no unapproved content, and old URLs retaining their filters. Launch checks cover owner/moderator permissions, state transitions, revision conflicts, one vote, optional repositories and GitHub failure handling.
