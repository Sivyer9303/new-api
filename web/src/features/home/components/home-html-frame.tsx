/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

/**
 * Renders admin-configured HomePageContent HTML in a sandboxed iframe.
 *
 * Isolated Shadow DOM + cloned app stylesheets fights self-contained pages
 * (custom <style>, particles, carousels) and can blank the home view. A blob
 * iframe matches the trusted-URL custom home path: own document, scripts
 * allowed, no host Tailwind leakage.
 *
 * Theme: the host posts `{ themeMode: 'light' | 'dark' }`. We inject a small
 * bridge so the iframe follows the app theme (critical for the overlaid public
 * header contrast), including CSS-variable remaps for the common `.ais` home
 * template and a few of its hardcoded light surfaces.
 *
 * Particles: custom home HTML often includes a static `#ais-fx` field of dots /
 * dashes. Inline page scripts are unreliable in some paste/save paths, so the
 * bridge owns a canvas particle runtime when `#ais-fx` is present.
 */
export function HomeHtmlFrame(props: {
  html: string
  theme?: 'light' | 'dark'
  onLoad?: () => void
  iframeRef?: React.RefObject<HTMLIFrameElement | null>
}) {
  const { t } = useTranslation()
  const [src, setSrc] = useState('')

  useEffect(() => {
    const documentHtml = wrapHomeHtmlDocument(props.html, props.theme)
    const blob = new Blob([documentHtml], { type: 'text/html' })
    const url = URL.createObjectURL(blob)
    setSrc(url)
    return () => {
      URL.revokeObjectURL(url)
    }
  }, [props.html, props.theme])

  if (!src) {
    return null
  }

  return (
    <iframe
      ref={props.iframeRef}
      src={src}
      className='h-screen w-full border-none'
      title={t('Custom Home Page')}
      // allow-same-origin: blob documents are generated here from admin HTML;
      // needed so the theme/particle bridge can measure layout and own #ais-fx.
      sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts allow-same-origin allow-top-navigation-by-user-activation'
      onLoad={props.onLoad}
    />
  )
}

const THEME_BRIDGE_STYLE = `
html,body{margin:0;padding:0;min-height:100%;}
html{background:#f7f9fc;color-scheme:light;}
html.dark,html[data-theme="dark"]{background:#0b1220;color-scheme:dark;}
html.dark body,html[data-theme="dark"] body{background:#0b1220;color:#e2e8f0;}

/* Remap .ais template tokens (AI Supply Station custom home) for dark theme */
html.dark .ais,
html[data-theme="dark"] .ais{
  --bg:#0b1220;
  --card:#111827;
  --text:#f1f5f9;
  --muted:#94a3b8;
  --faint:#64748b;
  --line:#334155;
  --line-soft:#1e293b;
  --blue:#60a5fa;
  --blue-deep:#93c5fd;
  --green:#34d399;
  --green-dot:#4ade80;
  --shadow:0 22px 56px rgba(0,0,0,.55);
  --particle-a:#93c5fd;
  --particle-b:#60a5fa;
  --particle-c:#38bdf8;
  --particle-d:#818cf8;
  --particle-e:#a5b4fc;
  background:var(--bg);
  color:var(--text);
}
html.dark .ais-brand img,
html[data-theme="dark"] .ais-brand img{
  background:#0f172a;
  box-shadow:0 0 0 1px rgba(96,165,250,.35);
}
html.dark .ais-demo,
html[data-theme="dark"] .ais-demo{
  background:rgba(17,24,39,.96);
  border-color:var(--line);
}
html.dark .ais-demo-head,
html[data-theme="dark"] .ais-demo-head{
  background:#111827;
  border-bottom-color:var(--line);
}
html.dark .ais-endpoint code,
html[data-theme="dark"] .ais-endpoint code{
  color:#cbd5e1;
}
html.dark .ais-block pre,
html[data-theme="dark"] .ais-block pre{
  background:#0f172a;
  border-color:var(--line);
  color:#cbd5e1;
}
html.dark .ais-meta,
html[data-theme="dark"] .ais-meta{
  background:#0f172a;
  border-top-color:var(--line);
}
html.dark .ais-step-dot,
html[data-theme="dark"] .ais-step-dot{
  box-shadow:0 0 0 6px rgba(37,99,235,.22);
}

/* Platform particle layer (canvas) — sits inside #ais-fx */
#ais-fx[data-ais-particles="platform"]{
  position:absolute;
  inset:0;
  pointer-events:none;
  z-index:0;
  overflow:hidden;
}
#ais-fx[data-ais-particles="platform"] canvas.ais-particle-canvas{
  position:absolute;
  inset:0;
  width:100%;
  height:100%;
  display:block;
  pointer-events:none;
}
`

const THEME_BRIDGE_SCRIPT = `
(function(){
  function applyTheme(mode){
    var dark = mode === 'dark';
    document.documentElement.classList.toggle('dark', dark);
    document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
    try { window.__aisRefreshParticles && window.__aisRefreshParticles(); } catch (e) {}
  }
  try {
    var prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    applyTheme(document.documentElement.getAttribute('data-theme') || (prefersDark ? 'dark' : 'light'));
  } catch (e) {}
  window.addEventListener('message', function(event){
    try {
      var data = event && event.data;
      if (!data || (data.themeMode !== 'dark' && data.themeMode !== 'light')) return;
      applyTheme(data.themeMode);
    } catch (e) {}
  });

  function bootParticles(){
    var host = document.getElementById('ais-fx');
    if (!host) return;
    var living =
      host.getAttribute('data-ais-particles') === 'platform' &&
      host.querySelector('canvas.ais-particle-canvas') &&
      window.__aisParticleEngineAlive;
    if (living) return;

    // Replace the node so pasted scripts holding an old #ais-fx reference
    // cannot wipe the canvas via innerHTML.
    var fx = host.cloneNode(false);
    fx.setAttribute('data-ais-particles', 'platform');
    if (!host.parentNode) return;
    host.parentNode.replaceChild(fx, host);

    var canvas = document.createElement('canvas');
    canvas.className = 'ais-particle-canvas';
    canvas.setAttribute('aria-hidden', 'true');
    fx.appendChild(canvas);
    var ctx = canvas.getContext('2d');
    if (!ctx) return;

    var particles = [];
    var w = 0;
    var h = 0;
    var dpr = 1;
    var mouse = { x: 0, y: 0, active: false };
    var attractRadius = 220;
    var linkDist = 118;
    var isDark = false;
    var reduceMotion = false;
    try {
      reduceMotion = !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
    } catch (e) {}
    var engineToken = Date.now();
    window.__aisParticleEngineAlive = engineToken;
    var pulse = 0;

    function themeColors(){
      var root = document.querySelector('.ais') || document.documentElement;
      var styles = getComputedStyle(root);
      isDark = document.documentElement.classList.contains('dark') ||
        document.documentElement.getAttribute('data-theme') === 'dark';
      var keys = ['--particle-a','--particle-b','--particle-c','--particle-d','--particle-e','--blue','--blue-deep'];
      var out = [];
      for (var i = 0; i < keys.length; i++) {
        var v = styles.getPropertyValue(keys[i]).trim();
        if (v) out.push(v);
      }
      if (!out.length) {
        out = isDark
          ? ['#93c5fd','#60a5fa','#38bdf8','#818cf8','#a5b4fc']
          : ['#2563eb','#3b82f6','#60a5fa','#38bdf8','#93c5fd'];
      }
      return out;
    }

    function targetCount(){
      // Denser field for a more "network / data" look.
      return Math.round(Math.min(420, Math.max(200, (w * h) / 4800)));
    }

    function resize(){
      var rect = fx.getBoundingClientRect();
      w = Math.max(1, Math.floor(rect.width || fx.clientWidth || window.innerWidth || 1));
      h = Math.max(1, Math.floor(rect.height || fx.clientHeight || window.innerHeight || 1));
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.floor(w * dpr);
      canvas.height = Math.floor(h * dpr);
      canvas.style.width = w + 'px';
      canvas.style.height = h + 'px';
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      attractRadius = Math.max(170, Math.min(280, Math.min(w, h) * 0.32));
      linkDist = Math.max(110, Math.min(160, Math.min(w, h) * 0.14));
      var want = targetCount();
      if (!particles.length || Math.abs(particles.length - want) > 40) spawn();
    }

    function pickKind(){
      var roll = Math.random();
      if (roll < 0.42) return 'dot';
      if (roll < 0.62) return 'ring';
      if (roll < 0.78) return 'dash';
      if (roll < 0.90) return 'cross';
      return 'diamond';
    }

    function spawn(){
      var colors = themeColors();
      var count = targetCount();
      particles = [];
      for (var i = 0; i < count; i++) {
        var kind = pickKind();
        var angle = Math.random() * Math.PI * 2;
        var speed = 0.28 + Math.random() * 0.9;
        var hub = Math.random() < 0.08;
        particles.push({
          x: Math.random() * w,
          y: Math.random() * h,
          vx: Math.cos(angle) * speed,
          vy: Math.sin(angle) * speed,
          kind: kind,
          size: hub ? 4 + Math.random() * 4 : 1.6 + Math.random() * 4.2,
          len: 10 + Math.random() * 28,
          rot: Math.random() * Math.PI * 2,
          spin: (Math.random() - 0.5) * 0.05,
          alpha: hub ? 0.55 + Math.random() * 0.35 : 0.28 + Math.random() * 0.5,
          color: colors[(Math.random() * colors.length) | 0],
          wander: 0.025 + Math.random() * 0.055,
          hub: hub,
          phase: Math.random() * Math.PI * 2
        });
      }
    }

    window.__aisRefreshParticles = function(){
      var colors = themeColors();
      for (var i = 0; i < particles.length; i++) {
        particles[i].color = colors[(Math.random() * colors.length) | 0];
      }
    };

    function onPointer(clientX, clientY){
      var rect = canvas.getBoundingClientRect();
      mouse.x = clientX - rect.left;
      mouse.y = clientY - rect.top;
      mouse.active = true;
    }

    document.addEventListener('pointermove', function(e){ onPointer(e.clientX, e.clientY); }, { passive: true, capture: true });
    document.addEventListener('mousemove', function(e){ onPointer(e.clientX, e.clientY); }, { passive: true, capture: true });
    document.addEventListener('pointerdown', function(e){ onPointer(e.clientX, e.clientY); }, { passive: true, capture: true });
    document.addEventListener('pointerleave', function(){ mouse.active = false; }, { passive: true });
    window.addEventListener('blur', function(){ mouse.active = false; });

    if (typeof ResizeObserver !== 'undefined') {
      var ro = new ResizeObserver(function(){ resize(); });
      ro.observe(fx);
      var root = document.querySelector('.ais');
      if (root) ro.observe(root);
    }
    window.addEventListener('resize', resize);

    function wrap(p){
      if (p.x < -30) p.x = w + 30;
      if (p.x > w + 30) p.x = -30;
      if (p.y < -30) p.y = h + 30;
      if (p.y > h + 30) p.y = -30;
    }

    function hexAlpha(color, a){
      // Accept #rgb/#rrggbb or fall back to rgba via globalAlpha.
      if (!color || color.charAt(0) !== '#' || (color.length !== 4 && color.length !== 7)) {
        return null;
      }
      var r, g, b;
      if (color.length === 4) {
        r = parseInt(color.charAt(1) + color.charAt(1), 16);
        g = parseInt(color.charAt(2) + color.charAt(2), 16);
        b = parseInt(color.charAt(3) + color.charAt(3), 16);
      } else {
        r = parseInt(color.slice(1, 3), 16);
        g = parseInt(color.slice(3, 5), 16);
        b = parseInt(color.slice(5, 7), 16);
      }
      return 'rgba(' + r + ',' + g + ',' + b + ',' + a + ')';
    }

    function drawLinks(){
      var maxLinks = reduceMotion ? 260 : 720;
      var drawn = 0;
      var baseLink = isDark ? 0.26 : 0.16;
      var n = particles.length;
      for (var i = 0; i < n && drawn < maxLinks; i++) {
        var a = particles[i];
        // Local window + strided far probes → denser constellation without O(n²).
        for (var step = 1; step <= 14 && drawn < maxLinks; step++) {
          var j = i + step;
          if (step > 8) j = i + 8 + (step - 8) * 5;
          if (j >= n) break;
          var b = particles[j];
          var dx = a.x - b.x;
          var dy = a.y - b.y;
          var dist = Math.sqrt(dx * dx + dy * dy);
          if (dist > linkDist || dist < 0.5) continue;
          var t = 1 - dist / linkDist;
          var boost = 0;
          if (mouse.active) {
            var mx = (a.x + b.x) * 0.5 - mouse.x;
            var my = (a.y + b.y) * 0.5 - mouse.y;
            var md = Math.sqrt(mx * mx + my * my);
            if (md < attractRadius) boost = (1 - md / attractRadius) * 0.42;
          }
          var alpha = (baseLink + t * 0.32 + boost) * (a.hub || b.hub ? 1.3 : 1);
          var stroke = hexAlpha(a.color, Math.min(0.6, alpha));
          ctx.beginPath();
          if (stroke) {
            ctx.strokeStyle = stroke;
            ctx.globalAlpha = 1;
          } else {
            ctx.strokeStyle = a.color;
            ctx.globalAlpha = Math.min(0.6, alpha);
          }
          ctx.lineWidth = a.hub || b.hub ? 1.2 : 0.75;
          ctx.moveTo(a.x, a.y);
          ctx.lineTo(b.x, b.y);
          ctx.stroke();
          drawn += 1;
        }
      }
      ctx.globalAlpha = 1;
    }

    function drawMouseField(){
      if (!mouse.active) return;
      var g = ctx.createRadialGradient(mouse.x, mouse.y, 0, mouse.x, mouse.y, attractRadius);
      if (isDark) {
        g.addColorStop(0, 'rgba(96,165,250,0.16)');
        g.addColorStop(0.45, 'rgba(56,189,248,0.06)');
        g.addColorStop(1, 'rgba(56,189,248,0)');
      } else {
        g.addColorStop(0, 'rgba(37,99,235,0.12)');
        g.addColorStop(0.45, 'rgba(59,130,246,0.05)');
        g.addColorStop(1, 'rgba(59,130,246,0)');
      }
      ctx.fillStyle = g;
      ctx.beginPath();
      ctx.arc(mouse.x, mouse.y, attractRadius, 0, Math.PI * 2);
      ctx.fill();
    }

    function draw(p){
      var twinkle = 0.85 + 0.15 * Math.sin(pulse * 2.4 + p.phase);
      var alpha = p.alpha * twinkle;
      if (mouse.active) {
        var mdx = mouse.x - p.x;
        var mdy = mouse.y - p.y;
        var md = Math.sqrt(mdx * mdx + mdy * mdy);
        if (md < attractRadius) alpha = Math.min(1, alpha + (1 - md / attractRadius) * 0.35);
      }
      ctx.globalAlpha = alpha;
      ctx.fillStyle = p.color;
      ctx.strokeStyle = p.color;

      if (p.hub) {
        var glow = hexAlpha(p.color, 0.12);
        if (glow) {
          ctx.fillStyle = glow;
          ctx.beginPath();
          ctx.arc(p.x, p.y, p.size * 2.8, 0, Math.PI * 2);
          ctx.fill();
          ctx.fillStyle = p.color;
        }
      }

      if (p.kind === 'dash') {
        ctx.save();
        ctx.translate(p.x, p.y);
        ctx.rotate(p.rot);
        ctx.lineWidth = 1.8;
        ctx.lineCap = 'round';
        ctx.beginPath();
        ctx.moveTo(-p.len / 2, 0);
        ctx.lineTo(p.len / 2, 0);
        ctx.stroke();
        // tiny bit tip
        ctx.globalAlpha = alpha * 0.9;
        ctx.beginPath();
        ctx.arc(p.len / 2, 0, 1.2, 0, Math.PI * 2);
        ctx.fill();
        ctx.restore();
      } else if (p.kind === 'ring') {
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.size * 0.75, 0, Math.PI * 2);
        ctx.stroke();
        if (p.hub) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, p.size * 0.28, 0, Math.PI * 2);
          ctx.fill();
        }
      } else if (p.kind === 'cross') {
        var arm = p.size * 1.1;
        ctx.lineWidth = 1.3;
        ctx.lineCap = 'square';
        ctx.beginPath();
        ctx.moveTo(p.x - arm, p.y);
        ctx.lineTo(p.x + arm, p.y);
        ctx.moveTo(p.x, p.y - arm);
        ctx.lineTo(p.x, p.y + arm);
        ctx.stroke();
      } else if (p.kind === 'diamond') {
        var s = p.size * 0.85;
        ctx.lineWidth = 1.2;
        ctx.beginPath();
        ctx.moveTo(p.x, p.y - s);
        ctx.lineTo(p.x + s, p.y);
        ctx.lineTo(p.x, p.y + s);
        ctx.lineTo(p.x - s, p.y);
        ctx.closePath();
        ctx.stroke();
      } else {
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.size * 0.48, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    var attractStrength = reduceMotion ? 0.09 : 0.17;
    var dampFar = 0.987;
    var dampNear = 0.9;
    var maxSpeed = reduceMotion ? 1.3 : 2.8;

    function tick(){
      if (window.__aisParticleEngineAlive !== engineToken) return;
      if (!canvas.isConnected) {
        window.__aisParticleEngineAlive = null;
        setTimeout(bootParticles, 0);
        return;
      }
      if (!w || !h) resize();
      pulse += 0.016;
      ctx.clearRect(0, 0, w, h);
      drawMouseField();

      for (var i = 0; i < particles.length; i++) {
        var p = particles[i];
        var near = false;
        p.vx += (Math.random() - 0.5) * p.wander;
        p.vy += (Math.random() - 0.5) * p.wander;
        // Mild orbital drift for hubs — more "system" than random dust.
        if (p.hub) {
          p.vx += Math.cos(pulse + p.phase) * 0.012;
          p.vy += Math.sin(pulse * 0.9 + p.phase) * 0.012;
        }
        if (mouse.active) {
          var dx = mouse.x - p.x;
          var dy = mouse.y - p.y;
          var dist = Math.sqrt(dx * dx + dy * dy) || 0.0001;
          if (dist < attractRadius) {
            near = true;
            var t = 1 - dist / attractRadius;
            var falloff = t * t * (3 - 2 * t);
            p.vx += (dx / dist) * attractStrength * falloff;
            p.vy += (dy / dist) * attractStrength * falloff;
            // Soft tangential swirl for a tech "field" feel.
            p.vx += (-dy / dist) * attractStrength * 0.35 * falloff;
            p.vy += (dx / dist) * attractStrength * 0.35 * falloff;
          }
        }
        var damp = near ? dampNear : dampFar;
        p.vx *= damp;
        p.vy *= damp;
        var speed = Math.sqrt(p.vx * p.vx + p.vy * p.vy);
        if (speed > maxSpeed) {
          p.vx = (p.vx / speed) * maxSpeed;
          p.vy = (p.vy / speed) * maxSpeed;
        }
        if (speed < 0.14) {
          var a = Math.random() * Math.PI * 2;
          p.vx += Math.cos(a) * 0.12;
          p.vy += Math.sin(a) * 0.12;
        }
        p.x += p.vx;
        p.y += p.vy;
        p.rot += p.spin + p.vx * 0.02;
        wrap(p);
      }

      drawLinks();
      for (var k = 0; k < particles.length; k++) draw(particles[k]);
      // Cursor links to nearby hubs/nodes
      if (mouse.active) {
        var linkColor = hexAlpha(isDark ? '#93c5fd' : '#3b82f6', 0.28) || 'rgba(59,130,246,0.28)';
        ctx.strokeStyle = linkColor;
        ctx.lineWidth = 0.8;
        var linked = 0;
        for (var m = 0; m < particles.length && linked < 14; m++) {
          var q = particles[m];
          var qdx = mouse.x - q.x;
          var qdy = mouse.y - q.y;
          var qd = Math.sqrt(qdx * qdx + qdy * qdy);
          if (qd < attractRadius * 0.72) {
            ctx.globalAlpha = (1 - qd / (attractRadius * 0.72)) * 0.55;
            ctx.beginPath();
            ctx.moveTo(mouse.x, mouse.y);
            ctx.lineTo(q.x, q.y);
            ctx.stroke();
            linked += 1;
          }
        }
      }
      ctx.globalAlpha = 1;
      requestAnimationFrame(tick);
    }

    resize();
    requestAnimationFrame(tick);
  }

  function scheduleBoot(){
    // Re-assert ownership against pasted particle scripts.
    setTimeout(bootParticles, 0);
    setTimeout(bootParticles, 50);
    setTimeout(bootParticles, 200);
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', scheduleBoot);
  } else {
    scheduleBoot();
  }
})();
`

function themeBridgeHead(): string {
  return `<style id="new-api-home-theme-bridge">${THEME_BRIDGE_STYLE}</style>
<script id="new-api-home-theme-bridge-script">${THEME_BRIDGE_SCRIPT}</script>`
}

function themeBridgeBodyTail(): string {
  // Ensure boot also runs if DOMContentLoaded already fired before head script... handled in script.
  // Extra empty marker helps debugging in view-source.
  return `<script id="new-api-home-particle-boot">/* particle boot registered in theme bridge */</script>`
}

function htmlOpenTag(initialTheme?: 'light' | 'dark'): string {
  if (initialTheme === 'dark') {
    return '<html lang="zh-CN" data-theme="dark" class="dark">'
  }
  if (initialTheme === 'light') {
    return '<html lang="zh-CN" data-theme="light">'
  }
  return '<html lang="zh-CN">'
}

export function wrapHomeHtmlDocument(
  html: string,
  initialTheme?: 'light' | 'dark'
): string {
  const trimmed = html.trim()
  const bridge = themeBridgeHead()
  const tail = themeBridgeBodyTail()

  if (/^<!doctype html/i.test(trimmed) || /^<html[\s>]/i.test(trimmed)) {
    let next = trimmed
    if (/<\/head>/i.test(next)) {
      next = next.replace(/<\/head>/i, `${bridge}</head>`)
    } else if (/<body[\s>]/i.test(next)) {
      next = next.replace(/<body([^>]*)>/i, `<body$1>${bridge}`)
    } else {
      next = `${bridge}${next}`
    }
    if (/<\/body>/i.test(next)) {
      next = next.replace(/<\/body>/i, `${tail}</body>`)
    } else {
      next = `${next}${tail}`
    }
    return next
  }

  return `<!DOCTYPE html>
${htmlOpenTag(initialTheme)}
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
${bridge}
</head>
<body>${html}${tail}</body>
</html>`
}
