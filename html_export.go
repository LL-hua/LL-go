package main

import (
	"encoding/json"
	"os"
)

// SectionDTO 传给 HTML 的 JSON 结构
type SectionDTO struct {
	StationK      string         `json:"stationK"`
	Station       float64        `json:"station"`
	CenterY       float64        `json:"centerY"`
	LOuter        [2]float64     `json:"lOuter"`
	ROuter        [2]float64     `json:"rOuter"`
	Ground        [][2]float64   `json:"ground"`
	Cleared       [][2]float64   `json:"cleared"`
	Design        [][2]float64   `json:"design"`
	LayerPolygons [][][2]float64 `json:"layerPolygons"`
	LeftSlopeTrim [][2]float64   `json:"leftSlopeTrim"`
	RightSlopeTrim [][2]float64  `json:"rightSlopeTrim"`
	LeftSlabs     [][2]float64   `json:"leftSlabs"`
	RightSlabs    [][2]float64   `json:"rightSlabs"`
	Fill          float64        `json:"fill"`
	Cut           float64        `json:"cut"`
	Clear         float64        `json:"clear"`
}

func toDTO(s SectionResult) SectionDTO {
	return SectionDTO{
		StationK:       HuaNum2K(s.Station),
		Station:        s.Station,
		CenterY:        s.CenterY,
		LOuter:         s.LOuter,
		ROuter:         s.ROuter,
		Ground:         s.Ground,
		Cleared:        s.Cleared,
		Design:         s.Design,
		LayerPolygons:  s.Layers,
		LeftSlopeTrim:  s.LeftSlopeTrimmed,
		RightSlopeTrim: s.RightSlopeTrimmed,
		LeftSlabs:      s.LeftSlabs,
		RightSlabs:     s.RightSlabs,
		Fill:           s.FillArea,
		Cut:            s.CutArea,
		Clear:          s.ClearArea,
	}
}

// SaveBatchHTML 批量导出所有断面到一个 HTML
func SaveBatchHTML(sections []SectionResult, title, path string) error {
	dtos := make([]SectionDTO, len(sections))
	for i, s := range sections {
		dtos[i] = toDTO(s)
	}
	jsonBytes, err := json.Marshal(dtos)
	if err != nil {
		return err
	}
	html := templateHTML
	html = replaceOnce(html, "/*__DATA__*/", string(jsonBytes))
	html = replaceOnce(html, "__TITLE__", title)
	return os.WriteFile(path, []byte(html), 0644)
}

func replaceOnce(s, old, new string) string {
	idx := indexOf(s, old)
	if idx < 0 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

func indexOf(s, sub string) int {
	n := len(s)
	m := len(sub)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}

const templateHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
<title>__TITLE__</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
html,body{height:100%;overflow:hidden;-webkit-text-size-adjust:100%;-webkit-tap-highlight-color:transparent}
body{font-family:-apple-system,"Microsoft YaHei",sans-serif;background:#f5f5f5;color:#333;font-size:15px;display:flex;flex-direction:column}
header{flex-shrink:0;padding:6px 12px;background:#0078d4;color:#fff;font-size:14px;font-weight:600}
.toolbar{flex-shrink:0;display:flex;gap:6px;align-items:center;padding:6px 8px;background:#fff;border-bottom:1px solid #e5e5e5}
.toolbar button{flex-shrink:0;padding:6px 12px;font-size:14px;background:#0078d4;color:#fff;border:none;border-radius:6px;cursor:pointer;touch-action:manipulation}
.toolbar button:active{background:#0066b4}
.toolbar input{flex:1;min-width:0;padding:6px 10px;font-size:14px;border:1px solid #ccc;border-radius:6px;background:#fff;text-align:center}
main{flex:1;display:flex;flex-direction:column;min-height:0;padding:6px}
.chart-card{flex:1;background:#fff;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,.08);min-height:0;overflow:hidden;position:relative}
#cv{display:block;width:100%;height:100%;touch-action:none;cursor:grab}
#cv:active{cursor:grabbing}
.reset-btn{position:absolute;top:6px;right:6px;font-size:11px;padding:4px 10px;background:rgba(0,120,212,.85);color:#fff;border:none;border-radius:4px;cursor:pointer;z-index:10;touch-action:manipulation}
.chart-hint{position:absolute;bottom:4px;right:8px;font-size:10px;color:#999;pointer-events:none;background:rgba(255,255,255,.85);padding:2px 6px;border-radius:4px}
</style>
</head>
<body>
<header>__TITLE__</header>
<div class="toolbar">
<button id="prevBtn">◀</button>
<input id="stationInput" type="text" inputmode="text" placeholder="桩号 如 85600 或 K85+600">
<button id="nextBtn">▶</button>
</div>
<main>
<div class="chart-card">
<canvas id="cv"></canvas>
<button id="resetBtn" class="reset-btn">⤢ 重置</button>
<div class="chart-hint">双指缩放 / 拖动 / 双击放大</div>
</div>
</main>
<script>
const allSections = /*__DATA__*/;
const cv=document.getElementById('cv'),ctx=cv.getContext('2d');
let canvasW=0,canvasH=0,dpr=window.devicePixelRatio||1,viewWorldW=1,currentData=null,currentIdx=0;
let tf={offsetX:0,offsetY:0,scale:1};

function w2c(wx,wy){return [wx*tf.scale+tf.offsetX,-wy*tf.scale+tf.offsetY];}
function poly(pts,color,w,dash){if(!pts||pts.length<2)return;ctx.beginPath();ctx.strokeStyle=color;ctx.lineWidth=w;ctx.setLineDash(dash||[]);for(let i=0;i<pts.length;i++){const[a,b]=w2c(pts[i][0],pts[i][1]);i?ctx.lineTo(a,b):ctx.moveTo(a,b);}ctx.stroke();ctx.setLineDash([]);}
function polyFill(pts,fill,stroke,w){if(!pts||pts.length<3)return;ctx.beginPath();for(let i=0;i<pts.length;i++){const[a,b]=w2c(pts[i][0],pts[i][1]);i?ctx.lineTo(a,b):ctx.moveTo(a,b);}ctx.closePath();if(fill){ctx.fillStyle=fill;ctx.fill();}if(stroke){ctx.strokeStyle=stroke;ctx.lineWidth=w||1;ctx.stroke();}}

function render(){
if(!currentData)return;
ctx.clearRect(0,0,canvasW,canvasH);ctx.fillStyle='#fff';ctx.fillRect(0,0,canvasW,canvasH);
const res=currentData;
const fontPx=Math.max(1,viewWorldW*0.02*tf.scale);
const offScr=0.2*tf.scale;

if(res.layerPolygons)res.layerPolygons.forEach((p,i)=>polyFill(p,i%2?'#e8f0f8':'#cfe4f7',i%2?'#7bc0e8':'#4a90d9',1));
if(res.ground)poly(res.ground,'#0a0',2);
if(res.cleared)poly(res.cleared,'#888',1,[4,3]);
if(res.design)poly(res.design,'#d00',2.5);

function labelV(pt,lines,color){const[a,b]=w2c(pt[0],pt[1]);ctx.save();ctx.translate(a,b-offScr);ctx.rotate(-Math.PI/2);ctx.font=fontPx+'px sans-serif';ctx.fillStyle=color;ctx.textAlign='left';ctx.textBaseline='middle';const g=fontPx*1.1;for(let i=0;i<lines.length;i++)ctx.fillText(lines[i],0,i*g);ctx.restore();}
if(res.lOuter)labelV(res.lOuter,['w='+Math.abs(res.lOuter[0]).toFixed(3),res.lOuter[1].toFixed(3)],'#0078d4');
if(res.centerY!==undefined)labelV([0,res.centerY],[res.centerY.toFixed(3)],'#0078d4');
if(res.rOuter)labelV(res.rOuter,['w='+Math.abs(res.rOuter[0]).toFixed(3),res.rOuter[1].toFixed(3)],'#0078d4');

function labelSlope(pts,color){if(!pts||pts.length<2)return;for(let i=0;i<pts.length-1;i++){const x1=pts[i][0],y1=pts[i][1],x2=pts[i+1][0],y2=pts[i+1][1];const dx=x2-x1,dy=y2-y1;if(Math.abs(dy)<1e-6)continue;const lab='1:'+Math.abs(dx/dy).toFixed(2);const ang=-Math.atan2(dy,dx);const[a,b]=w2c((x1+x2)/2,(y1+y2)/2);ctx.save();ctx.translate(a,b);ctx.rotate(ang);ctx.font=fontPx+'px sans-serif';ctx.fillStyle=color;ctx.textAlign='center';ctx.textBaseline='bottom';ctx.fillText(lab,0,0);ctx.restore();}}
labelSlope(res.leftSlopeTrim,'#d00');labelSlope(res.rightSlopeTrim,'#d00');

function labelCF(slabs,sx,sy,isLeft,color){if(!slabs||!slabs.length)return;let cx=sx,cy=sy;const idxs=isLeft?[...slabs.keys()].reverse():[...slabs.keys()];for(const i of idxs){const w=slabs[i][0],sl=slabs[i][1];const mx=isLeft?cx-w/2:cx+w/2;const my=cy+(w/2)*(sl/100);const vs=isLeft?-sl:sl;const ang=-Math.atan2(vs/100,1);const nm=Math.hypot(1,vs/100);const nx=-(vs/100)/nm,ny=1/nm;const[a,b]=w2c(mx+nx*0.3,my+ny*0.3);ctx.save();ctx.translate(a,b);ctx.rotate(ang);ctx.font=fontPx+'px sans-serif';ctx.fillStyle=color;ctx.textAlign='center';ctx.textBaseline='middle';ctx.fillText(sl.toFixed(2)+'%',0,0);ctx.restore();cx=isLeft?cx-w:cx+w;cy=cy+w*(sl/100);}}
labelCF(res.leftSlabs,0,res.centerY,true,'#0050a0');labelCF(res.rightSlabs,0,res.centerY,false,'#0050a0');

ctx.font='bold 14px sans-serif';ctx.fillStyle='#333';ctx.textAlign='left';ctx.textBaseline='alphabetic';
ctx.fillText(res.stationK,10,22);
ctx.font='12px sans-serif';
ctx.fillStyle='#2a8';ctx.fillText('填 '+res.fill.toFixed(3)+' ㎡',10,42);
ctx.fillStyle='#d33';ctx.fillText('挖 '+res.cut.toFixed(3)+' ㎡',10,60);
ctx.fillStyle='#888';ctx.fillText('清 '+res.clear.toFixed(3)+' ㎡',10,78);
}

function resizeCv(){const r=cv.getBoundingClientRect();canvasW=r.width;canvasH=r.height;dpr=window.devicePixelRatio||1;cv.width=Math.round(canvasW*dpr);cv.height=Math.round(canvasH*dpr);ctx.setTransform(dpr,0,0,dpr,0,0);}
window.addEventListener('resize',()=>{resizeCv();render();});

function fit(res){const pts=[];const push=a=>{if(a)for(const p of a)pts.push(p);};push(res.ground);push(res.cleared);push(res.design);if(res.layerPolygons)for(const p of res.layerPolygons)push(p);if(!pts.length)return;let mnx=pts[0][0],mxx=pts[0][0],mny=pts[0][1],mxy=pts[0][1];for(const p of pts){if(p[0]<mnx)mnx=p[0];if(p[0]>mxx)mxx=p[0];if(p[1]<mny)mny=p[1];if(p[1]>mxy)mxy=p[1];}const px=(mxx-mnx)*0.08||1,py=(mxy-mny)*0.15||1;mnx-=px;mxx+=px;mny-=py;mxy+=py;const ww=mxx-mnx,wh=mxy-mny;viewWorldW=ww;const sc=Math.min(canvasW/ww,canvasH/wh);const ccx=(mnx+mxx)/2,ccy=(mny+mxy)/2;tf.offsetX=canvasW/2-ccx*sc;tf.offsetY=canvasH/2+ccy*sc;tf.scale=sc;render();}

let drag=false,dsx=0,dsy=0,pinch=false,pd=0,pls=1,pcx=0,pcy=0,pox=0,poy=0,lastTap=0,ltx=0,lty=0;
function z(cx,cy,f){const os=tf.scale;let ns=Math.max(0.01,Math.min(200,os*f));const r=ns/os;tf.offsetX=cx-(cx-tf.offsetX)*r;tf.offsetY=cy-(cy-tf.offsetY)*r;tf.scale=ns;render();}
cv.addEventListener('mousedown',e=>{drag=true;const r=cv.getBoundingClientRect();dsx=e.clientX-r.left-tf.offsetX;dsy=e.clientY-r.top-tf.offsetY;});
window.addEventListener('mousemove',e=>{if(!drag)return;const r=cv.getBoundingClientRect();tf.offsetX=e.clientX-r.left-dsx;tf.offsetY=e.clientY-r.top-dsy;render();});
window.addEventListener('mouseup',()=>drag=false);
cv.addEventListener('wheel',e=>{e.preventDefault();const r=cv.getBoundingClientRect();z(e.clientX-r.left,e.clientY-r.top,e.deltaY>0?0.95:1.05);},{passive:false});
cv.addEventListener('touchstart',e=>{const t=e.touches,r=cv.getBoundingClientRect();if(t.length===1){drag=true;pinch=false;const cx=t[0].clientX-r.left,cy=t[0].clientY-r.top;dsx=cx-tf.offsetX;dsy=cy-tf.offsetY;const now=Date.now();if(now-lastTap<300&&Math.hypot(cx-ltx,cy-lty)<20){z(cx,cy,1.5);lastTap=0;}else{lastTap=now;ltx=cx;lty=cy;}}else if(t.length===2){drag=false;pinch=true;pd=Math.hypot(t[0].clientX-t[1].clientX,t[0].clientY-t[1].clientY);pcx=(t[0].clientX+t[1].clientX)/2-r.left;pcy=(t[0].clientY+t[1].clientY)/2-r.top;pls=tf.scale;pox=tf.offsetX;poy=tf.offsetY;}},{passive:false});
cv.addEventListener('touchmove',e=>{e.preventDefault();const t=e.touches,r=cv.getBoundingClientRect();if(drag&&t.length===1){const cx=t[0].clientX-r.left,cy=t[0].clientY-r.top;tf.offsetX=cx-dsx;tf.offsetY=cy-dsy;render();}else if(pinch&&t.length===2){const cd=Math.hypot(t[0].clientX-t[1].clientX,t[0].clientY-t[1].clientY);if(pd>0){const ccx=(t[0].clientX+t[1].clientX)/2-r.left,ccy=(t[0].clientY+t[1].clientY)/2-r.top;const f=cd/pd;let ns=Math.max(0.01,Math.min(200,pls*f));const rr=ns/pls;tf.offsetX=ccx-(pcx-pox)*rr;tf.offsetY=ccy-(pcy-poy)*rr;tf.scale=ns;render();}}},{passive:false});
cv.addEventListener('touchend',e=>{const rem=e.touches.length;if(rem===0){drag=false;pinch=false;}else if(rem===1){drag=true;pinch=false;const r=cv.getBoundingClientRect(),t=e.touches[0];dsx=t.clientX-r.left-tf.offsetX;dsy=t.clientY-r.top-tf.offsetY;}},{passive:false});

const inputEl=document.getElementById('stationInput');
function show(i){if(i<0)i=0;if(i>=allSections.length)i=allSections.length-1;currentIdx=i;currentData=allSections[i];inputEl.value=currentData.stationK;document.querySelector('header').textContent='__TITLE__ · '+currentData.stationK;resizeCv();fit(currentData);}
function parseStation(s){s=(s||'').trim().toUpperCase().replace(/\s+/g,'');if(!s)return null;const m=s.match(/^K?(\d+)\+(\d+(?:\.\d+)?)$/);if(m)return parseInt(m[1],10)*1000+parseFloat(m[2]);const v=parseFloat(s);return isNaN(v)?null:v;}
function jumpByInput(){const v=parseStation(inputEl.value);if(v===null){inputEl.value=currentData.stationK;return;}let best=0,bestDiff=Infinity;for(let i=0;i<allSections.length;i++){const d=Math.abs(allSections[i].station-v);if(d<bestDiff){bestDiff=d;best=i;}}show(best);}
inputEl.addEventListener('focus',()=>inputEl.select());
inputEl.addEventListener('keydown',e=>{if(e.key==='Enter'){e.preventDefault();inputEl.blur();}});
inputEl.addEventListener('blur',jumpByInput);
document.getElementById('prevBtn').addEventListener('click',()=>show(currentIdx-1));
document.getElementById('nextBtn').addEventListener('click',()=>show(currentIdx+1));
document.getElementById('resetBtn').addEventListener('click',()=>fit(currentData));
window.addEventListener('keydown',e=>{if(e.target===inputEl)return;if(e.key==='ArrowLeft')show(currentIdx-1);else if(e.key==='ArrowRight')show(currentIdx+1);});

resizeCv();show(0);
</script>
</body>
</html>
`
