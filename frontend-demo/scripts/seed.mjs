import { pages } from "./content.mjs";
import { writeFile, rename } from "node:fs/promises";
const base = process.env.DEMO_API_TARGET || "http://localhost:8080";
const domain = process.env.DEMO_DOMAIN || "demo.localhost";
let token;
async function api(path, method = "GET", data) {
  const response = await fetch(base + path, {
    method,
    signal: AbortSignal.timeout(30000),
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: data === undefined ? undefined : JSON.stringify(data),
  });
  const body = await response.text();
  if (!response.ok)
    throw new Error(`${method} ${path}: ${response.status} ${body}`);
  return JSON.parse(body);
}
async function allSites() {
  const items = [];
  for (let page = 1; ; page++) {
    const result = await api(`/api/sites?page=${page}&per_page=100`);
    items.push(...result.items);
    if (items.length >= result.pagination.total) return items;
  }
}
try {
  token = (
    await api("/api/auth/login", "POST", {
      identifier: process.env.DEMO_ADMIN_LOGIN || "admin",
      password: process.env.DEMO_ADMIN_PASSWORD || "admin-dev-only-2026",
    })
  ).access_token;
  let site = (await allSites()).find((s) => s.domain === domain);
  if (!site) {
    site = (
      await api("/api/sites", "POST", {
        profile_code: "dev",
        domain,
        locale: "ru-RU",
        is_public: true,
        settings: {
          string_value: "Контур — демо",
          integer_value: 3,
          float_value: 1,
          checkbox_value: false,
          radio_value: "first",
          select_value: "alpha",
          multi_select_value: [],
          textarea_value: "Демонстрационная архитектурная студия",
          email_value: "hello@kontur.example",
          phone_value: "+79000000000",
        },
      })
    ).site;
  }
  if (site.profile_code !== "dev" || !site.is_public)
    throw new Error(
      "Demo domain belongs to a non-public or incompatible site; choose another DEMO_DOMAIN.",
    );
  // Fill missing defaults only; an editor's existing values, including empty ones, win.
  const current = (await api(`/api/sites/${site.id}`)).site;
  const defaults = { phone_value: "+79000000000", email_value: "hello@kontur.example" };
  if (Object.keys(defaults).some(key => !Object.hasOwn(current.settings, key))) {
    site = (await api(`/api/sites/${site.id}`, "PATCH", {
      domain: current.domain, profile_code: current.profile_code,
      locale: current.locale, is_public: current.is_public,
      settings: { ...defaults, ...current.settings },
    })).site;
  }
  const endpoint = `/api/sites/${site.id}/resources`;
  async function update(item, changes) {
    const names = ["parent_id", "type", "template_code", "title", "menu_title", "slug", "annotation", "content_type", "content", "target_resource_id", "external_url", "is_public", "is_searchable", "in_menu", "in_sitemap", "sort", "published_at", "unpublished_at", "fields", "type_settings", "image_media_id"];
    const body = Object.fromEntries(names.filter(name => name in item).map(name => [name, item[name]]));
    return (await api(`${endpoint}/${item.id}`, "PATCH", { ...body, ...changes, expected_version: item.version })).resource;
  }
  const existing = new Map();
  async function children(parent) {
    const result = await api(endpoint + (parent ? `?parent_id=${parent}` : ""));
    for (const item of result.items) {
      const full = (await api(`${endpoint}/${item.id}`)).resource;
      existing.set(full.path, full);
      if (item.has_children) await children(item.id);
    }
  }
  await children();
  for (const [sort, page] of pages.entries()) {
    let item = existing.get(page.path);
    // Preserve editor changes on repeated runs. Resume only empty initial resources.
    if (item) {
      item = (await api(`${endpoint}/${item.id}`)).resource;
      if (page.path !== "/" && page.path.split("/").length === 2 && item.parent_id !== null) {
        item = await update(item, { parent_id: null, sort: 2147483647 });
        existing.set(page.path, item);
        console.log(`Moved to root ${page.path}`);
      }
      if (
        item.version > 1 ||
        item.content ||
        (page.path === "/" && item.title !== "Первая страница")
      ) {
        if (page.path === "/" && item.menu_title === page.title) {
          item = await update(item, { menu_title: "Главная" });
          existing.set(page.path, item);
        }
        console.log(`Keep ${page.path}`);
        continue;
      }
    } else {
      const parentPath = page.path.slice(0, page.path.lastIndexOf("/")) || "/";
      const parent = existing.get(parentPath);
      if (parentPath !== "/" && !parent) throw new Error(`Missing parent ${parentPath}`);
      const created = await api(endpoint, "POST", {
        parent_id: parentPath === "/" ? null : parent.id,
        type: "page",
        title: page.title,
        menu_title: page.path === "/" ? "Главная" : page.title,
        slug: page.path.split("/").at(-1),
        content_type: "html",
        content: "",
        fields: {},
        type_settings: {},
      });
      item = (await api(`${endpoint}/${created.id}`)).resource;
    }
    const fields = {
      page_title: page.title,
      page_text: page.annotation,
      show_title: true,
      layout: "wide",
    };
    item = (
      await api(`${endpoint}/${item.id}`, "PATCH", {
        expected_version: item.version,
        parent_id: item.parent_id,
        type: "page",
        template_code: "page",
        title: page.title,
        menu_title: page.path === "/" ? "Главная" : page.title,
        slug: item.slug,
        annotation: page.annotation,
        content_type: "html",
        content: page.content,
        is_public: true,
        is_searchable: true,
        in_menu: true,
        in_sitemap: true,
        sort,
        fields,
        type_settings: {},
      })
    ).resource;
    existing.set(page.path, item);
    console.log(`Created ${page.path} (resource ${item.id})`);
  }
  async function ensureResource(path, data) {
    let item = existing.get(path);
    if (!item) {
      const { is_public = true, in_menu = true, is_searchable = true, in_sitemap = true, sort = 0, ...creation } = data;
      const created = await api(endpoint, "POST", { fields: {}, type_settings: {}, ...creation });
      item = (await api(`${endpoint}/${created.id}`)).resource;
      item = await update(item, { is_public, in_menu, is_searchable, in_sitemap, sort });
      existing.set(path, item);
      console.log(`Created ${path} (resource ${item.id})`);
    }
    return item;
  }
  const menuConfig = { projects: existing.get("/projects").id };
  for (const [index, targets] of [["/projects", "/services"], ["/about", "/journal", "/contacts"]].entries()) {
    const path = `/footer-${index + 1}`;
    const parent = await ensureResource(path, { parent_id: null, type: "page", slug: path.slice(1), title: `Футер ${index + 1}`, in_menu: false, is_searchable: false, in_sitemap: false, content_type: "html", sort: 100 + index });
    menuConfig[`footer${index + 1}`] = parent.id;
    for (const [sort, targetPath] of targets.entries()) {
      const target = existing.get(targetPath);
      await ensureResource(path + targetPath, { parent_id: parent.id, type: "resource_link", slug: targetPath.slice(1), title: target.menu_title || target.title, target_resource_id: target.id, is_searchable: false, in_sitemap: false, sort });
    }
  }
  const configPath = process.env.DEMO_MENU_CONFIG || "/data/menu-config.json";
  await writeFile(configPath + ".tmp", JSON.stringify(menuConfig, null, 2));
  await rename(configPath + ".tmp", configPath);
  console.log(`Menu IDs: ${JSON.stringify(menuConfig)}`);
  console.log(
    `Demo site ${domain}, site ID ${site.id}; ${pages.length} content pages. Open http://localhost:${process.env.DEMO_PORT || 4173}`,
  );
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
