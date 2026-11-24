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

Available in the connected site and editorial app. A signed-in founder can open **Submit a launch** from the catalog, save a draft in **My launch submissions**, and send it for review. Editors review it under **Launches awaiting review** in the startup launches section. They can publish the submitted revision or return it with feedback. The founder can revise, resubmit, and open the public card from their profile once it is published.

The first release supports an immediate publication decision, with the option to request changes before publication. The launch date is shown on the card.

### Planned extensions

Extend submissions with screenshots, stage and an optional public GitHub repository. Allow edits to approved founder submissions through a further editorial review.

Flow: draft → review → changes requested → review → scheduled / published; rejected is separate. Publication requires an editor, chief editor or administrator. Hermes never publishes.

Public product pages and the launch feed include community votes, a website button and an optional GitHub button. One account has one vote per product; the founder cannot vote for their own product. Repository stars remain separate from Creata votes. Visitors star with their own GitHub account. Star counts come from GitHub, with collection time and period snapshots retained internally; unavailable counts are never replaced by invented numbers. A change in stars is a change during the launch period, not a claim that Creata caused every star.

The planned weekly view considers launches in the selected period and the votes they received during it. Account sessions and CSRF checks use the existing API. Nonpublic submissions are visible only to their owner and authorized moderators.

Next slices include richer logo/demo uploads, verified team ownership, visit/click analytics and weekly awards, editorial launch announcements and digest inclusion. The agreed end state connects launches to the newsroom and Telegram through editorial publication.

## Validation

Check dialog focus, Escape, outside dismissal, route dismissal, search and topic/region navigation. Exercise Latest at 360, 768 and 1440 pixels, reduced motion, both themes and RU/kk/en. Verify source category aliases, no repeated sections, no unapproved content, and old URLs retaining their filters. Launch checks cover owner/moderator permissions, state transitions, revision conflicts, one vote, optional repositories and GitHub failure handling.
