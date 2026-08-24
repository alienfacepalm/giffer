/* Magical GIF-forge animation (Three.js). Exposes window.GifferForge. */
(() => {
  const ACCENT = 0xff4d1a;
  const AMBER = 0xffb347;
  const MIST = 0xa8c0d4;
  const CORE = 0xffe8d6;

  let renderer = null;
  let scene = null;
  let camera = null;
  let animId = 0;
  let ro = null;
  let mount = null;
  let caption = null;
  let running = false;
  let progress = 0;
  let t0 = 0;
  let ownsCanvas = false;

  let core = null;
  let wire = null;
  let ring = null;
  let sparks = null;
  let dust = null;
  let ribbons = [];

  function disposeObject(obj) {
    if (!obj) return;
    obj.traverse((child) => {
      if (child.geometry) child.geometry.dispose();
      if (child.material) {
        if (Array.isArray(child.material)) child.material.forEach((m) => m.dispose());
        else child.material.dispose();
      }
    });
  }

  function makeSparkTexture() {
    const size = 64;
    const c = document.createElement("canvas");
    c.width = size;
    c.height = size;
    const ctx = c.getContext("2d");
    const g = ctx.createRadialGradient(32, 32, 0, 32, 32, 32);
    g.addColorStop(0, "rgba(255,255,255,1)");
    g.addColorStop(0.25, "rgba(255,180,120,0.85)");
    g.addColorStop(0.55, "rgba(255,77,26,0.35)");
    g.addColorStop(1, "rgba(0,0,0,0)");
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, size, size);
    const tex = new THREE.CanvasTexture(c);
    tex.needsUpdate = true;
    return tex;
  }

  function buildRibbons(parent) {
    const out = [];
    for (let i = 0; i < 3; i++) {
      const pts = [];
      const turns = 2.2 + i * 0.35;
      const radius = 1.15 + i * 0.22;
      for (let j = 0; j <= 80; j++) {
        const u = j / 80;
        const a = u * Math.PI * 2 * turns + i * 1.7;
        pts.push(
          new THREE.Vector3(
            Math.cos(a) * radius * (0.7 + 0.3 * Math.sin(u * Math.PI * 3)),
            (u - 0.5) * 2.4,
            Math.sin(a) * radius * (0.7 + 0.3 * Math.cos(u * Math.PI * 2))
          )
        );
      }
      const curve = new THREE.CatmullRomCurve3(pts);
      const geo = new THREE.TubeGeometry(curve, 120, 0.018 + i * 0.006, 6, false);
      const mat = new THREE.MeshBasicMaterial({
        color: i === 0 ? ACCENT : i === 1 ? AMBER : MIST,
        transparent: true,
        opacity: 0.55,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
      });
      const mesh = new THREE.Mesh(geo, mat);
      mesh.userData.spin = 0.35 + i * 0.18;
      parent.add(mesh);
      out.push(mesh);
    }
    return out;
  }

  function buildSparks(count, spread, size, color) {
    const positions = new Float32Array(count * 3);
    const seeds = new Float32Array(count);
    for (let i = 0; i < count; i++) {
      const r = 0.4 + Math.random() * spread;
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      positions[i * 3] = r * Math.sin(phi) * Math.cos(theta);
      positions[i * 3 + 1] = r * Math.cos(phi);
      positions[i * 3 + 2] = r * Math.sin(phi) * Math.sin(theta);
      seeds[i] = Math.random() * Math.PI * 2;
    }
    const geo = new THREE.BufferGeometry();
    geo.setAttribute("position", new THREE.BufferAttribute(positions, 3));
    geo.setAttribute("seed", new THREE.BufferAttribute(seeds, 1));
    const mat = new THREE.PointsMaterial({
      color,
      map: makeSparkTexture(),
      size,
      transparent: true,
      opacity: 0.9,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      sizeAttenuation: true,
    });
    return new THREE.Points(geo, mat);
  }

  function resize() {
    if (!renderer || !camera || !mount) return;
    const target = renderer.domElement || mount;
    const w = Math.max(1, target.clientWidth || mount.clientWidth);
    const h = Math.max(1, target.clientHeight || mount.clientHeight);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setSize(w, h, false);
  }

  function tick(now) {
    if (!running) return;
    animId = requestAnimationFrame(tick);
    const t = (now - t0) * 0.001;
    const energy = 0.55 + progress * 0.45;

    if (core) {
      core.rotation.y = t * 0.55 * energy;
      core.rotation.x = Math.sin(t * 0.7) * 0.25;
      const pulse = 1 + Math.sin(t * 2.4) * 0.04 + progress * 0.12;
      core.scale.setScalar(pulse);
      if (core.material && core.material.emissiveIntensity !== undefined) {
        core.material.emissiveIntensity = 0.55 + Math.sin(t * 3) * 0.15 + progress * 0.5;
      }
    }
    if (wire) {
      wire.rotation.y = -t * 0.35 * energy;
      wire.rotation.z = t * 0.2;
    }
    if (ring) {
      ring.rotation.z = t * 0.8 * energy;
      ring.rotation.x = Math.PI / 2 + Math.sin(t * 0.5) * 0.15;
      ring.scale.setScalar(1 + progress * 0.25 + Math.sin(t * 2) * 0.03);
    }
    ribbons.forEach((r, i) => {
      r.rotation.y = t * r.userData.spin * energy;
      r.rotation.x = Math.sin(t * 0.4 + i) * 0.2;
      if (r.material) r.material.opacity = 0.35 + progress * 0.4 + Math.sin(t * 2 + i) * 0.08;
    });

    [sparks, dust].forEach((cloud, ci) => {
      if (!cloud) return;
      cloud.rotation.y = t * (0.15 + ci * 0.08) * energy;
      const pos = cloud.geometry.attributes.position;
      const seeds = cloud.geometry.attributes.seed;
      for (let i = 0; i < pos.count; i++) {
        const seed = seeds.getX(i);
        const wobble = Math.sin(t * (1.2 + ci) + seed) * 0.012 * energy;
        pos.setY(i, pos.getY(i) + wobble * 0.15);
      }
      pos.needsUpdate = true;
      if (cloud.material) {
        cloud.material.size = (ci === 0 ? 0.12 : 0.06) * (1 + progress * 0.5);
        cloud.material.opacity = 0.55 + progress * 0.35;
      }
    });

    if (camera) {
      camera.position.x = Math.sin(t * 0.25) * 0.35;
      camera.position.y = 0.15 + Math.sin(t * 0.33) * 0.12;
      camera.lookAt(0, 0, 0);
    }
    renderer.render(scene, camera);
  }

  function setCaption(text) {
    if (caption) caption.textContent = text;
  }

  function start(mountEl, opts) {
    if (!window.THREE) {
      console.warn("GifferForge: THREE missing");
      return;
    }
    stop();
    mount = mountEl;
    caption = opts && opts.captionEl ? opts.captionEl : null;
    progress = Math.max(0, Math.min(1, (opts && opts.progress) || 0));
    t0 = performance.now();

    const canvas = opts && opts.canvas ? opts.canvas : null;
    ownsCanvas = !canvas;
    const sizeEl = canvas || mount;
    const w = Math.max(1, sizeEl.clientWidth || mount.clientWidth);
    const h = Math.max(1, sizeEl.clientHeight || mount.clientHeight);

    scene = new THREE.Scene();
    scene.fog = new THREE.FogExp2(0x0a0c0f, 0.085);

    camera = new THREE.PerspectiveCamera(42, w / h, 0.1, 40);
    camera.position.set(0, 0.2, 5.2);

    renderer = new THREE.WebGLRenderer({
      canvas: canvas || undefined,
      antialias: true,
      alpha: true,
      powerPreference: "high-performance",
    });
    renderer.setClearColor(0x000000, 0);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setSize(w, h, false);
    if (ownsCanvas) {
      renderer.domElement.className = "forge-canvas";
      mount.appendChild(renderer.domElement);
    }

    const root = new THREE.Group();
    scene.add(root);

    scene.add(new THREE.AmbientLight(0xffffff, 0.35));
    const key = new THREE.PointLight(ACCENT, 2.2, 12);
    key.position.set(2.2, 2.4, 3);
    scene.add(key);
    const fill = new THREE.PointLight(MIST, 1.1, 10);
    fill.position.set(-2.5, -1.2, 2);
    scene.add(fill);

    core = new THREE.Mesh(
      new THREE.IcosahedronGeometry(0.72, 2),
      new THREE.MeshStandardMaterial({
        color: CORE,
        emissive: ACCENT,
        emissiveIntensity: 0.7,
        metalness: 0.35,
        roughness: 0.25,
        flatShading: true,
      })
    );
    root.add(core);

    wire = new THREE.Mesh(
      new THREE.IcosahedronGeometry(0.95, 1),
      new THREE.MeshBasicMaterial({
        color: AMBER,
        wireframe: true,
        transparent: true,
        opacity: 0.35,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
      })
    );
    root.add(wire);

    ring = new THREE.Mesh(
      new THREE.TorusGeometry(1.45, 0.03, 12, 96),
      new THREE.MeshBasicMaterial({
        color: ACCENT,
        transparent: true,
        opacity: 0.7,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
      })
    );
    ring.rotation.x = Math.PI / 2;
    root.add(ring);

    ribbons = buildRibbons(root);
    sparks = buildSparks(140, 2.2, 0.12, ACCENT);
    dust = buildSparks(220, 3.2, 0.05, MIST);
    root.add(sparks);
    root.add(dust);

    if (typeof ResizeObserver !== "undefined") {
      ro = new ResizeObserver(() => resize());
      ro.observe(mount);
    } else {
      window.addEventListener("resize", resize);
    }

    running = true;
    mount.classList.add("is-live");
    setCaption("Conjuring frames…");
    animId = requestAnimationFrame(tick);
  }

  function setProgress(pct, label) {
    progress = Math.max(0, Math.min(1, (pct || 0) / 100));
    if (label) setCaption(label);
    else if (progress >= 0.95) setCaption("Sealing the GIF…");
    else if (progress >= 0.55) setCaption("Weaving motion…");
    else if (progress >= 0.15) setCaption("Gathering frames…");
    else setCaption("Conjuring frames…");
  }

  function stop() {
    running = false;
    if (animId) {
      cancelAnimationFrame(animId);
      animId = 0;
    }
    if (ro) {
      ro.disconnect();
      ro = null;
    } else {
      window.removeEventListener("resize", resize);
    }
    if (scene) disposeObject(scene);
    if (renderer) {
      const el = renderer.domElement;
      renderer.dispose();
      if (ownsCanvas && el && el.parentNode) el.parentNode.removeChild(el);
    }
    renderer = null;
    scene = null;
    camera = null;
    core = wire = ring = sparks = dust = null;
    ribbons = [];
    if (mount) mount.classList.remove("is-live");
    mount = null;
    caption = null;
    progress = 0;
    ownsCanvas = false;
  }

  window.GifferForge = { start, stop, setProgress };
})();
