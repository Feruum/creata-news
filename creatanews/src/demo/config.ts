/** Ordinary builds always use the real backend. */
export const demoMode=process.env.NEXT_PUBLIC_CREATA_DEMO_MODE==='1'
export const demoCopy={
 ru:{notice:'Демо Creata News · Примерные материалы. Сохранения и настройки остаются только в вашем браузере.',login:'Открыть демо-профиль',telegram:'В демо Telegram не подключается и рассылки не отправляются.'},
 kk:{notice:'Creata News демосы · Үлгі материалдар. Сақталғандар мен баптаулар тек браузеріңізде қалады.',login:'Демо-профильді ашу',telegram:'Демода Telegram қосылмайды және дайджест жіберілмейді.'},
 en:{notice:'Creata News demo · Sample content. Saves and preferences stay in your browser only.',login:'Open demo profile',telegram:'Telegram connections and digest delivery are disabled in this demo.'},
} as const
