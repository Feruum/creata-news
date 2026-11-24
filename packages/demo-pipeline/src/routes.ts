/** Closed API surface: typos, unsupported verbs and extra suffixes must fail. */
const routes:Array<[string,RegExp]>=[
 ['GET',/^auth\/me$/],['POST',/^auth\/(logout|dev-login)$/],['GET',/^auth\/(google|telegram)\/(start|callback)$/],
 ['PATCH',/^account$/],['GET',/^account\/saved$/],['POST|DELETE',/^account\/saved\/(articles?|opportunit(?:y|ies))\/[^/]+$/],['POST',/^account\/telegram-link$/],
 ['GET',/^account\/followed-stories$/],['POST|DELETE',/^account\/followed-stories\/[^/]+$/],
 ['POST',/^account\/followed-stories\/[^/]+\/read$/],
 ['GET',/^public\/(categories|articles|popular|authors|opportunities|launches|newspapers)$/],['GET',/^public\/(articles|launches|newspapers|pages)\/[^/]+$/],
 ['GET|POST',/^admin\/(articles|sources|authors|opportunities|launches|newspapers)$/],
 ['GET|PATCH',/^admin\/articles\/[^/]+$/],['GET',/^admin\/articles\/[^/]+\/(revisions|research)$/],['POST',/^admin\/articles\/[^/]+\/(actions|graphic-draft)$/],
 ['GET|PUT|DELETE',/^admin\/articles\/[^/]+\/follow-up$/],['GET',/^admin\/story-follow-ups$/],
 ['PATCH|DELETE',/^admin\/(sources|authors|opportunities|launches)\/[^/]+$/],['POST',/^admin\/sources\/[^/]+\/(run|check|test)$/],
 ['GET',/^admin\/(bank|jobs|runs|digests|deliveries|audit|analytics|users|staff|pages|media)$/],
 ['GET|PATCH',/^admin\/settings$/],['PATCH',/^admin\/users\/[^/]+$/],
 ['GET',/^admin\/(jobs|runs)\/[^/]+$/],['POST',/^admin\/jobs\/[^/]+\/(retry|cancel|fail)$/],
 ['POST',/^admin\/bank\/collect$/],['GET',/^admin\/bank\/(similar|merges)$/],['POST',/^admin\/bank\/merges\/[^/]+\/(undo|split)$/],['POST',/^admin\/bank\/[^/]+\/(draft|graphic|merge)$/],['GET',/^admin\/bank\/[^/]+\/evidence$/],
 ['POST',/^admin\/digests(?:\/preview)?$/],['GET',/^admin\/digests\/[^/]+$/],['POST',/^admin\/digests\/[^/]+\/(approve|send)$/],['POST',/^admin\/deliveries\/[^/]+\/resolve$/],
 ['GET|PATCH',/^admin\/pages\/[^/]+$/],['GET',/^admin\/pages\/[^/]+\/revisions$/],['POST',/^admin\/pages\/[^/]+\/(publish|unpublish)$/],
 ['POST',/^admin\/media(?:\/cleanup)?$/],['PATCH|DELETE',/^admin\/media\/[^/]+$/],
 ['GET|PATCH',/^admin\/newspapers\/[^/]+$/],['GET',/^admin\/newspapers\/[^/]+\/preview$/],['POST',/^admin\/newspapers\/[^/]+\/(publish|unpublish)$/],
]
export function supportsRoute(method:string,path:string) {return routes.some(([methods,pattern])=>methods.split('|').includes(method)&&pattern.test(path))}
