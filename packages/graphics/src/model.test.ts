import {test,expect} from 'bun:test'
import {csvToDataset,datasetCsv,localized,validateDataset,donutSegments,axisLabel,type GraphicData} from './model'
const sample=():GraphicData=>({type:'bar',unit:'USD',period:'2025',sourceUrl:'https://example.org/report',sourceName:'Report',methodology:localized('Reported values'),verified:false,labels:[localized('A'),localized('B')],series:[{name:localized('Amount'),values:[1,2]}]})
test('financial CSV refuses absent values and never converts a blank into zero',()=>{
 for(const csv of ['Label,Value\nA,1\nB,','Label,Value\nA,USD 100\nB,2','Label,Value\nA,Infinity\nB,2'])expect(()=>csvToDataset(csv,sample())).toThrow()
 expect(csvToDataset('Label;Value\n"A; B";0\nC;-1.5',sample()).series[0].values).toEqual([0,-1.5])
})
test('CSV round-trip keeps quoted labels, negative numbers and revokes data verification',()=>{
 const g=sample();g.verified=true;const imported=csvToDataset('Label,Amount\n"A, B",-1\nC,2',g)
 expect(imported.labels[0].ru).toBe('A, B');expect(imported.verified).toBe(false);expect(csvToDataset(datasetCsv(imported,'en'),g).series[0].values).toEqual([-1,2])
})
test('misleading donut datasets and unequal series are rejected',()=>{
 const g=sample();g.type='donut';g.series[0].values=[-1,2];expect(validateDataset(g)).not.toBeNull()
 g.type='line';g.series[0].values=[1];expect(validateDataset(g)).not.toBeNull()
})
test('a one-hundred-percent donut keeps a complete ring and large ticks stay readable',()=>{
 const [full,empty]=donutSegments([100,0]);expect(full.length).toBe(full.circumference);expect(empty.length).toBe(0)
 expect(axisLabel(1e12,'en')).toBe('1 T');expect(axisLabel(-1e9,'ru')).toBe('-1 млрд')
})
