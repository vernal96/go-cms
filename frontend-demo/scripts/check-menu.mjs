// Live acceptance check. Creates and removes an isolated temporary CMS site.
import assert from "node:assert/strict";
import { setTimeout as delay } from "node:timers/promises";
import http from "node:http";

const base = process.env.DEMO_API_TARGET || "http://localhost:8080";
let domain = `menu-check-${Date.now()}.test`;
let token, siteID, destinationSiteID;
async function request(path, method = "GET", body, publicRequest = false) {
  if (publicRequest) return new Promise((resolve, reject) => {
    const req = http.get(new URL(path, base), { headers: { Host: domain, Accept: "application/json" } }, response => {
      let text = "";
      response.setEncoding("utf8"); response.on("data", chunk => { text += chunk; });
      response.on("end", () => { let data; try { data = JSON.parse(text); } catch { data = text; } resolve({ status: response.statusCode, data }); });
      response.on("error", reject);
    });
    req.setTimeout(30000, () => req.destroy(new Error("CMS timeout")));
    req.on("error", reject);
  });
  const response = await fetch(base + path, {
    method, signal: AbortSignal.timeout(30000),
    headers: { "Content-Type": "application/json", ...(publicRequest ? { Host: domain } : token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  return { status: response.status, data: text ? JSON.parse(text) : null };
}
async function api(path, method, body) {
  const result = await request(path, method, body);
  assert.ok(result.status >= 200 && result.status < 300, `${method || "GET"} ${path}: ${JSON.stringify(result)}`);
  return result.data;
}
const endpoint = () => `/api/sites/${siteID}/resources`;
async function update(id, changes) {
  const item = (await api(`${endpoint()}/${id}`)).resource;
  const names = ["parent_id", "type", "template_code", "title", "menu_title", "slug", "annotation", "content_type", "content", "target_resource_id", "external_url", "is_public", "is_searchable", "in_menu", "in_sitemap", "sort", "published_at", "unpublished_at", "fields", "type_settings"];
  const body = Object.fromEntries(names.map(key => [key, item[key]]));
  return (await api(`${endpoint()}/${id}`, "PATCH", { ...body, ...changes, expected_version: item.version })).resource;
}
async function create(slug, parent_id = null, extras = {}, changes = {}) {
  const item = await api(endpoint(), "POST", { parent_id, slug, title: slug, type: "page", content_type: "html", fields: {}, type_settings: {}, ...extras });
  return update(item.id, { is_public: true, in_menu: true, ...changes });
}
async function menu(query = "") {
  const result = await request("/menu" + query, "GET", undefined, true);
  assert.equal(result.status, 200, JSON.stringify(result));
  return result.data.items;
}
try {
  token = (await api("/api/auth/login", "POST", { identifier: process.env.DEMO_ADMIN_LOGIN || "admin", password: process.env.DEMO_ADMIN_PASSWORD || "admin-dev-only-2026" })).access_token;
  siteID = (await api("/api/sites", "POST", { domain, profile_code: "dev", locale: "ru-RU", is_public: true, settings: { string_value: "menu test", integer_value: 3, float_value: 1, checkbox_value: false, radio_value: "first", select_value: "alpha", multi_select_value: [], textarea_value: "", email_value: "menu@example.test" } })).site.id;
  const products = await create("products");
  const product = await create("one", products.id);
  const nested = await create("nested", product.id);
  const footer = await create("footer", null, {}, { in_menu: false });
  const link = await create("shortcut", footer.id, { type: "resource_link", content_type: null, target_resource_id: product.id }, { menu_title: "Shortcut" });
  const external = await create("external", null, { type: "link", content_type: null, external_url: "https://example.org/" });
  assert.equal((await menu()).find(i => i.id === footer.id), undefined);
  assert.equal((await menu("?depth=1")).find(i => i.id === products.id).children.length, 0);
  assert.equal((await menu(`?parent_id=${products.id}&depth=1`))[0].children.length, 0);
  assert.equal((await menu(`?parent_id=${products.id}`))[0].children[0].id, nested.id);
  const footerQuery = `?parent_id=${footer.id}`;
  assert.equal((await menu(footerQuery))[0].url, "/products/one");
  assert.equal((await menu()).find(i => i.id === external.id).url, "https://example.org/");
  // Warm both variants, mutate a target outside the footer branch, then read again.
  await menu(footerQuery); await menu("?depth=2");
  await update(product.id, { slug: "renamed", in_menu: false });
  assert.equal((await menu(footerQuery))[0].url, "/products/renamed");
  assert.equal((await menu(`?parent_id=${products.id}`)).length, 0);
  assert.equal((await menu(`?parent_id=${product.id}`))[0].id, nested.id);
  await update(product.id, { is_public: false });
  assert.equal((await menu(footerQuery)).length, 0);
  assert.equal((await request(`/menu?parent_id=${product.id}`, "GET", undefined, true)).status, 404);
  await update(product.id, { is_public: true, published_at: new Date(Date.now() + 1500).toISOString() });
  assert.equal((await menu(footerQuery)).length, 0);
  await delay(1700);
  assert.equal((await menu(footerQuery))[0].id, link.id);
  await update(product.id, { published_at: null, unpublished_at: new Date(Date.now() + 1500).toISOString() });
  assert.equal((await menu(footerQuery)).length, 1);
  await delay(1700);
  assert.equal((await menu(footerQuery)).length, 0);
  await update(product.id, { unpublished_at: null, in_menu: true });
  await update(link.id, { parent_id: products.id, sort: 0 });
  assert.equal((await menu(footerQuery)).length, 0);
  assert.equal((await menu(`?parent_id=${products.id}`))[0].id, link.id);
  const before = (await api(`${endpoint()}/${link.id}`)).resource;
  const changed = await update(link.id, { menu_title: "Revised label" });
  assert.equal((await menu(`?parent_id=${products.id}`))[0].title, "Revised label");
  await api(`${endpoint()}/${link.id}/revisions/${before.version}/restore`, "POST", { expected_version: changed.version });
  assert.equal((await menu(`?parent_id=${products.id}`))[0].title, before.menu_title);
  await api(`${endpoint()}/${link.id}`, "DELETE");
  assert.ok(!(await menu(`?parent_id=${products.id}`)).some(item => item.id === link.id));
  await api(`${endpoint()}/${link.id}/restore`, "POST", { with_descendants: true });
  assert.ok((await menu(`?parent_id=${products.id}`)).some(item => item.id === link.id));
  for (const query of ["?depth=0", "?depth=1&depth=2", "?parent_id=-1", "?unknown=1"]) assert.equal((await request("/menu" + query, "GET", undefined, true)).status, 400);
  const otherSite = (await api("/api/sites?per_page=100")).items.find(item => item.id !== siteID);
  if (otherSite) {
    const foreign = (await api(`/api/sites/${otherSite.id}/resources`)).items[0];
    if (foreign) assert.equal((await request(`/menu?parent_id=${foreign.id}`, "GET", undefined, true)).status, 404);
  }
  const sourceDomain = domain;
  const destinationDomain = `target-${domain}`;
  const source = (await api(`/api/sites/${siteID}`)).site;
  destinationSiteID = (await api("/api/sites", "POST", { domain: destinationDomain, profile_code: "dev", locale: "ru-RU", is_public: true, settings: source.settings })).site.id;
  assert.ok((await menu()).some(item => item.id === external.id));
  domain = destinationDomain;
  assert.ok(!(await menu()).some(item => item.id === external.id));
  domain = sourceDomain;
  const currentExternal = (await api(`${endpoint()}/${external.id}`)).resource;
  await api(`${endpoint()}/${external.id}/transfer`, "POST", { target_site_id: destinationSiteID, expected_version: currentExternal.version });
  assert.ok(!(await menu()).some(item => item.id === external.id));
  domain = destinationDomain;
  assert.ok((await menu()).some(item => item.id === external.id));
  console.log("PASS: PostgreSQL/Redis live menu: root, depth, hidden parent, links, mutation invalidation, timed publication, ordering, revision restore, deletion/restore, cross-site transfer invalidation, site isolation, validation");
} finally {
  if (siteID) await api(`/api/sites/${siteID}`, "DELETE");
  if (destinationSiteID) await api(`/api/sites/${destinationSiteID}`, "DELETE");
}
