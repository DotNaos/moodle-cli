package chatgptapp

const widgetURI = "ui://widget/moodle-browser-v3.html"
const resourceMimeType = "text/html;profile=mcp-app"
const widgetDomain = "https://moodle-services.vercel.app"

const widgetHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Moodle</title>
  <style>
    :root { color: #14171a; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    * { box-sizing: border-box; }
    body { margin: 0; background: #f6f7f9; }
    main { height: min(760px, 100vh); min-height: 420px; display: grid; grid-template-rows: auto 1fr; overflow: hidden; background: #f6f7f9; }
    header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 14px 16px; background: #fff; border-bottom: 1px solid #e5e8ec; }
    h1 { margin: 0; font-size: 15px; font-weight: 750; letter-spacing: 0; }
    .meta { color: #677281; font-size: 12px; }
    .content { overflow: auto; padding: 14px 16px 18px; }
    .grid { display: grid; gap: 10px; }
    .row { background: #fff; border: 1px solid #e5e8ec; border-radius: 8px; padding: 12px; }
    .row-title { font-size: 14px; font-weight: 650; margin: 0 0 5px; }
    .row-sub { color: #677281; font-size: 12px; margin: 0; }
    .toolbar { display: flex; align-items: center; gap: 8px; position: sticky; top: -14px; z-index: 5; margin: -14px -16px 12px; padding: 10px 16px; background: rgba(246,247,249,.96); border-bottom: 1px solid #e5e8ec; backdrop-filter: blur(10px); }
    button { border: 0; border-radius: 999px; padding: 8px 12px; color: white; background: #1f6feb; font-size: 12px; font-weight: 750; cursor: pointer; }
    button.secondary { color: #1f2937; background: #e8eef8; }
    button:disabled { opacity: .45; cursor: not-allowed; }
    .spacer { flex: 1; }
    .pdf-pages { display: grid; gap: 16px; justify-items: center; }
    .pdf-page { width: min(100%, 980px); background: #fff; border: 1px solid #dfe4ea; border-radius: 8px; box-shadow: 0 12px 32px rgba(20,23,26,.10); padding: 8px; }
    .pdf-page canvas { width: 100%; height: auto; display: block; border-radius: 4px; background: #fff; }
    .page-label { color: #677281; font-size: 11px; padding: 4px 2px 0; }
    .empty { color: #677281; padding: 20px 4px; }
    pre { white-space: pre-wrap; overflow-wrap: anywhere; margin: 0; font: 13px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1 id="title">Moodle</h1>
        <div class="meta" id="meta">Waiting for data</div>
      </div>
    </header>
    <section class="content" id="content"></section>
  </main>
  <script type="module">
    const titleEl = document.getElementById("title");
    const metaEl = document.getElementById("meta");
    const contentEl = document.getElementById("content");
    let pdfjsLib = null;
    let activePDF = null;
    let currentPage = 1;
    let updateTimer = 0;
    let saveTimer = 0;

    async function loadPDFJS() {
      if (pdfjsLib) return pdfjsLib;
      pdfjsLib = await import("https://cdn.jsdelivr.net/npm/pdfjs-dist@4.10.38/build/pdf.mjs");
      pdfjsLib.GlobalWorkerOptions.workerSrc = "https://cdn.jsdelivr.net/npm/pdfjs-dist@4.10.38/build/pdf.worker.mjs";
      return pdfjsLib;
    }

    function renderToolPayload(payload, metadata) {
      const data = payload?.structuredContent ?? payload;
      const meta = payload?._meta ?? metadata ?? window.openai?.toolResponseMetadata ?? {};
      render(data, meta);
    }

    function render(data, metadata) {
      if (!data) return false;
      if (data.viewer?.fileType === "pdf") {
        renderEmbeddedPDF(data.viewer, metadata?.pdfUrl);
        return true;
      }
      if (Array.isArray(data.courses)) return renderCourses(data.courses);
      if (Array.isArray(data.materials)) return renderMaterials(data.materials);
      if (Array.isArray(data.events)) return renderEvents(data.events);
      if (data.document) return renderDocument(data.document);
      return renderDocument({ title: "Moodle result", text: JSON.stringify(data, null, 2) });
    }

    function renderCourses(courses) {
      titleEl.textContent = "Moodle courses";
      metaEl.textContent = courses.length + " courses";
      contentEl.innerHTML = '<div class="grid">' + courses.map(course =>
        '<article class="row"><p class="row-title">' + escapeHTML(course.fullname || course.shortname || "Course") +
        '</p><p class="row-sub">' + escapeHTML([course.shortname, course.category].filter(Boolean).join(" · ")) + '</p></article>'
      ).join("") + "</div>";
      return true;
    }

    function renderMaterials(materials) {
      titleEl.textContent = "Course materials";
      metaEl.textContent = materials.length + " items";
      contentEl.innerHTML = '<div class="grid">' + materials.map(item =>
        '<article class="row"><p class="row-title">' + escapeHTML(item.name || "Material") +
        '</p><p class="row-sub">' + escapeHTML([item.sectionName, item.fileType || item.type].filter(Boolean).join(" · ")) + '</p></article>'
      ).join("") + "</div>";
      return true;
    }

    function renderEvents(events) {
      titleEl.textContent = "School calendar";
      metaEl.textContent = events.length + " events";
      contentEl.innerHTML = '<div class="grid">' + events.map(event =>
        '<article class="row"><p class="row-title">' + escapeHTML(event.summary || "Calendar event") +
        '</p><p class="row-sub">' + escapeHTML(formatDate(event.start) + (event.location ? " · " + event.location : "")) + '</p></article>'
      ).join("") + "</div>";
      return true;
    }

    function renderDocument(document) {
      titleEl.textContent = document.title || "Document";
      metaEl.textContent = document.metadata?.fileType ? document.metadata.fileType.toUpperCase() : "Text";
      contentEl.innerHTML = '<article class="row"><pre>' + escapeHTML(document.text || "") + "</pre></article>";
      return true;
    }

    async function renderEmbeddedPDF(viewer, pdfUrl) {
      titleEl.textContent = viewer.title || "PDF";
      metaEl.textContent = "Loading PDF";
      currentPage = Math.max(1, Number(viewer.target?.page || 1));
      activePDF = { viewer, pdfUrl, pageCount: 0 };
      contentEl.innerHTML = toolbarHTML() + '<div class="empty">Loading embedded PDF...</div>';
      wireToolbar();
      if (!pdfUrl) {
        contentEl.innerHTML = toolbarHTML() + '<div class="empty">PDF URL is missing.</div>';
        wireToolbar();
        return;
      }
      try {
        const lib = await loadPDFJS();
        const pdf = await lib.getDocument({ url: pdfUrl }).promise;
        activePDF.document = pdf;
        activePDF.pageCount = pdf.numPages;
        metaEl.textContent = pdf.numPages + " pages";
        contentEl.innerHTML = toolbarHTML() + '<div class="pdf-pages" id="pdf-pages"></div>';
        wireToolbar();
        await renderAllPages(pdf);
        await scrollToTarget(viewer.target || {});
        reportState();
      } catch (error) {
        contentEl.innerHTML = toolbarHTML() + '<div class="empty">The PDF could not be rendered: ' + escapeHTML(error.message || String(error)) + "</div>";
        wireToolbar();
        metaEl.textContent = "PDF render failed";
      }
    }

    function toolbarHTML() {
      const total = activePDF?.pageCount || "?";
      return '<div class="toolbar"><button id="prev" class="secondary">Previous</button><span class="meta" id="page-state">Page ' +
        currentPage + " of " + total + '</span><button id="next">Next</button><span class="spacer"></span><button id="full" class="secondary">Fullscreen</button></div>';
    }

    function wireToolbar() {
      const prev = document.getElementById("prev");
      const next = document.getElementById("next");
      const full = document.getElementById("full");
      if (prev) prev.onclick = () => scrollToPage(Math.max(1, currentPage - 1));
      if (next) next.onclick = () => scrollToPage(Math.min(activePDF?.pageCount || currentPage + 1, currentPage + 1));
      if (full) full.onclick = () => window.openai?.requestDisplayMode?.({ mode: "fullscreen" });
      updatePageState();
    }

    async function renderAllPages(pdf) {
      const holder = document.getElementById("pdf-pages");
      const width = Math.min(contentEl.clientWidth - 36, 980);
      for (let pageNo = 1; pageNo <= pdf.numPages; pageNo++) {
        const page = await pdf.getPage(pageNo);
        const viewport = page.getViewport({ scale: 1 });
        const scale = Math.max(.8, Math.min(1.8, width / viewport.width));
        const scaled = page.getViewport({ scale });
        const article = document.createElement("article");
        article.className = "pdf-page";
        article.id = "page-" + pageNo;
        article.dataset.page = String(pageNo);
        const canvas = document.createElement("canvas");
        const context = canvas.getContext("2d");
        canvas.width = Math.floor(scaled.width);
        canvas.height = Math.floor(scaled.height);
        article.appendChild(canvas);
        const label = document.createElement("div");
        label.className = "page-label";
        label.textContent = "Page " + pageNo;
        article.appendChild(label);
        holder.appendChild(article);
        await page.render({ canvasContext: context, viewport: scaled }).promise;
      }
    }

    async function scrollToTarget(target) {
      if (target.query) {
        const page = await findTextPage(String(target.query));
        if (page) {
          scrollToPage(page);
          return;
        }
      }
      scrollToPage(Math.min(Math.max(1, Number(target.page || 1)), activePDF?.pageCount || 1));
    }

    async function findTextPage(query) {
      if (!activePDF?.document || !query.trim()) return 0;
      const needle = query.trim().toLowerCase();
      for (let pageNo = 1; pageNo <= activePDF.document.numPages; pageNo++) {
        const page = await activePDF.document.getPage(pageNo);
        const text = await page.getTextContent();
        const haystack = text.items.map(item => item.str || "").join(" ").toLowerCase();
        if (haystack.includes(needle)) return pageNo;
      }
      return 0;
    }

    function scrollToPage(pageNo) {
      const page = document.getElementById("page-" + pageNo);
      if (!page) return;
      currentPage = pageNo;
      page.scrollIntoView({ behavior: "smooth", block: "start" });
      updatePageState();
      reportState();
    }

    contentEl.addEventListener("scroll", () => {
      if (!activePDF) return;
      const pages = Array.from(document.querySelectorAll(".pdf-page"));
      let best = currentPage;
      let bestDistance = Infinity;
      for (const page of pages) {
        const distance = Math.abs(page.getBoundingClientRect().top - contentEl.getBoundingClientRect().top);
        if (distance < bestDistance) {
          bestDistance = distance;
          best = Number(page.dataset.page);
        }
      }
      if (best !== currentPage) {
        currentPage = best;
        updatePageState();
        window.clearTimeout(updateTimer);
        updateTimer = window.setTimeout(reportState, 250);
      }
    }, { passive: true });

    function updatePageState() {
      const state = document.getElementById("page-state");
      if (state) state.textContent = "Page " + currentPage + " of " + (activePDF?.pageCount || "?");
    }

    function reportState() {
      if (!activePDF) return;
      const text = "Moodle PDF open: " + (activePDF.viewer.title || "PDF") + ". Current visible page: " +
        currentPage + " of " + (activePDF.pageCount || "?") + ".";
      queueSaveState();
      if (window.openai?.updateModelContext) {
        window.openai.updateModelContext(text);
        return;
      }
      window.parent.postMessage({ jsonrpc: "2.0", id: "pdf-state-" + Date.now(), method: "ui/update-model-context", params: { text } }, "*");
    }

    function queueSaveState() {
      window.clearTimeout(saveTimer);
      saveTimer = window.setTimeout(saveState, 350);
    }

    async function saveState() {
      if (!activePDF || !window.openai?.callTool) return;
      const screenshotDataURL = captureCurrentPage();
      const payload = {
        title: activePDF.viewer.title || "PDF",
        courseId: activePDF.viewer.courseId || "",
        resourceId: activePDF.viewer.resourceId || "",
        page: currentPage,
        pageCount: activePDF.pageCount || 1,
        screenshotDataURL
      };
      try {
        await window.openai.callTool("save_pdf_view_state", payload);
      } catch (error) {
        try {
          await window.openai.callTool({ name: "save_pdf_view_state", arguments: payload });
        } catch (_) {}
      }
    }

    function captureCurrentPage() {
      const canvas = document.querySelector('#page-' + currentPage + ' canvas');
      if (!canvas) return "";
      const maxWidth = 900;
      const scale = Math.min(1, maxWidth / canvas.width);
      if (scale >= 1) return canvas.toDataURL("image/jpeg", .72);
      const thumbnail = document.createElement("canvas");
      thumbnail.width = Math.floor(canvas.width * scale);
      thumbnail.height = Math.floor(canvas.height * scale);
      const context = thumbnail.getContext("2d");
      context.drawImage(canvas, 0, 0, thumbnail.width, thumbnail.height);
      return thumbnail.toDataURL("image/jpeg", .72);
    }

    function escapeHTML(value) {
      return String(value ?? "").replace(/[&<>"']/g, char => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char]));
    }

    function formatDate(value) {
      const date = new Date(value);
      return Number.isNaN(date.getTime()) ? "" : date.toLocaleString();
    }

    window.addEventListener("message", event => {
      if (event.source !== window.parent) return;
      const message = event.data;
      if (!message || message.jsonrpc !== "2.0") return;
      if (message.method === "ui/notifications/tool-result") renderToolPayload(message.params);
    }, { passive: true });

    window.addEventListener("openai:set_globals", event => {
      const globals = event.detail?.globals || {};
      renderToolPayload(globals.toolOutput, globals.toolResponseMetadata);
    }, { passive: true });

    const tryInitialRender = () => renderToolPayload(window.openai?.toolOutput, window.openai?.toolResponseMetadata);
    tryInitialRender();
    setTimeout(tryInitialRender, 50);
    setTimeout(tryInitialRender, 250);
    setTimeout(tryInitialRender, 1000);
  </script>
</body>
</html>`
