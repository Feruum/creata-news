import fs from 'fs';
import { geoOrthographic, geoPath, geoCentroid, geoGraticule10 } from 'd3-geo';
import * as topo from 'topojson-client';
import countries from 'i18n-iso-countries';
const world = JSON.parse(fs.readFileSync('node_modules/world-atlas/countries-110m.json'));
const fc = topo.feature(world, world.objects.countries);
const land = topo.feature(world, world.objects.land);
fs.mkdirSync('../site/assets/maps',{recursive:true});
for (const cc of ['UZ','KE','BR','JP','IN','KZ']) {
  const num = countries.alpha2ToNumeric(cc);
  const f = fc.features.find(x=>String(x.id).padStart(3,'0')===num);
  const [lon,lat]=geoCentroid(f);
  const W=1200,H=800,R=820;
  const pr=geoOrthographic().rotate([-lon,-lat+4]).translate([W/2,H/2+70]).scale(R).clipAngle(90);
  const p=geoPath(pr);
  const others=fc.features.filter(x=>x!==f).map(x=>p(x)).filter(Boolean).join('');
  const [cx,cy]=p.centroid(f);
  const svg=`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMid slice">
<defs><radialGradient id="g" cx="40%" cy="30%" r="75%"><stop offset="0" stop-color="#2a2760"/><stop offset="1" stop-color="#0b0b14"/></radialGradient>
<radialGradient id="s" cx="50%" cy="50%" r="50%"><stop offset="0.85" stop-color="#4B44F0" stop-opacity="0"/><stop offset="1" stop-color="#4B44F0" stop-opacity=".35"/></radialGradient></defs>
<rect width="${W}" height="${H}" fill="#0b0b14"/>
<circle cx="${W/2}" cy="${H/2+70}" r="${R+30}" fill="url(#s)"/>
<path d="${p({type:'Sphere'})}" fill="url(#g)"/>
<path d="${p(geoGraticule10())}" fill="none" stroke="#ffffff" stroke-opacity=".07" stroke-width="1"/>
<path d="${others}" fill="#3a3a52" stroke="#0b0b14" stroke-width="1"/>
<path d="${p(f)}" fill="#4B44F0" stroke="#c9c6ff" stroke-width="2"/>
<circle cx="${cx}" cy="${cy}" r="9" fill="#fff"/><circle cx="${cx}" cy="${cy}" r="26" fill="none" stroke="#fff" stroke-opacity=".6" stroke-width="2"/>
</svg>`;
  fs.writeFileSync(`../site/assets/maps/${cc}.svg`,svg);
  console.log(cc, (svg.length/1024).toFixed(0)+'KB');
}
