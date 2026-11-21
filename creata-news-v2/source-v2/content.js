/* Контент v2: RU + EN для всего публичного. Собирается в site2/assets/data.js скриптом build-data.js */
const P = (f) => "assets/photos/" + f + ".jpg";

exports.verticalsEn = {
  ai: "AI & data", startups: "Startups & VC", stem: "Science & STEM", robotics: "Robotics & hardware", space: "Space",
  climate: "Climate & impact", health: "HealthTech & bio", edtech: "EdTech & education", fintech: "FinTech",
  cities: "Mobility & cities", agri: "AgriTech & food", creative: "Creative industries",
  "eco-group": "Creata Group", "eco-transfornation": "TransforNATION", "eco-lab": "Creata Lab", "eco-games": "Creata Games",
  "eco-ciaq": "CIAQ & GCA", "eco-ventures": "Creata Ventures", "eco-team": "Creata Team"
};

exports.authorsEn = {
  amina: { name: "Amina", role: "Editor-in-chief", bio: "Signs off the daily plan, gives final approval to Creata Ecosystem stories and sets the quality bar." },
  merey: { name: "Merey", role: "Managing editor", bio: "Runs the daily board and sends a weekly metrics report." },
  tamiris: { name: "Tamiris", role: "Editor: creative industries & gaming", bio: "Reports from events and edits stories in her vertical." },
  zarina: { name: "Zarina", role: "Editor: STEM & EdTech", bio: "Puts together the weekly “Week at Creata” roundup." },
  saya: { name: "Saya", role: "Impact & interviews", bio: "Interviews founders and collects photo consent from the people in our stories." },
  elnura: { name: "Elnura", role: "Opportunities moderator", bio: "Checks grants, contests, internships and deadlines." },
  anel: { name: "Anel", role: "Living photos & video", bio: "Finishes AI animation and shoots video for social media." },
  alisher: { name: "Alisher", role: "Postiz & social stats", bio: "Schedules story versions in Postiz and tracks reach." },
  veta: { name: "Veta", role: "Templates & image style", bio: "Keeps the visual style and social templates consistent." },
  aruzhan: { name: "Aruzhan", role: "Long reads", bio: "Writes one or two big features a week." }
};

exports.authorsExtra = {
  amina: { cover: P("c39"), followers: 1240, achievements: ["first-issue", "editor-100", "streak-30"] },
  merey: { cover: P("c26"), followers: 860, achievements: ["board-master", "streak-30"] },
  tamiris: { cover: P("c10"), followers: 1530, achievements: ["reporter", "first-issue", "photo-100"] },
  zarina: { cover: P("c105"), followers: 710, achievements: ["weekly-10"] },
  saya: { cover: P("auliekol-1"), followers: 930, achievements: ["interviews-20", "consent"] },
  aruzhan: { cover: P("akku-2"), followers: 1105, achievements: ["longread", "reporter"] }
};

exports.opportunitiesEn = {
  "o-designthon": { type: "Contest", title: "Designthon: TransforNATION design marathon", forWhom: "Students and young designers", country: "Kazakhstan", conditions: "Teams of up to 4, offline final." },
  "o-creative-start": { type: "Programme", title: "Creative Start: your first project in the creative industries", forWhom: "High-school and university students", country: "Kazakhstan", conditions: "Online application, mentors from the industry." },
  "o-gamejam": { type: "Hackathon", title: "Creata Games jam: a game in 48 hours", forWhom: "Developers, artists, game designers", country: "Online", conditions: "Solo or as a team." },
  "o-academy": { type: "Internship", title: "Intern at Creata Academy", forWhom: "EdTech and pedagogy students", country: "Almaty / remote", conditions: "8–11 hours a week." },
  "o-stem-week": { type: "Grant", title: "Invent Future Global: a grant for a STEM project", forWhom: "School and student teams", country: "Worldwide", conditions: "A prototype and a 2-minute video pitch." },
  "o-mamapro": { type: "Hackathon", title: "“Mama Pro” hackathon", forWhom: "Mom entrepreneurs and their teams", country: "Kazakhstan", conditions: "Awaiting moderator review." },
  "o-unknown": { type: "Grant", title: "$10,000 grant for an idea, no strings attached", forWhom: "Everyone", country: "—", conditions: "Came in through the form. No official link." }
};
exports.opportunityPhotos = { "o-designthon": P("c48"), "o-creative-start": P("c113"), "o-gamejam": P("c28"), "o-academy": P("c105"), "o-stem-week": P("c112"), "o-mamapro": P("c76"), "o-unknown": P("c74") };

/* ---------- Мир: переводы и фото для четырёх пересказов ---------- */
exports.world = {
  "a-vahta": {
    photo: { url: P("vahta-1"), credit: "Фото: Карлыгаш Нуржан / Forbes.kz", credit_en: "Photo: Karlygash Nurzhan / Forbes.kz" },
    photos: [
      { src: P("vahta-3"), cap: "", cap_en: "" },
      { src: P("vahta-4"), cap: "«Каспий нефть»", cap_en: "Caspiy Neft" },
      { src: P("vahta-5"), cap: "Габит Аюфкалиев, начальник месторождения", cap_en: "Gabit Ayufkaliyev, head of the field" },
      { src: P("vahta-6"), cap: "", cap_en: "" }
    ],
    en: {
      place: "Kazakhstan · Atyrau region",
      tags: ["energy", "jobs", "reportage"],
      tldr: "Airankol runs two shifts of about 110 people each, and a rotation lasts 14 days. The pre-shift medical check is automated, meals are free, and repairing a single well takes five to six hours just to lower the pipes.",
      why: "You read about engineering careers and STEM",
      blocks: [
        { t: "stats", items: [{ n: "≈110", l: "people per shift" }, { n: "14", l: "days per rotation" }, { n: "1,400 m", l: "depth the pipes go down" }] },
        { t: "p", text: "Where is this? The Airankol field in Atyrau region, operated by Caspiy Neft. Work runs in two shifts of about 110 people each. A rotation used to last 15 days; now it is 14." },
        { t: "p", text: "How does the day start? The kitchen opens at 3 a.m. and the working day begins at 7:30. Workers live in a camp of five dormitories; one of them has 20 rooms. Meals are free." },
        { t: "quote", text: "Discipline and organisation are the key principles I follow.", who: "Gabit Ayufkaliyev, head of the field" },
        { t: "h", text: "The medical check is your pass to the shift" },
        { t: "p", text: "Before each shift a doctor checks blood pressure and pulse and runs a breathalyser test. Part of this is already automated: an electronic medical check system records the results without paper logs." },
        { t: "quote", text: "People who have been drinking are not allowed to work at the field.", who: "Zhumanali Saidov, field doctor" },
        { t: "h", text: "What happens at the well", lvl: "deep" },
        { t: "p", lvl: "deep", text: "The Forbes.kz reporter arrived during a major underground repair of well No. 2. Pipes are lowered 1,400 metres down, and that stage alone takes five to six hours. An operator in the control centre watches the process in real time." },
        { t: "p", lvl: "deep", text: "Before the oil goes into the KazTransOil pipeline, a lab checks it for sulphur, viscosity, salt and density. Twice a year, in March and September, the field holds sports days and quiz games." },
        { t: "facts", title: "In short", items: ["Some staff have up to 17 years of experience", "Sports days twice a year: March and September", "Oil is tested for sulphur, viscosity, salt and density"] }
      ]
    }
  },
  "a-akku": {
    photo: { url: P("akku-1"), credit: "Фото: Zakon.kz", credit_en: "Photo: Zakon.kz" },
    photos: [
      { src: P("akku-2"), cap: "Люди казахской культуры: поэты, актёры, писатели, художники", cap_en: "People of Kazakh culture: poets, actors, writers, artists" },
      { src: P("akku-3"), cap: "Писательница и журналист Магира Кожахметова", cap_en: "Writer and journalist Magira Kozhakhmetova" },
      { src: P("akku-4"), cap: "Оралхан Бокей, писатель, 1943–1993", cap_en: "Oralkhan Bokei, writer, 1943–1993" },
      { src: P("akku-5"), cap: "Актриса Бақыт Жанғалиева, 1947–2017", cap_en: "Actress Bakyt Zhangaliyeva, 1947–2017" },
      { src: P("akku-6"), cap: "С народной поэтессой Казахской ССР Мариям Хакимжановой", cap_en: "With Mariyam Khakimzhanova, People's Poet of the Kazakh SSR" }
    ],
    en: {
      place: "Kazakhstan · Almaty",
      tags: ["Almaty", "urban life", "culture"],
      tldr: "Akku is a café with roughly 50 years of history next to the Writers' Union. The swans disappeared, the building burned down, and now it is being restored. Residents are asking for a light summer pavilion, a public discussion of the design and prices students can afford.",
      why: "Popular with readers in Almaty",
      blocks: [
        { t: "p", text: "What is this place? The Akku café in Almaty stood next to the Writers' Union and the Theatre and Art Institute. For about half a century poets, artists and Kazakh State University students came here. According to Askar Suleimenov, Mukagali Makatayev wrote down his thoughts in a notebook here every day." },
        { t: "quote", text: "For us this place was sacred and intimate; there was always a kind of natural beauty here.", who: "Magira Kozhakhmetova, writer" },
        { t: "timeline", title: "Half a century of Akku", items: [
          { y: "1960s", text: "An open-air terrace; coffee brewed in cezves on hot sand" },
          { y: "1969", text: "Magira Kozhakhmetova starts writing portraits of cultural figures at the Akku aesthetic club" },
          { y: "1980s", text: "Packed in the evenings with Kazakh State University students" },
          { y: "1989", text: "The swan pond is visible from the terrace" },
          { y: "Later", text: "The swans disappear and the café burns down" },
          { y: "2024", text: "Restoration begins and residents discuss the design" }
        ] },
        { t: "h", text: "What people want it to be" },
        { t: "p", text: "An architect's view: Yerzhan Suleimenov proposes a light summer pavilion, an eco-friendly building and a public discussion of the design. Yerkin Suleimenov suggests discounts before 6 p.m. for students, families and pensioners." },
        { t: "quote", text: "But then the swans disappeared, and part of the soul of this place went with them.", who: "Myrzan Kenzhebai" },
        { t: "p", lvl: "deep", text: "One detail from the memories: Akku never sold alcohol. Zauresh Kubekova recalls ordering ten skewers of shashlik here when she was pregnant. She says the ground floor of the new building will house the kitchen and storage." },
        { t: "poll", q: "What should the new Akku be?", options: ["A light summer pavilion", "A year-round café", "A space for talks and meetups"] }
      ]
    }
  },
  "a-mosques": {
    photo: { url: P("mosques-1"), credit: "Фото: BES.media", credit_en: "Photo: BES.media" },
    photos: [
      { src: P("mosques-2"), cap: "Карта Семиречья, составленная маньчжурами", cap_en: "A map of Zhetysu drawn by the Manchus" },
      { src: P("mosques-3"), cap: "Археологические раскопки в Алматы", cap_en: "Archaeological excavations in Almaty" },
      { src: P("mosques-4"), cap: "Посеребрённые медные дирхемы монетного двора Алматы, 1271–1277", cap_en: "Silver-plated copper dirhams from the Almaty mint, 1271–1277" },
      { src: P("mosques-5"), cap: "Список мусульман города Верного, 1920 год", cap_en: "A list of the Muslims of Verny, 1920" },
      { src: P("mosques-6"), cap: "Кинотеатр", cap_en: "The cinema" }
    ],
    en: {
      place: "Kazakhstan · Almaty",
      tags: ["architecture", "city history", "Almaty"],
      tldr: "Almaty has 56 mosques today. The story starts with a 1285 dirham inscribed “balad Almatu”. Verny had 12 mosques, some were demolished in the 20th century, the Central Mosque was built in the 1990s, and a complex for more than 20,000 worshippers was due to open in 2025.",
      why: "A new topic for you: architecture and city history",
      blocks: [
        { t: "p", text: "Where does it begin? With a forge. In 1978 archaeologists excavating a 9th–13th century settlement found a dirham from 1285. It is stamped “balad Almatu”, the earliest known written mention of the city." },
        { t: "timeline", title: "Almaty's mosques on a timeline", items: [
          { y: "1285", text: "A dirham inscribed “balad Almatu”" },
          { y: "19th c.", text: "Verny has 12 mosques for about 17,000 Muslims; the Fatima mosque is built in the Tatar quarter" },
          { y: "1887", text: "An earthquake partly destroys the Fatima mosque" },
          { y: "1930s", text: "The Iskhakiya mosque building becomes a theatre, then the Vostok cinema" },
          { y: "1960s", text: "The Iskhakiya building is demolished" },
          { y: "1990s", text: "The Central Mosque is built on the site of the Karatal Kazakhs' mosque; Fatima is restored" },
          { y: "2014", text: "The Baiken mosque opens on the site of a former casino" },
          { y: "2025", text: "A complex for more than 20,000 worshippers is due to open" }
        ] },
        { t: "stats", items: [{ n: "56", l: "mosques in Almaty (2024)" }, { n: "47 m", l: "height of the Central Mosque minaret" }, { n: "7,000", l: "places in the Central Mosque" }] },
        { t: "h", text: "Why there is still not enough room", lvl: "deep" },
        { t: "p", lvl: "deep", text: "On 6 December 2024 several dozen worshippers at the Central Mosque could not fit in the courtyard and stepped onto Manshuk Mametova Street, stopping traffic. The new complex, unofficially called the “Kulibayev mosque”, is designed for 13,500 m² of carpet and parking for 1,100 cars." },
        { t: "p", text: "The Central Mosque was designed by architects S. Baimagambetov, Zh. Sharapiyev and K. Zharylganov. Its dome is 20 metres across and 36 metres high." },
        { t: "quiz", q: "What was in the Iskhakiya mosque building in the 1930s?", options: ["A library", "A theatre, then a cinema", "A school"], answer: 1, explain: "First a theatre, then the Vostok cinema. The building was demolished in the 1960s." }
      ],
      corrections: [{ date: "2026-11-12", text: "We corrected the dome height: 36 metres, not 35." }]
    }
  },
  "a-hogwarts": {
    title: "Детсад-замок в Аулиеколе: где не повышают голос",
    photo: { url: P("auliekol-1"), credit: "Фото: Tengrinews.kz", credit_en: "Photo: Tengrinews.kz" },
    photos: [
      { src: P("auliekol-2"), cap: "", cap_en: "" },
      { src: P("auliekol-3"), cap: "", cap_en: "" },
      { src: P("auliekol-4"), cap: "", cap_en: "" },
      { src: P("auliekol-5"), cap: "", cap_en: "" },
      { src: P("auliekol-6"), cap: "", cap_en: "" }
    ],
    en: {
      place: "Kazakhstan · Kostanay region",
      tags: ["education", "regions", "architecture"],
      tldr: "The Altyn Besik kindergarten was built by the Halyk foundation for 1.3 billion tenge. About 200 children from three villages go there, in Kazakh and Russian groups. There are no punishments, children choose their own activities, and parents can stay in class during the first days.",
      why: "You follow EdTech & education",
      blocks: [
        { t: "stats", items: [{ n: "≈200", l: "children enrolled" }, { n: "12", l: "playgrounds" }, { n: "₸1.3 bn", l: "construction cost" }] },
        { t: "p", text: "What kind of castle? The Altyn Besik kindergarten in the village of Auliekol was built by the Halyk charity foundation. According to its head, Svetlana Birzhanova, construction cost 1.3 billion tenge and equipment another 57 million." },
        { t: "p", text: "Who goes there? About 200 children from nursery age to five, from Auliekol, Novoselovka and Amankaragai. Before, these villages only had half-day mini-centres attached to schools." },
        { t: "quote", text: "We can't talk to children about beauty and harmony when the radiators around them are broken.", who: "Svetlana Birzhanova, head of the kindergarten" },
        { t: "h", text: "The method: freedom of choice" },
        { t: "p", text: "Classes are play-based and children decide what to do. No punishments and no raised voices. To settle in, a child can bring a favourite toy, and in the first days parents can stay in class." },
        { t: "p", lvl: "deep", text: "The assembly hall has an LED screen, an interactive whiteboard and a projector, and every teacher has a computer and a printer. There is a Kazakh language room for the Russian-speaking groups. Teacher Aruna Menbekova moved back to her home village from Kostanay when the kindergarten opened; both her children go there too." },
        { t: "quote", text: "I think every region of Kazakhstan needs modern kindergartens like this.", who: "Gulzhanat, mother of three" }
      ]
    }
  }
};
exports.worldRuPatch = {
  "a-hogwarts": {
    lead: "Здание в селе Костанайской области местные называют замком. Tengrinews съездил внутрь: 200 детей, 12 площадок и методика, где ребёнок сам выбирает, чем заняться.",
    lead_en: "Locals in a Kostanay region village call the building a castle. Tengrinews went inside: 200 children, 12 playgrounds and a method where the child chooses what to do."
  }
};

/* ---------- Экосистема Creata: репортажи из номера 01 creata.news ---------- */
const SRC01 = { name: "Creata News № 01", author: "Редакция Creata News", author_en: "Creata News editors", date: "2026-10-15", url: "https://creata.news/issues/01" };
const CR = "Фото: Creata News", CR_EN = "Photo: Creata News";
const g = (n, cap, cap_en) => ({ src: P("c" + n), cap, cap_en });

exports.eco = [
  {
    id: "e-dcz", vertical: "eco-group", author: "tamiris", published: "2026-11-13T10:00:00+05:00", readMin: 4, views: 2650, saves: 188, live: true,
    place: "Казахстан · Астана", place_en: "Kazakhstan · Astana",
    title: "Digital Creative Zone: VR-арена, ретро-приставки и футпонг на крыше",
    title_en: "Digital Creative Zone: a VR arena, retro consoles and footpong on a roof",
    lead: "2 октября зона Creata Group собрала студии, косплееров и художников. Смотрим, что там происходило, по кадрам из первого номера.",
    lead_en: "On 2 October the Creata Group zone brought together studios, cosplayers and artists. Here is what happened there, frame by frame from our first issue.",
    tldr: "Digital Creative Zone прошла 2 октября. В программе — турнир FRONTLINE VR BATTLE, ретро-зона с кинескопными телевизорами, футпонг на стенде Creata Group, Аллея авторов и косплей.",
    tldr_en: "Digital Creative Zone took place on 2 October. It featured the FRONTLINE VR BATTLE tournament, a retro zone with CRT televisions, footpong at the Creata Group stand, an Artists' Alley and cosplay.",
    tags: ["события", "геймдев", "фоторепортаж"], tags_en: ["events", "gamedev", "photo essay"],
    photo: { url: P("c01"), credit: CR, credit_en: CR_EN },
    photos: [
      g("03", "Давид Туганов и Хишам Нуху, сооснователи Creata Group, на стенде группы", "David Tuganov and Hisham Nukhu, co-founders of Creata Group, at the group's stand"),
      g("05", "Награждение FRONTLINE VR BATTLE", "The FRONTLINE VR BATTLE awards"),
      g("06", "Ретро-зона: приставки и кинескопные телевизоры", "The retro zone: consoles and CRT televisions"),
      g("31", "Футпонг: поле на крыше, Байтерек и Хан Шатыр на горизонте", "Footpong: a pitch on the roof, with Baiterek and Khan Shatyr on the horizon"),
      g("32", "Гости играют в футпонг на стенде Creata Group", "Guests playing footpong at the Creata Group stand"),
      g("44", "VR-арена: матч два на два", "The VR arena: a two-on-two match"),
      g("47", "Первый раз в VR-шлеме", "First time in a VR headset"),
      g("48", "Художница на Аллее авторов", "An artist in the Artists' Alley"),
      g("50", "Робот-гид в зоне технологий", "A robot guide in the tech zone"),
      g("42", "Бэтгёрл в Digital Creative Zone", "Batgirl at Digital Creative Zone")
    ],
    blocks: [
      { t: "p", text: "Что это было? Digital Creative Zone — площадка, где 2 октября встретились игровые студии, художники и игроки. Creata Group привезла свой стенд, а сооснователи группы Давид Туганов и Хишам Нуху встречали гостей сами." },
      { t: "h", text: "Играть, а не смотреть" },
      { t: "p", text: "Главный шум шёл от VR-арены: там играли FRONTLINE VR BATTLE — командный VR-шутер, матчи два на два, а вечером наградили победителей. В ретро-зоне стояли приставки и кинескопные телевизоры." },
      { t: "p", text: "На стенде Creata Group гости играли в футпонг. Поле для промо-съёмки построили на крыше — с видом на Байтерек и Хан Шатыр." },
      { t: "facts", title: "Что ещё было", items: ["Аллея авторов: мерч, брелоки и скетчи с натуры в подарок", "Робот-гид в зоне технологий", "Косплей у стендов студий"] }
    ],
    blocks_en: [
      { t: "p", text: "What was it? Digital Creative Zone is a venue where game studios, artists and players met on 2 October. Creata Group brought its own stand, and the group's co-founders David Tuganov and Hisham Nukhu greeted guests themselves." },
      { t: "h", text: "Play, don't just watch" },
      { t: "p", text: "Most of the noise came from the VR arena: people played FRONTLINE VR BATTLE, a team VR shooter, in two-on-two matches, and the winners were awarded in the evening. The retro zone had consoles and CRT televisions." },
      { t: "p", text: "At the Creata Group stand guests played footpong. The pitch for the promo shoot was built on a roof, with a view of Baiterek and Khan Shatyr." },
      { t: "facts", title: "What else", items: ["Artists' Alley: merch, keychains and free sketches from life", "A robot guide in the tech zone", "Cosplay by the studio stands"] }
    ],
    why: "Ты интересуешься геймдевом", why_en: "You're into gamedev", actions: ["o-gamejam"]
  },
  {
    id: "e-gamesday", vertical: "eco-games", author: "tamiris", published: "2026-11-12T15:00:00+05:00", readMin: 3, views: 1980, saves: 140, live: true,
    place: "Казахстан · Астана", place_en: "Kazakhstan · Astana",
    title: "Creata Games Day: от ИИ в геймдеве до топ-10 GamePreneurs",
    title_en: "Creata Games Day: from AI in gamedev to the GamePreneurs top 10",
    lead: "Три дня в начале октября — доклады студий, панель об инвестициях и паблишинге и десять лучших команд-предпринимателей.",
    lead_en: "Three days in early October: studio talks, a panel on investment and publishing, and the ten best game-founder teams.",
    tldr: "Creata Games Day открылся 1 октября. На сцене выступали студии и эксперты, FrontlineVR рассказал об ИИ в геймдеве, 3 октября прошла панель об инвестициях и паблишинге и назвали топ-10 GamePreneurs.",
    tldr_en: "Creata Games Day opened on 1 October. Studios and experts took the stage, FrontlineVR spoke about AI in gamedev, and on 3 October there was a panel on investment and publishing and the GamePreneurs top 10 was announced.",
    tags: ["геймдев", "инвестиции", "события"], tags_en: ["gamedev", "investment", "events"],
    photo: { url: P("c10"), credit: CR, credit_en: CR_EN },
    photos: [
      g("04", "Creata Games Day, 1 октября", "Creata Games Day, 1 October"),
      g("14", "Gilbert Antaki на сцене", "Gilbert Antaki on stage"),
      g("15", "Доклад FrontlineVR «ИИ в геймдеве»", "FrontlineVR's talk “AI in gamedev”"),
      g("13", "Владимир Белоусов, OGames", "Vladimir Belousov, OGames"),
      g("16", "Сергей Сыскин, Lipsar Studio", "Sergei Syskin, Lipsar Studio"),
      g("18", "Малика Игибаева, SHU", "Malika Igibayeva, SHU"),
      g("19", "Панель об инвестициях и паблишинге в геймдеве, 3 октября", "Panel on investment and publishing in gamedev, 3 October"),
      g("23", "Топ-10 GamePreneurs, 3 октября", "GamePreneurs top 10, 3 October")
    ],
    blocks: [
      { t: "p", text: "Когда? Creata Games Day открылся 1 октября и продолжался до 3-го. Это день, когда студии, эксперты и начинающие разработчики оказываются в одном зале." },
      { t: "p", text: "О чём говорили? FrontlineVR сделал доклад «ИИ в геймдеве». Выступали Gilbert Antaki, Владимир Белоусов (OGames), Сергей Сыскин (Lipsar Studio), Малика Игибаева (SHU) и Назым Жұмабаева (GameLab KBTU)." },
      { t: "h", text: "Деньги и паблишинг" },
      { t: "p", text: "3 октября прошла панель об инвестициях и паблишинге в геймдеве. В тот же день объявили топ-10 GamePreneurs — команд, которые строят игровой бизнес." },
      { t: "facts", title: "Что сделать", items: ["Посмотреть ближайший геймджем Creata Games", "Подписаться на вертикаль Creata Games"] }
    ],
    blocks_en: [
      { t: "p", text: "When? Creata Games Day opened on 1 October and ran until the 3rd. It is the day when studios, experts and new developers end up in the same room." },
      { t: "p", text: "What was discussed? FrontlineVR gave a talk on “AI in gamedev”. Speakers included Gilbert Antaki, Vladimir Belousov (OGames), Sergei Syskin (Lipsar Studio), Malika Igibayeva (SHU) and Nazym Zhumabayeva (GameLab KBTU)." },
      { t: "h", text: "Money and publishing" },
      { t: "p", text: "On 3 October there was a panel on investment and publishing in gamedev. The same day the GamePreneurs top 10 was announced: teams building game businesses." },
      { t: "facts", title: "What to do", items: ["Check the next Creata Games jam", "Follow the Creata Games vertical"] }
    ],
    why: "Ты интересуешься геймдевом", why_en: "You're into gamedev", actions: ["o-gamejam"]
  },
  {
    id: "e-genai-jam", vertical: "eco-games", author: "alisher", published: "2026-11-11T12:00:00+05:00", readMin: 2, views: 1430, saves: 96, live: false,
    place: "Казахстан", place_en: "Kazakhstan",
    title: "Финал GenAI Jam: Beat Grid, ProjectSiren и Candle Crypt",
    title_en: "GenAI Jam final: Beat Grid, ProjectSiren and Candle Crypt",
    lead: "3 октября назвали трёх победителей геймджема, где игры делали вместе с генеративным ИИ.",
    lead_en: "On 3 October the three winners of a game jam built with generative AI were announced.",
    tldr: "Первое место — Beat Grid команды VoidChub, второе — ProjectSiren от SquareTeam, третье — Candle Crypt от Qma team.",
    tldr_en: "First place went to Beat Grid by VoidChub, second to ProjectSiren by SquareTeam, third to Candle Crypt by Qma team.",
    tags: ["геймджем", "генеративный ИИ"], tags_en: ["game jam", "generative AI"],
    photo: { url: P("c27"), credit: CR, credit_en: CR_EN },
    photos: [
      g("28", "Финал GenAI Jam, 3 октября", "GenAI Jam final, 3 October"),
      g("29", "ProjectSiren — SquareTeam, второе место", "ProjectSiren by SquareTeam, second place"),
      g("30", "Candle Crypt — Qma team, третье место", "Candle Crypt by Qma team, third place")
    ],
    blocks: [
      { t: "stats", items: [{ n: "1", l: "Beat Grid · VoidChub" }, { n: "2", l: "ProjectSiren · SquareTeam" }, { n: "3", l: "Candle Crypt · Qma team" }] },
      { t: "p", text: "Что за джем? GenAI Jam — геймджем, где команды делают игры с помощью генеративного ИИ. Финал прошёл 3 октября." },
      { t: "p", text: "Кто победил? Первое место у Beat Grid — игры команды VoidChub. Второе — ProjectSiren от SquareTeam, третье — Candle Crypt от Qma team." }
    ],
    blocks_en: [
      { t: "stats", items: [{ n: "1", l: "Beat Grid · VoidChub" }, { n: "2", l: "ProjectSiren · SquareTeam" }, { n: "3", l: "Candle Crypt · Qma team" }] },
      { t: "p", text: "What jam? GenAI Jam is a game jam where teams build games with generative AI. The final took place on 3 October." },
      { t: "p", text: "Who won? First place went to Beat Grid, a game by VoidChub. Second was ProjectSiren by SquareTeam, third was Candle Crypt by Qma team." }
    ],
    why: "Ты интересуешься AI", why_en: "You're into AI", actions: ["o-gamejam"]
  },
  {
    id: "e-sft", vertical: "eco-transfornation", author: "zarina", published: "2026-11-10T09:00:00+05:00", readMin: 3, views: 2210, saves: 175, live: true,
    place: "Казахстан · Центральная Азия", place_en: "Kazakhstan · Central Asia",
    title: "Solve for Tomorrow 2026: от буткемпа в июне до финала в августе",
    title_en: "Solve for Tomorrow 2026: from a June bootcamp to an August final",
    lead: "Школьные и студенческие команды прошли очный буткемп и вышли в финал Samsung Solve for Tomorrow. Endosim и SAMRUQ.AI — в призёрах.",
    lead_en: "School and student teams went through an in-person bootcamp and reached the Samsung Solve for Tomorrow final. Endosim and SAMRUQ.AI were among the winners.",
    tldr: "Очный буткемп прошёл 10 июня, финал — 25 августа. Endosim занял второе место, SAMRUQ.AI — третье, команда OquBot показала демо робота.",
    tldr_en: "The in-person bootcamp took place on 10 June and the final on 25 August. Endosim came second, SAMRUQ.AI third, and the OquBot team demoed its robot.",
    tags: ["STEM", "конкурс", "школьники"], tags_en: ["STEM", "contest", "students"],
    photo: { url: P("c106"), credit: CR, credit_en: CR_EN },
    photos: [
      g("112", "Участники очного буткемпа, 10 июня 2026", "In-person bootcamp participants, 10 June 2026"),
      g("107", "Участники интенсива в Душанбе", "Participants of the intensive in Dushanbe"),
      g("02", "Финал, 25 августа", "The final, 25 August"),
      g("108", "Endosim — второе место", "Endosim, second place"),
      g("109", "Жюри у стенда Endosim", "The jury at the Endosim stand"),
      g("110", "Демо робота OquBot", "OquBot robot demo"),
      g("111", "SAMRUQ.AI — третье место", "SAMRUQ.AI, third place")
    ],
    blocks: [
      { t: "timeline", title: "Сезон 2026", items: [{ y: "10 июня", text: "Очный буткемп для команд" }, { y: "Лето", text: "Интенсив в Душанбе" }, { y: "25 августа", text: "Финал Solve for Tomorrow" }] },
      { t: "p", text: "Что это? Samsung Solve for Tomorrow — конкурс, где команды решают реальные задачи с помощью технологий. В 2026 году финалисты прошли очный буткемп 10 июня, а интенсив прошёл и в Душанбе." },
      { t: "p", text: "Кто в призёрах? Endosim занял второе место, SAMRUQ.AI — третье. OquBot показал демо робота." },
      { t: "facts", title: "Что сделать", items: ["Собрать команду и посмотреть грант Invent Future Global", "Подписаться на TransforNATION"] }
    ],
    blocks_en: [
      { t: "timeline", title: "The 2026 season", items: [{ y: "10 June", text: "In-person bootcamp for the teams" }, { y: "Summer", text: "An intensive in Dushanbe" }, { y: "25 August", text: "Solve for Tomorrow final" }] },
      { t: "p", text: "What is it? Samsung Solve for Tomorrow is a contest where teams solve real problems with technology. In 2026 the finalists went through an in-person bootcamp on 10 June, and an intensive was also held in Dushanbe." },
      { t: "p", text: "Who placed? Endosim came second, SAMRUQ.AI third, and OquBot demoed its robot." },
      { t: "facts", title: "What to do", items: ["Form a team and look at the Invent Future Global grant", "Follow TransforNATION"] }
    ],
    why: "Ты подписан на STEM", why_en: "You follow STEM", actions: ["o-stem-week"]
  },
  {
    id: "e-creative-start", vertical: "eco-transfornation", author: "saya", published: "2026-11-08T11:00:00+05:00", readMin: 3, views: 1720, saves: 132, live: false,
    place: "Казахстан", place_en: "Kazakhstan",
    title: "Creative Start: игра про экологию, комикс о фишинге и музыка",
    title_en: "Creative Start: an eco game, a comic about phishing, and music",
    lead: "6 июня назвали победителей Creative Start в трёх номинациях. Комиксы финалистов напечатали.",
    lead_en: "On 6 June the Creative Start winners were named in three categories. The finalists' comics were printed.",
    tldr: "Победители Creative Start: EcoShift (геймдев), «Консенсус» (комиксы) и Jetim Dreams (музыка). Комиксы «Голос леса» и комикс о защите от фишинга на казахском вышли на бумаге.",
    tldr_en: "Creative Start winners: EcoShift (gamedev), Konsensus (comics) and Jetim Dreams (music). The comics “Voice of the Forest” and a Kazakh-language comic about phishing protection were printed.",
    tags: ["молодёжь", "комиксы", "музыка"], tags_en: ["youth", "comics", "music"],
    photo: { url: P("c113"), credit: CR, credit_en: CR_EN },
    photos: [
      g("115", "Creative Start, 6 июня", "Creative Start, 6 June"),
      g("116", "EcoShift — первое место, геймдев", "EcoShift, first place, gamedev"),
      g("117", "«Консенсус» — первое место, комиксы", "Konsensus, first place, comics"),
      g("118", "Jetim Dreams — первое место, музыка", "Jetim Dreams, first place, music"),
      g("114", "Напечатанные комиксы финалистов: «Голос леса» и комикс о защите от фишинга на казахском", "The finalists' printed comics: “Voice of the Forest” and a Kazakh-language comic about phishing protection")
    ],
    blocks: [
      { t: "p", text: "Что за программа? Creative Start — первый проект в креативных индустриях для школьников и студентов. Финал прошёл 6 июня." },
      { t: "stats", items: [{ n: "EcoShift", l: "1 место · геймдев" }, { n: "Консенсус", l: "1 место · комиксы" }, { n: "Jetim Dreams", l: "1 место · музыка" }] },
      { t: "p", text: "Что осталось после финала? Комиксы команд-финалистов напечатали: «Голос леса» и комикс о защите от фишинга на казахском языке." }
    ],
    blocks_en: [
      { t: "p", text: "What programme? Creative Start is a first project in the creative industries for school and university students. The final took place on 6 June." },
      { t: "stats", items: [{ n: "EcoShift", l: "1st · gamedev" }, { n: "Konsensus", l: "1st · comics" }, { n: "Jetim Dreams", l: "1st · music" }] },
      { t: "p", text: "What came out of it? The finalists' comics were printed: “Voice of the Forest” and a comic about phishing protection in Kazakh." }
    ],
    why: "Подходит для твоей стадии: исследую", why_en: "Fits your stage: exploring", actions: ["o-creative-start"]
  },
  {
    id: "e-mytona", vertical: "eco-ciaq", author: "veta", published: "2026-11-07T13:00:00+05:00", readMin: 2, views: 1180, saves: 88, live: false,
    place: "Казахстан", place_en: "Kazakhstan",
    title: "Myths and Tales 2026: юрта, Самрук и бешбармак в игре Mytona",
    title_en: "Myths and Tales 2026: a yurt, Samruk and beshbarmak in a Mytona game",
    lead: "Mytona подвела итоги конкурса Myths and Tales, а в Seekers Notes появилась локация, вдохновлённая Казахстаном.",
    lead_en: "Mytona announced the results of its Myths and Tales contest, and Seekers Notes got a location inspired by Kazakhstan.",
    tldr: "Mytona — партнёр Digital Creative Zone. Конкурс Myths and Tales 2026 прошёл в номинациях 2D и 3D. В Seekers Notes есть локация с юртой, священной лошадью, Самруком и бешбармаком.",
    tldr_en: "Mytona is a Digital Creative Zone partner. Myths and Tales 2026 ran in 2D and 3D categories. Seekers Notes has a location with a yurt, a sacred horse, Samruk and beshbarmak.",
    tags: ["игры", "культура", "конкурс"], tags_en: ["games", "culture", "contest"],
    photo: { url: P("c58"), credit: CR, credit_en: CR_EN },
    photos: [
      g("64", "Локация Seekers Notes, вдохновлённая Казахстаном", "A Seekers Notes location inspired by Kazakhstan"),
      g("61", "Итоги Myths and Tales 2026", "Myths and Tales 2026 results"),
      g("62", "Myths and Tales, 2D", "Myths and Tales, 2D"),
      g("63", "Myths and Tales, 3D", "Myths and Tales, 3D"),
      g("60", "Global Game Jam KZ 2026: «Выбор Mytona» — команда Prism", "Global Game Jam KZ 2026: Mytona's Choice went to team Prism"),
      g("07", "Стенд Mytona в Digital Creative Zone", "Mytona's stand at Digital Creative Zone")
    ],
    blocks: [
      { t: "p", text: "Кто это? Mytona — игровая компания и партнёр Digital Creative Zone. На главном фото — работа победителя конкурса Myths and Tales 2026 в номинации 3D." },
      { t: "p", text: "Что ещё? В игре Seekers Notes появилась локация, вдохновлённая Казахстаном: юрта, священная лошадь, Самрук и бешбармак. На Global Game Jam KZ 2026 «Выбор Mytona» получила команда Prism." }
    ],
    blocks_en: [
      { t: "p", text: "Who is this? Mytona is a game company and a Digital Creative Zone partner. The main photo shows the winning entry of Myths and Tales 2026 in the 3D category." },
      { t: "p", text: "What else? Seekers Notes has a new location inspired by Kazakhstan: a yurt, a sacred horse, Samruk and beshbarmak. At Global Game Jam KZ 2026, Mytona's Choice went to team Prism." }
    ],
    why: "Креативные индустрии", why_en: "Creative industries", actions: ["o-designthon"]
  },
  {
    id: "e-ciaq", vertical: "eco-ciaq", author: "merey", published: "2026-11-06T10:00:00+05:00", readMin: 3, views: 990, saves: 64, live: false,
    place: "Казахстан · Астана", place_en: "Kazakhstan · Astana",
    title: "CIAQ × МФЦА и Курултай: креативная экономика на повестке",
    title_en: "CIAQ × AIFC and the Kurultai: the creative economy on the agenda",
    lead: "Сентябрь у CIAQ — панель с МФЦА 10-го, Курултай 25-го и AI & Digital Bridge 2026 в Астане.",
    lead_en: "September for CIAQ: a panel with the AIFC on the 10th, the Kurultai on the 25th and AI & Digital Bridge 2026 in Astana.",
    tldr: "10 сентября CIAQ провёл панель с МФЦА, 25 сентября прошёл Курултай. Также в номере 01: AI & Digital Bridge 2026 в Астане и платформа QazaqCreative.",
    tldr_en: "On 10 September CIAQ held a panel with the AIFC, and the Kurultai took place on 25 September. Also in issue 01: AI & Digital Bridge 2026 in Astana and the QazaqCreative platform.",
    tags: ["креативная экономика", "политика"], tags_en: ["creative economy", "policy"],
    photo: { url: P("c69"), credit: CR, credit_en: CR_EN },
    photos: [
      g("70", "Участники панели, 10 сентября", "Panel participants, 10 September"),
      g("68", "Курултай, 25 сентября", "The Kurultai, 25 September"),
      g("73", "AI & Digital Bridge 2026, Астана", "AI & Digital Bridge 2026, Astana"),
      g("74", "Платформа QazaqCreative", "The QazaqCreative platform"),
      g("88", "Открытие выставки «Батыр Аманаты», SPIRAL, Алматы, 24 апреля (фото @ciaq.kz)", "Opening of the “Batyr Amanaty” exhibition, SPIRAL, Almaty, 24 April (photo @ciaq.kz)")
    ],
    blocks: [
      { t: "p", text: "Что это? CIAQ — альянс креативных индустрий в экосистеме Creata. 10 сентября он провёл панель вместе с МФЦА." },
      { t: "timeline", title: "Календарь", items: [{ y: "24 апреля", text: "Выставка «Батыр Аманаты», SPIRAL, Алматы" }, { y: "10 сентября", text: "Панель CIAQ × МФЦА" }, { y: "25 сентября", text: "Курултай" }] },
      { t: "p", text: "Где следить? В номере 01 — также AI & Digital Bridge 2026 в Астане и платформа QazaqCreative." }
    ],
    blocks_en: [
      { t: "p", text: "What is it? CIAQ is the creative industries alliance in the Creata ecosystem. On 10 September it held a panel together with the AIFC." },
      { t: "timeline", title: "Calendar", items: [{ y: "24 April", text: "“Batyr Amanaty” exhibition, SPIRAL, Almaty" }, { y: "10 September", text: "CIAQ × AIFC panel" }, { y: "25 September", text: "The Kurultai" }] },
      { t: "p", text: "Where to follow? Issue 01 also covers AI & Digital Bridge 2026 in Astana and the QazaqCreative platform." }
    ],
    why: "Креативные индустрии", why_en: "Creative industries", actions: []
  },
  {
    id: "e-efm", vertical: "eco-lab", author: "anel", published: "2026-11-05T16:00:00+05:00", readMin: 2, views: 870, saves: 52, live: false,
    place: "Казахстан", place_en: "Kazakhstan",
    title: "ИИ и кино: сессии EFM, Impact-кино и мультфильмы для детей",
    title_en: "AI and film: EFM sessions, impact cinema and cartoons for kids",
    lead: "В конце сентября Давид Туганов говорил об ИИ в кино, а 3 октября прошла панель об ИИ-мультфильмах и детском кино.",
    lead_en: "In late September David Tuganov spoke about AI in film, and on 3 October there was a panel on AI cartoons and children's cinema.",
    tldr: "29 сентября — сессия «Artificial Intelligence & Film» с Давидом Тугановым и Даяной Уакпаевой, 30 сентября — сессия «Impact кино», 3 октября — панель «Цифровое потребление: ИИ-мультфильмы и детское кино».",
    tldr_en: "29 September: the “Artificial Intelligence & Film” session with David Tuganov and Dayana Uakpayeva. 30 September: the “Impact cinema” session. 3 October: the panel “Digital consumption: AI cartoons and children's film”.",
    tags: ["кино", "AI"], tags_en: ["film", "AI"],
    photo: { url: P("c81"), credit: CR, credit_en: CR_EN },
    photos: [
      g("119", "EFM, 29 сентября", "EFM, 29 September"),
      g("82", "Сессия «Impact кино», 30 сентября", "The “Impact cinema” session, 30 September"),
      g("75", "Панель «Цифровое потребление: ИИ-мультфильмы и детское кино», 3 октября", "Panel “Digital consumption: AI cartoons and children's film”, 3 October"),
      g("76", "Воркшоп о гибридном производстве, Vision Hall", "A workshop on hybrid production, Vision Hall")
    ],
    blocks: [
      { t: "timeline", title: "Три сессии", items: [{ y: "29 сентября", text: "«Artificial Intelligence & Film»: Давид Туганов и Даяна Уакпаева" }, { y: "30 сентября", text: "Сессия «Impact кино»" }, { y: "3 октября", text: "Панель об ИИ-мультфильмах и детском кино" }] },
      { t: "p", text: "Ещё одна точка — воркшоп о гибридном производстве в Vision Hall." }
    ],
    blocks_en: [
      { t: "timeline", title: "Three sessions", items: [{ y: "29 September", text: "“Artificial Intelligence & Film”: David Tuganov and Dayana Uakpayeva" }, { y: "30 September", text: "The “Impact cinema” session" }, { y: "3 October", text: "Panel on AI cartoons and children's film" }] },
      { t: "p", text: "One more stop: a workshop on hybrid production at Vision Hall." }
    ],
    why: "Ты интересуешься AI", why_en: "You're into AI", actions: []
  },
  {
    id: "e-ventures", vertical: "eco-ventures", author: "merey", published: "2026-11-04T10:00:00+05:00", readMin: 2, views: 1340, saves: 101, live: false,
    place: "Казахстан", place_en: "Kazakhstan",
    title: "Creata Ventures × BytePlus и визит в Qazaq AI Research University",
    title_en: "Creata Ventures × BytePlus and a visit to Qazaq AI Research University",
    lead: "Два кадра из номера 01 о том, с кем Creata строит AI-направление.",
    lead_en: "Two frames from issue 01 about who Creata is building its AI track with.",
    tldr: "В номере 01 — встреча Creata Ventures и BytePlus и визит Давида Туганова в Qazaq AI Research University, где ректор — Хишам Нуху.",
    tldr_en: "Issue 01 features a meeting between Creata Ventures and BytePlus, and David Tuganov's visit to Qazaq AI Research University, whose rector is Hisham Nukhu.",
    tags: ["венчур", "AI", "партнёрства"], tags_en: ["venture", "AI", "partnerships"],
    photo: { url: P("c65"), credit: CR, credit_en: CR_EN },
    photos: [g("67", "Хишам Нуху, ректор QAIRU, и Давид Туганов в Qazaq AI Research University", "Hisham Nukhu, rector of QAIRU, and David Tuganov at Qazaq AI Research University")],
    blocks: [
      { t: "p", text: "Что на фото? Creata Ventures и BytePlus. Второй кадр — Хишам Нуху, ректор QAIRU, и Давид Туганов в Qazaq AI Research University." },
      { t: "p", text: "Подробности сделок и программ появятся здесь, когда их подтвердят стороны." }
    ],
    blocks_en: [
      { t: "p", text: "What's in the photos? Creata Ventures and BytePlus. The second frame shows Hisham Nukhu, rector of QAIRU, and David Tuganov at Qazaq AI Research University." },
      { t: "p", text: "Details of deals and programmes will appear here once both sides confirm them." }
    ],
    why: "Стартапы и венчур", why_en: "Startups & VC", actions: []
  },
  {
    id: "e-gamedev-fest", vertical: "eco-games", author: "alisher", published: "2026-11-03T12:00:00+05:00", readMin: 2, views: 760, saves: 40, live: false,
    place: "Казахстан", place_en: "Kazakhstan",
    title: "GameDev Fest и лекции акселератора: инди-зона, VR и менторы",
    title_en: "GameDev Fest and accelerator lectures: an indie zone, VR and mentors",
    lead: "17 июля — GameDev Fest с зоной инди-игр и VR, 10 октября — лекции для команд акселератора.",
    lead_en: "17 July: GameDev Fest with an indie games zone and VR. 10 October: lectures for the accelerator teams.",
    tldr: "GameDev Fest прошёл 17 июля. 10 октября командам акселератора читали лекции Ираклий Петров (Mytona) и Григорий Иванов.",
    tldr_en: "GameDev Fest took place on 17 July. On 10 October the accelerator teams heard lectures from Irakli Petrov (Mytona) and Grigory Ivanov.",
    tags: ["геймдев", "акселератор"], tags_en: ["gamedev", "accelerator"],
    photo: { url: P("c20"), credit: CR, credit_en: CR_EN },
    photos: [
      g("21", "Зона инди-игр", "The indie games zone"),
      g("22", "VR-зона GameDev Fest", "The GameDev Fest VR zone"),
      g("26", "Команды акселератора на лекциях, 10 октября", "Accelerator teams at the lectures, 10 October"),
      g("24", "Ираклий Петров (Mytona), 10 октября", "Irakli Petrov (Mytona), 10 October"),
      g("25", "Григорий Иванов, 10 октября", "Grigory Ivanov, 10 October")
    ],
    blocks: [
      { t: "p", text: "Что было летом? GameDev Fest 17 июля: зона инди-игр и VR-зона." },
      { t: "p", text: "А осенью? 10 октября команды акселератора слушали лекции Ираклия Петрова (Mytona) и Григория Иванова." }
    ],
    blocks_en: [
      { t: "p", text: "What happened in summer? GameDev Fest on 17 July, with an indie games zone and a VR zone." },
      { t: "p", text: "And in autumn? On 10 October the accelerator teams attended lectures by Irakli Petrov (Mytona) and Grigory Ivanov." }
    ],
    why: "Ты интересуешься геймдевом", why_en: "You're into gamedev", actions: ["o-gamejam"]
  }
];

/* правки существующих eco-материалов */
exports.ecoPatch = {
  "e-paper": {
    photo: { url: P("c39"), credit: CR, credit_en: CR_EN }, photos: [g("01", "Обложка номера 01: Digital Creative Zone, 2 октября", "Issue 01 cover: Digital Creative Zone, 2 October")],
    title_en: "Creata News goes to print: the first issue is out in November",
    lead_en: "Do we need a newspaper? We do. Once a month the best stories from the site and Telegram are collected into a print issue. The first one is November's.",
    tldr_en: "Creata News has three entry points: Telegram t.me/creata_news, the creata.news site and a monthly print issue. The first issue comes out in November 2026.",
    tags_en: ["Creata News", "print"], why_en: "A Creata Group story", place: "Казахстан · Алматы", place_en: "Kazakhstan · Almaty",
    blocks_en: [
      { t: "p", text: "What's inside? The month's best stories from two layers: World, innovation news from everywhere, and Creata Ecosystem, the group's projects, people and events." },
      { t: "p", text: "Where to read every day? On Telegram at t.me/creata_news and here on creata.news. The paper is for those who like to turn pages." },
      { t: "facts", title: "What to do", items: ["Subscribe to t.me/creata_news", "Suggest a person for the next issue", "Pick up an issue at Creata events"] }
    ]
  },
  "e-morning": {
    photo: { url: P("c47"), credit: CR, credit_en: CR_EN }, photos: [],
    title_en: "Creata Morning: three minutes at 08:00",
    lead_en: "Three minutes and you know what changed in the world while you slept. We open with the Creator's Forecast and close with the Game of the Day.",
    tldr_en: "The morning edition arrives at 08:00 your time: five news items, three opportunities and one story outside your bubble. It opens with the Creator's Forecast and ends with the Game of the Day.",
    tags_en: ["Creata Morning", "digest"], why_en: "A Creata Group story", place: "Онлайн", place_en: "Online",
    blocks_en: [
      { t: "p", text: "How does it work? Pick Telegram or email, choose 3–5 topics and your stage: exploring, trying or building. In the morning you get five news items, three opportunities and one story outside your bubble." },
      { t: "p", text: "The Creator's Forecast is a tip for the day by your sign. Your sign is your profession or interest, and the tip is based on a real trend or deadline. The Game of the Day is a five-letter word puzzle you can share." }
    ]
  },
  "e-seifert": { photo: { url: P("c14"), credit: CR, credit_en: CR_EN } },
  "e-pilot": { photo: { url: P("c26"), credit: CR, credit_en: CR_EN } },
  "d-robot-school": { photo: { url: P("c50"), credit: "Демо-фото", credit_en: "Demo photo" } },
  "d-climate": { photo: { url: P("c76"), credit: "Демо-фото", credit_en: "Demo photo" } }
};

exports.commentsEn = [
  { id: "c4", article: "a-hogwarts", name: "Erlan", text: "I wish every district had a kindergarten like this!", status: "published", date: "2026-11-13T09:05:00+05:00", lang: "en" },
  { id: "c5", article: "e-dcz", name: "Dana", text: "The footpong pitch on the roof is the best thing I've seen this month.", status: "published", date: "2026-11-13T11:30:00+05:00", lang: "en" }
];
exports.commentsRuExtra = [
  { id: "c6", article: "e-dcz", name: "Айгерим", text: "Впервые надела VR-шлем именно там. Ждём следующий год!", status: "published", date: "2026-11-13T11:10:00+05:00", lang: "ru" }
];

/* ---------- Мир: новости из других стран (пересказы с источниками) ---------- */
const MAP = (cc) => ({ url: "assets/maps/" + cc + ".svg", credit: "Карта: Creata News", credit_en: "Map: Creata News", map: true });
exports.worldMore = [
  {
    id: "w-ucode", vertical: "ai", country: "UZ", author: "merey", published: "2026-11-09T12:00:00+05:00", readMin: 3, views: 1260, saves: 94,
    place: "Узбекистан · Ташкент", place_en: "Uzbekistan · Tashkent",
    title: "Ucode из Ташкента: ИИ собирает бэкенд за 20 минут",
    title_en: "Tashkent's Ucode: AI builds a backend in 20 minutes",
    lead: "Основатели аутсорс-компании Udevs вложили около $300 тысяч собственных денег в платформу для разработчиков и вышли на рынок США.",
    lead_en: "The founders of outsourcing firm Udevs put about $300,000 of their own money into a developer platform and entered the US market.",
    tldr: "Ucode генерирует и меняет бэкенд приложений с помощью ИИ: MVP — за 15–20 минут. У платформы 20–30 платящих клиентов, двое из США, выручка около $5 тысяч в месяц. Цель — $10 тысяч в месяц и безубыточность к концу 2026 года.",
    tldr_en: "Ucode uses AI to generate and change app backends: an MVP takes 15–20 minutes. It has 20–30 paying clients, two of them in the US, and about $5,000 in monthly revenue. The goal is $10,000 a month and break-even by the end of 2026.",
    tags: ["AI", "стартапы", "Центральная Азия"], tags_en: ["AI", "startups", "Central Asia"],
    blocks: [
      { t: "stats", items: [{ n: "$300K", l: "своих денег, без инвесторов" }, { n: "15–20 мин", l: "на генерацию MVP" }, { n: "$33", l: "базовый тариф в месяц" }] },
      { t: "p", text: "Кто это сделал? Абдулло Хидоятов и Азизбек Баходиров, которые в 2020 году основали аутсорс-компанию Udevs. Ucode вырос из их собственной боли: поддерживать готовые системы было дорого." },
      { t: "p", text: "Как работает? Платформа строит и меняет бэкенд по описанию. Для планирования и архитектуры используется Claude, для задач попроще — GPT-5 и Gemini. В одном из проектов получилось 163 таблицы и 293 связи." },
      { t: "quote", text: "Я бы сделал самый простой MVP и сказал: сначала платите, остальное улучшим потом.", who: "Абдулло Хидоятов, сооснователь" },
      { t: "p", lvl: "deep", text: "Команда — пять человек. Пилоты в Пакистане, Бангладеш и Индии собрали около 200 заинтересованных пользователей. Осенью 2025 года основатель прошёл Silicon Valley Residency от Astana Hub." }
    ],
    blocks_en: [
      { t: "stats", items: [{ n: "$300K", l: "own money, no investors" }, { n: "15–20 min", l: "to generate an MVP" }, { n: "$33", l: "basic plan per month" }] },
      { t: "p", text: "Who built it? Abdullo Khidoyatov and Azizbek Bakhodirov, who founded the outsourcing company Udevs in 2020. Ucode grew out of their own pain: maintaining finished systems was expensive." },
      { t: "p", text: "How does it work? The platform builds and changes a backend from a description. Claude handles planning and architecture; GPT-5 and Gemini handle simpler tasks. One project ended up with 163 tables and 293 relationships." },
      { t: "quote", text: "I'd build the most basic MVP and say: pay first, and we'll improve everything else afterwards.", who: "Abdullo Khidoyatov, co-founder" },
      { t: "p", lvl: "deep", text: "The team is five people. Pilots in Pakistan, Bangladesh and India drew about 200 interested users. In autumn 2025 the founder went through Astana Hub's Silicon Valley Residency." }
    ],
    source: { name: "DigitalBusiness.kz", author: "Александрина Есенгулова", author_en: "Aleksandrina Esengulova", date: "2026-10-05", url: "https://digitalbusiness.kz/en/2026-10-05/uzbekistantsi-vlozhili-300-tisyach-v-ii-servis-dlya-razrabotchikov-i-vishli-na-rinok-ssha/" },
    why: "Ты интересуешься AI", why_en: "You're into AI", actions: ["o-gamejam"]
  },
  {
    id: "w-uz-hk", vertical: "fintech", country: "UZ", author: "merey", published: "2026-11-10T15:00:00+05:00", readMin: 2, views: 830, saves: 51,
    place: "Узбекистан → Гонконг", place_en: "Uzbekistan → Hong Kong",
    title: "Пять стартапов из Узбекистана съездили на Hong Kong FinTech Week",
    title_en: "Five Uzbek startups went to Hong Kong FinTech Week",
    lead: "IT Park привёз в Гонконг Tass Vision, OX System, Unitlab AI, Aziza AI и Qwatt — в ответ на визит Cyberport в Ташкент.",
    lead_en: "IT Park brought Tass Vision, OX System, Unitlab AI, Aziza AI and Qwatt to Hong Kong, returning Cyberport's visit to Tashkent.",
    tldr: "2–6 ноября 2026 года пять стартапов из Узбекистана участвовали в Hong Kong FinTech Week и Cyberport Venture Capital Forum. OX System и Tass Vision питчили инвесторам, IT Park провёл встречи с Cyberport, HKSTP, InvestHK и Hang Seng Bank.",
    tldr_en: "On 2–6 November 2026 five Uzbek startups took part in Hong Kong FinTech Week and the Cyberport Venture Capital Forum. OX System and Tass Vision pitched to investors, and IT Park met Cyberport, HKSTP, InvestHK and Hang Seng Bank.",
    tags: ["финтех", "венчур"], tags_en: ["fintech", "venture"],
    blocks: [
      { t: "p", text: "Что за поездка? Программа Global Landing Program 2026 от Cyberport. Это ответный визит: до этого представители Cyberport приезжали на ICT Week Uzbekistan." },
      { t: "facts", title: "Кто поехал", items: ["Tass Vision", "OX System", "Unitlab AI", "Aziza AI", "Qwatt"] },
      { t: "p", text: "Что получилось? OX System и Tass Vision выступили на питч-сессии Global Track. Обсуждали выход на рынок Гонконга, инкубацию и инвестиции, а с VOTEE AI и FCC Analytics — сотрудничество в ИИ и аналитике данных." }
    ],
    blocks_en: [
      { t: "p", text: "What trip? Cyberport's Global Landing Program 2026. It was a return visit: Cyberport representatives had earlier attended ICT Week Uzbekistan." },
      { t: "facts", title: "Who went", items: ["Tass Vision", "OX System", "Unitlab AI", "Aziza AI", "Qwatt"] },
      { t: "p", text: "What came of it? OX System and Tass Vision pitched at the Global Track session. Talks covered Hong Kong market entry, incubation and investment, plus AI and data analytics cooperation with VOTEE AI and FCC Analytics." }
    ],
    source: { name: "IT Park Uzbekistan", author: "", date: "2026-11-10", url: "https://www.it-park.uz/en/itpark/news/uzbekistan-s-startups-participate-in-hong-kong-fintech-week-and-cyberport-venture-capital-forum-2026" },
    why: "Стартапы и венчур", why_en: "Startups & VC", actions: []
  },
  {
    id: "w-kenya-ai", vertical: "ai", country: "KE", author: "saya", published: "2026-11-08T10:00:00+05:00", readMin: 3, views: 1040, saves: 77,
    place: "Кения · Найроби", place_en: "Kenya · Nairobi",
    title: "База данных на Raspberry Pi и жестовый язык на 3D-аватарах: ИИ из Найроби",
    title_en: "A database on a Raspberry Pi and sign language on 3D avatars: AI from Nairobi",
    lead: "На AI Everything Kenya × GITEX Kenya африканские стартапы показали ИИ, который работает на дешёвом железе и для людей, которых технологии обычно забывают.",
    lead_en: "At AI Everything Kenya × GITEX Kenya, African startups showed AI that runs on cheap hardware and serves people tech usually forgets.",
    tldr: "В 2025 году Кения привлекла $1,04 млрд технологических инвестиций, на 72% больше, чем годом раньше. На выставке в Найроби показали HeatherDB, Signvrse, straitPay, Joritu и AuraLearn.",
    tldr_en: "In 2025 Kenya attracted $1.04 billion in tech investment, up 72% year on year. The Nairobi expo featured HeatherDB, Signvrse, straitPay, Joritu and AuraLearn.",
    tags: ["Африка", "AI", "инклюзия"], tags_en: ["Africa", "AI", "inclusion"],
    blocks: [
      { t: "stats", items: [{ n: "$1,04 млрд", l: "техинвестиций в Кению в 2025" }, { n: "+72%", l: "за год" }, { n: "100+", l: "инвесторов из 20+ стран" }] },
      { t: "facts", title: "Кого показали", items: ["Aphorion Labs — HeatherDB, база данных со встроенным ИИ, работает на Raspberry Pi и сервере за $5", "Signvrse — жестовый язык через 3D-аватары для медицины, школ и госуслуг", "Strait Sahara (Нигерия) — straitPay, трансграничные платежи с ML против мошенничества", "Joritu — ИИ для планирования стройки", "AuraLearn — превращает графики и схемы в пространственный звук для незрячих студентов"] },
      { t: "p", text: "Где и когда? Саммит прошёл 19 мая в Sarit Expo Centre, выставка — 20–21 мая 2026 года в KICC, Найроби. Организаторы — inD и офис спецпосланника Кении по технологиям." }
    ],
    blocks_en: [
      { t: "stats", items: [{ n: "$1.04 bn", l: "tech investment in Kenya, 2025" }, { n: "+72%", l: "year on year" }, { n: "100+", l: "investors from 20+ countries" }] },
      { t: "facts", title: "Who showed what", items: ["Aphorion Labs: HeatherDB, a database with built-in AI that runs on a Raspberry Pi and a $5 server", "Signvrse: sign language through 3D avatars for healthcare, schools and public services", "Strait Sahara (Nigeria): straitPay, ML-powered cross-border payments against fraud", "Joritu: AI for construction planning", "AuraLearn: turns charts and diagrams into spatial audio for blind students"] },
      { t: "p", text: "Where and when? The summit took place on 19 May at Sarit Expo Centre and the expo on 20–21 May 2026 at KICC, Nairobi. Organisers: inD and the Office of Kenya's Special Envoy on Technology." }
    ],
    source: { name: "Africa.com", author: "", date: "2026-05-19", url: "https://africa.com/whats-happening/african-startups-rewrite-the-ai-playbook-at-ai-everything-kenya-x-gitex-kenya-2026" },
    why: "Новая тема для тебя: Африка", why_en: "New for you: Africa", actions: ["o-stem-week"]
  },
  {
    id: "w-opin", vertical: "climate", country: "BR", countries: ["BR", "IN"], author: "saya", published: "2026-11-06T14:00:00+05:00", readMin: 2, views: 690, saves: 38,
    place: "Бразилия · Индия", place_en: "Brazil · India",
    title: "Бразилия и Индия запустили OPIN — сеть «планетарного интеллекта»",
    title_en: "Brazil and India launch OPIN, an “open planetary intelligence” network",
    lead: "Две страны предлагают Глобальному Югу общую цифровую инфраструктуру для климата: от раннего оповещения до платежей фермерам.",
    lead_en: "The two countries offer the Global South shared digital infrastructure for climate: from early warnings to payments for farmers.",
    tldr: "Open Planetary Intelligence Network объявили 22 февраля 2026 года во время визита президента Лулы в Индию. Идея — превратить разрозненные данные в «планетарный интеллект» и быстрее доводить климатическое финансирование до развивающихся стран.",
    tldr_en: "The Open Planetary Intelligence Network was announced on 22 February 2026 during President Lula's visit to India. The idea is to turn scattered data into “planetary intelligence” and get climate finance to developing countries faster.",
    tags: ["климат", "цифровая инфраструктура"], tags_en: ["climate", "digital infrastructure"],
    blocks: [
      { t: "p", text: "Что это? Совместная инициатива Бразилии и Индии, которая использует цифровую общественную инфраструктуру (DPI) для климатических задач. Остальные страны Глобального Юга приглашают присоединиться." },
      { t: "facts", title: "Где хотят применять", items: ["Системы раннего оповещения", "Платежи мелким фермерам за климатически умное земледелие", "Децентрализованные рынки энергии", "Совместимые платформы климатического финансирования"] },
      { t: "p", text: "На чём строится? На опыте India Stack, председательства Индии в G20 в 2023 году, Бразилии в G20 в 2024 году и COP30 в 2025 году." }
    ],
    blocks_en: [
      { t: "p", text: "What is it? A joint Brazil–India initiative that uses digital public infrastructure (DPI) for climate goals. Other Global South countries are invited to join." },
      { t: "facts", title: "Where it should apply", items: ["Early warning systems", "Payments to smallholders for climate-smart farming", "Decentralised energy markets", "Interoperable climate finance platforms"] },
      { t: "p", text: "What does it build on? India Stack, India's 2023 G20 presidency, Brazil's 2024 G20 presidency and COP30 in 2025." }
    ],
    source: { name: "COP30", author: "", date: "2026-02-22", url: "https://cop30.br/en/news-about-cop30/following-cop30-and-the-ai-impact-summit-brazil-and-india-launch-the-open-planetary-intelligence-network-opin" },
    why: "Климат и импакт", why_en: "Climate & impact", actions: []
  },
  {
    id: "w-tokyo-humanoids", vertical: "robotics", country: "JP", author: "zarina", published: "2026-11-05T11:00:00+05:00", readMin: 3, views: 1520, saves: 110,
    place: "Япония · Токио", place_en: "Japan · Tokyo",
    title: "Гуманоиды танцуют и вдевают нитку в иголку: Токио догоняет Китай",
    title_en: "Humanoids dance and thread needles: Tokyo tries to catch up with China",
    lead: "На Humanoids Summit Tokyo японские разработчики показали роботов для аэропортов и руку, которая вдевает нитку, а китайские — танцующего робота от $5 500.",
    lead_en: "At Humanoids Summit Tokyo, Japanese developers showed airport robots and a hand that threads a needle, while Chinese firms showed a dancing robot from $5,500.",
    tldr: "Япония придумала гуманоидов (Asimo от Honda показали в 2000 году), но массовое дешёвое производство сейчас у Китая. Эксперты называют отставание Японии «синдромом Галапагосов»: изобретения не выходят на мировой рынок.",
    tldr_en: "Japan pioneered humanoids (Honda showed Asimo in 2000), but cheap mass production now belongs to China. Experts call Japan's lag “Galapagos syndrome”: inventions don't reach global markets.",
    tags: ["роботы", "Азия"], tags_en: ["robots", "Asia"],
    blocks: [
      { t: "stats", items: [{ n: "$5 500", l: "стартовая цена Mini Pi Plus" }, { n: "2000", l: "год первого показа Asimo" }, { n: "28%", l: "японцев тревожатся из-за ИИ (в США — 50%)" }] },
      { t: "facts", title: "Что показали", items: ["High Torque (Китай) — маленький танцующий Mini Pi Plus", "GMO (Токио) — гуманоид для грузов Japan Airlines и работы в аэропорту", "Honda — четырёхпалая рука, которая закручивает винты и вдевает нитку", "Хироси Исигуро (Осакский университет) — андроид-копия самого себя"] },
      { t: "p", text: "Зачем Японии роботы? Чтобы закрыть нехватку рабочих рук — например, в аэропортах." }
    ],
    blocks_en: [
      { t: "stats", items: [{ n: "$5,500", l: "Mini Pi Plus starting price" }, { n: "2000", l: "year Asimo was first shown" }, { n: "28%", l: "of Japanese anxious about AI (US: 50%)" }] },
      { t: "facts", title: "What was shown", items: ["High Torque (China): the small dancing Mini Pi Plus", "GMO (Tokyo): a humanoid for Japan Airlines cargo and airport chores", "Honda: a four-fingered hand that screws bolts and threads a needle", "Hiroshi Ishiguro (Osaka University): an android clone of himself"] },
      { t: "p", text: "Why does Japan need robots? To fill labour shortages, for example at airports." }
    ],
    source: { name: "Associated Press / KSL.com", author: "Юри Кагеяма", author_en: "Yuri Kageyama", date: "2026-05-29", url: "https://www.ksl.com/article/51503542/humanoids-dance-and-thread-needles-as-japanese-robotics-developers-look-to-outdo-chinese" },
    why: "Робототехника", why_en: "Robotics", actions: ["o-stem-week"]
  }
];
exports.worldMore.forEach((a) => { a.photo = MAP(a.country); a.photos = []; });
