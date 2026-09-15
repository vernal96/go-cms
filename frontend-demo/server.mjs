import http from "node:http";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const backend = new URL(process.env.DEMO_API_TARGET || "http://localhost:8080");
const domain = process.env.DEMO_DOMAIN || "demo.localhost";
const port = Number(process.env.PORT || 4173);
const escape = (value) =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const menuConfigPath = process.env.DEMO_MENU_CONFIG || "/data/menu-config.json";
const safeURL = (value) => typeof value === "string" && (/^\/(?!\/)/.test(value) || /^https?:\/\//i.test(value)) ? value : "#";
function menuLinks(items, path) {
  return items.map(item => `<a href="${escape(safeURL(item.url))}" ${path === item.url || (item.url !== "/" && path.startsWith(item.url + "/")) ? 'aria-current="page"' : ""}>${escape(item.title)}</a>${item.children?.length ? `<span class="menu-children">${menuLinks(item.children, path)}</span>` : ""}`).join("");
}
async function loadMenus(path) {
  const config = JSON.parse(await readFile(menuConfigPath, "utf8"));
  if (![config.projects, config.footer1, config.footer2].every(id => Number.isSafeInteger(id) && id > 0)) throw new Error("Invalid menu configuration");
  const paths = ["/menu?depth=1", `/menu?parent_id=${config.footer1}`, `/menu?parent_id=${config.footer2}`];
  if (path === "/projects" || path.startsWith("/projects/")) paths.push(`/menu?parent_id=${config.projects}`);
  const responses = await Promise.all(paths.map(publicData));
  if (responses.some(r => r.status !== 200 || !Array.isArray(r.data?.items))) throw new Error("Menu unavailable");
  return responses.map(r => r.data.items);
}
const assets = new Map(
  await Promise.all(
    ["style.css", "house.svg", "courtyard.svg", "interior.svg"].map(
      async (name) => [
        name,
        await readFile(new URL(`./public/${name}`, import.meta.url)),
      ],
    ),
  ),
);

// Only anonymous public requests. Management credentials are never used by the website.
function publicData(path) {
  return new Promise((resolve, reject) => {
    const request = http.get(
      {
        hostname: backend.hostname,
        port: backend.port || 80,
        path,
        headers: { Host: domain, Accept: "application/json" },
      },
      (response) => {
        let body = "";
        response.setEncoding("utf8");
        response.on("data", (chunk) => {
          body += chunk;
        });
        response.on("end", () => {
          if (response.statusCode !== 200)
            return resolve({ status: response.statusCode });
          try {
            resolve({ status: 200, data: JSON.parse(body) });
          } catch {
            reject(new Error("Invalid CMS response"));
          }
        });
        response.on("error", reject);
      },
    );
    request.setTimeout(10000, () => request.destroy(new Error("CMS timeout")));
    request.on("error", reject);
  });
}
async function loadChrome(path) {
  const settings = await loadSite();
  return [await loadMenus(path), settings];
}
async function loadSite() {
  const result = await publicData('/site');
  const settings = result.data?.settings;
  if (result.status !== 200 || !settings || typeof settings !== 'object' || Array.isArray(settings) || typeof settings.string_value !== 'string') throw new Error('Site settings unavailable');
  return settings;
}
function layout(title, description, content, path, menus = [[], [], [], []], settings = {}) {
  const name = typeof settings.string_value === 'string' ? settings.string_value : 'CMS';
  const email = typeof settings.email_value === 'string' ? settings.email_value : '';
  const phone = typeof settings.phone_value === 'string' ? settings.phone_value : '';
  const contacts = `${email ? `<a href="mailto:${escape(encodeURIComponent(email))}">${escape(email)}</a><br>` : ''}${phone ? `<a href="tel:${escape(phone.replace(/[^+0-9]/g, ''))}">${escape(phone)}</a><br>` : ''}`;
  return `<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escape(title)} — ${escape(name)}</title><meta name="description" content="${escape(description)}"><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/style.css"></head><body><a class="skip" href="#main">К содержимому</a><header class="header"><a class="brand" href="/" aria-label="${escape(name)} — главная"><span class="brand-icon">${escape(name[0] ?? "")}</span>${escape(name)}<span class="brand-caption">архитектурная<br>студия</span></a><nav aria-label="Основная навигация">${menuLinks(menus[0], path)}</nav><a class="contact-link" href="/contacts">Обсудим идею <span>↗</span></a></header><main id="main">${menus[3]?.length ? `<nav class="project-menu" aria-label="Меню проектов">${menuLinks(menus[3], path)}</nav>` : ""}${content}</main><footer><div><a class="brand" href="/">${escape(name)}<span class="dot">®</span></a><p>Архитектура начинается с внимания.</p></div><nav aria-label="Футер 1">${menuLinks(menus[1], path)}</nav><nav aria-label="Футер 2">${menuLinks(menus[2], path)}</nav><div class="footer-note">${contacts}Демонстрационная студия<br>Все проекты и истории вымышлены.<br>© ${escape(name)}, 2026</div></footer></body></html>`;
}
function searchForm(q = "") {
  return `<form class="search" method="get" action="/search"><label for="q">Найти проект или статью</label><div><input id="q" name="q" type="search" minlength="3" maxlength="200" required value="${escape(q)}" placeholder="Например, свет или дом"><button type="submit">Найти ↗</button></div></form>`;
}
export const server = http.createServer(async (req, res) => {
  const send = (status, body, type = "text/html; charset=utf-8") => {
    res.writeHead(status, {
      "Content-Type": type,
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
      "Content-Security-Policy":
        "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'none'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'",
    });
    res.end(req.method === "HEAD" ? undefined : body);
  };
  if (!["GET", "HEAD"].includes(req.method)) {
    res.setHeader("Allow", "GET, HEAD");
    return send(405, "Method not allowed", "text/plain");
  }
  const url = new URL(req.url, "http://demo.localhost");
  if (url.pathname === "/healthz") return send(200, "ok", "text/plain");
  if (url.pathname.startsWith("/assets/")) {
    const name = url.pathname.slice(8);
    return assets.has(name)
      ? send(
          200,
          assets.get(name),
          name.endsWith(".css") ? "text/css; charset=utf-8" : "image/svg+xml",
        )
      : send(404, "Not found", "text/plain");
  }
  if (/^\/(api|_cms)(\/|$)/.test(url.pathname))
    return send(404, "Not found", "text/plain");
  try {
    if (url.pathname === "/search") {
      const q = (url.searchParams.get("q") || "").trim();
      let content =
        '<section class="page-head"><p class="eyebrow">Навигация по идеям</p><h1>Найти своё.</h1></section>' +
        searchForm(q);
      let status = 200;
      if (q) {
        if ([...q].length < 3 || [...q].length > 200) {
          status = 400;
          content += '<p class="notice">Введите от 3 до 200 символов.</p>';
        } else {
          const page = url.searchParams.get("page") || "1";
          const result = await publicData(
            `/search?q=${encodeURIComponent(q)}&page=${encodeURIComponent(page)}&per_page=10`,
          );
          if (result.status !== 200) {
            const error = new Error("Search failed");
            error.status = result.status;
            throw error;
          }
          content += `<section class="results"><p>Найдено: ${result.data.pagination.total}</p>${result.data.items.map((item) => `<article><h2><a href="${escape(item.url.startsWith("/") && !item.url.startsWith("//") ? item.url : "/")}">${escape(item.title)} ↗</a></h2><p>${escape(item.annotation)}</p></article>`).join("") || "<p>Совпадений нет. Попробуйте другой запрос.</p>"}`;
          const pagination = result.data.pagination;
          for (const [n, label] of [
            [pagination.page - 1, "← Назад"],
            [pagination.page + 1, "Далее →"],
          ]) {
            if (n > 0 && (n - 1) * pagination.per_page < pagination.total)
              content += `<a class="button secondary" href="/search?q=${encodeURIComponent(q)}&page=${n}">${label}</a> `;
          }
          content += "</section>";
        }
      }
      return send(
        status,
        layout(
          "Поиск",
          "Поиск по проектам и журналу студии.",
          content,
          url.pathname,
          ...await loadChrome(url.pathname),
        ),
      );
    }
    const result = await publicData(url.pathname);
    if (result.status !== 200) {
      const error = new Error("Page unavailable");
      error.status = result.status;
      throw error;
    }
    if (!result.data.resource) throw new Error("Missing resource");
    const item = result.data.resource;
    const contentWidget = result.data.widgets?.body?.find(widget => widget.code === "core_content");
    if (contentWidget?.error || typeof contentWidget?.data?.content !== "string") throw new Error("Content widget unavailable");
    // HTML is authored by trusted CMS editors; CSP prevents scripts from running.
    const content =
      item.content_type === "html"
        ? contentWidget.data.content
        : `<section class="prose"><h1>${escape(item.title)}</h1><p>${escape(contentWidget.data.content)}</p></section>`;
    send(200, layout(item.title, item.annotation, content, url.pathname, ...await loadChrome(url.pathname)));
  } catch (error) {
    const status = [400, 403, 404].includes(error.status) ? error.status : 502;
    const title =
      status === 404
        ? "Здесь пока пусто."
        : status === 403
          ? "Страница закрыта."
          : status === 400
            ? "Проверьте запрос."
            : "Сайт временно недоступен.";
    if (status === 502) console.error("CMS request failed:", error.message);
    send(
      status,
      layout(
        title,
        "",
        `<section class="page-head"><p class="eyebrow">${status}</p><h1>${title}</h1><p>Вернитесь на главную или попробуйте позднее.</p><a class="button" href="/">На главную ↗</a></section>`,
        url.pathname,
      ),
    );
  }
});
if (process.argv[1] === fileURLToPath(import.meta.url))
  server.listen(port, "0.0.0.0", () =>
    console.log(`Demo: http://localhost:${port} (CMS site ${domain})`),
  );
