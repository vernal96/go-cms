import test from "node:test";
import assert from "node:assert/strict";
import http from "node:http";
import { once } from "node:events";
import { mkdtemp, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

test("public rendering, CMS authorization and HTTP boundaries", async (t) => {
  let upstreamStatus = 200;
  let requests = [];
  let menuTitle = "Меню из CMS";
  let menuFailure = false;
  let siteFailure = false;
  let siteName = "Студия из CMS";
  let contentType = "html";
  let content = "<h1>Live CMS content</h1>";
  let contentWidgets = () => [{ code: "core_content", data: { content } }];
  const api = http.createServer((req, res) => {
    requests.push({
      path: req.url,
      host: req.headers.host,
      authorization: req.headers.authorization,
    });
    res.writeHead((menuFailure && req.url.startsWith("/menu")) || (siteFailure && req.url === "/site") ? 500 : upstreamStatus, { "Content-Type": "application/json" });
    res.end(
      JSON.stringify(
        req.url === "/site" ? { settings: { string_value: siteName, email_value: "hello@example.test", phone_value: "+79000000000" } } : req.url.startsWith("/menu")
          ? { items: [{ id: 12, title: menuTitle, url: "/projects", children: [] }] }
          : req.url.startsWith("/search")
          ? {
              items: [
                {
                  title: "<script>alert(1)</script>",
                  url: "javascript:alert(1)",
                  annotation: "A & B",
                },
              ],
              pagination: { page: 1, per_page: 10, total: 1 },
            }
          : {
              resource: {
                title: "Demo & CMS",
                annotation: '"quoted"',
                content_type: contentType,
              },
              widgets: { body: contentWidgets(), sidebar: [] },
            },
      ),
    );
  });
  api.listen(0, "127.0.0.1");
  await once(api, "listening");
  process.env.DEMO_API_TARGET = `http://127.0.0.1:${api.address().port}`;
  process.env.DEMO_DOMAIN = "demo.localhost";
  const directory = await mkdtemp(join(tmpdir(), "demo-menu-"));
  process.env.DEMO_MENU_CONFIG = join(directory, "menu.json");
  await writeFile(process.env.DEMO_MENU_CONFIG, JSON.stringify({ projects: 10, footer1: 11, footer2: 12 }));
  const { server } = await import("./server.mjs");
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const url = `http://127.0.0.1:${server.address().port}`;
  t.after(async () => {
    server.closeAllConnections();
    server.close();
    api.closeAllConnections();
    api.close();
    await rm(directory, { recursive: true });
  });
  await t.test(
    "renders current public content and never forwards browser credentials",
    async () => {
      const response = await fetch(url + "/about", {
        headers: { Authorization: "Bearer should-not-be-forwarded" },
      });
      const body = await response.text();
      assert.equal(response.status, 200);
      assert.match(body, /<h1>Live CMS content<\/h1>/);
      assert.match(body, /<title>Demo &amp; CMS/);
      assert.match(
        response.headers.get("content-security-policy"),
        /script-src 'none'/,
      );
      assert.deepEqual(requests.find(r => r.path === "/about"), {
        path: "/about",
        host: "demo.localhost",
        authorization: undefined,
      });
    },
  );
  await t.test("renders plain text from the content widget with HTML escaping", async () => {
    contentType = "text";
    content = "<script>alert(1)</script> & text";
    try {
      const response = await fetch(url + "/about");
      const body = await response.text();
      assert.equal(response.status, 200);
      assert.match(body, /<h1>Demo &amp; CMS<\/h1>/);
      assert.match(body, /<p>&lt;script&gt;alert\(1\)&lt;\/script&gt; &amp; text<\/p>/);
      assert.doesNotMatch(body, /<script>/);
    } finally {
      contentType = "html";
      content = "<h1>Live CMS content</h1>";
    }
  });
  await t.test("reports missing or failed content widgets", async () => {
    const original = contentWidgets;
    try {
      for (const widgets of [[], [{ code: "core_content", error: { code: "render_failed" } }]]) {
        contentWidgets = () => widgets;
        assert.equal((await fetch(url + "/about")).status, 502);
      }
    } finally {
      contentWidgets = original;
    }
  });
  await t.test("loads public site settings, escapes values and fails on CMS errors", async () => {
    let body = await (await fetch(url + "/")).text();
    assert.match(body, /Студия из CMS/);
    assert.match(body, /hello@example.test/);
    assert.match(body, /tel:\+79000000000/);
    siteName = '<script>private?</script>';
    body = await (await fetch(url + "/")).text();
    assert.match(body, /&lt;script&gt;private\?&lt;\/script&gt;/);
    siteFailure = true;
    assert.equal((await fetch(url + "/")).status, 502);
    siteFailure = false;
  });
  await t.test("loads four CMS menus, reflects changes and reports menu failure", async () => {
    requests = [];
    const response = await fetch(url + "/projects");
    assert.equal(response.status, 200);
    assert.match(await response.text(), /Меню из CMS/);
    assert.deepEqual(requests.filter(r => r.path.startsWith("/menu")).map(r => r.path).sort(), ["/menu?depth=1", "/menu?parent_id=10", "/menu?parent_id=11", "/menu?parent_id=12"]);
    assert.ok(requests.every(r => r.authorization === undefined));
    menuTitle = "Новая подпись <CMS>";
    assert.match(await (await fetch(url + "/projects")).text(), /Новая подпись &lt;CMS&gt;/);
    menuFailure = true;
    assert.equal((await fetch(url + "/projects")).status, 502);
    menuFailure = false;
  });
  await t.test(
    "preserves CMS denial and not-found responses; maps failures to 502",
    async () => {
      for (const [upstream, expected] of [
        [403, 403],
        [404, 404],
        [500, 502],
      ]) {
        upstreamStatus = upstream;
        const response = await fetch(url + "/private");
        assert.equal(response.status, expected);
        assert.doesNotMatch(await response.text(), /Live CMS content/);
      }
      upstreamStatus = 200;
    },
  );
  await t.test(
    "search escapes stored text and rejects unsafe result links",
    async () => {
      const response = await fetch(url + "/search?q=дом");
      const body = await response.text();
      assert.equal(response.status, 200);
      assert.match(body, /&lt;script&gt;/);
      assert.doesNotMatch(body, /href="javascript:/);
      assert.equal((await fetch(url + "/search?q=ab")).status, 400);
    },
  );
  await t.test(
    "does not expose management endpoints or support mutations",
    async () => {
      const before = requests.length;
      for (const path of ["/api/sites", "/_cms/runtime", "/assets/unknown.svg"])
        assert.equal((await fetch(url + path)).status, 404);
      assert.equal((await fetch(url + "/", { method: "POST" })).status, 405);
      assert.equal(requests.length, before);
    },
  );
  await t.test(
    "serves local assets and HEAD without a response body",
    async () => {
      assert.equal(
        (await fetch(url + "/assets/house.svg")).headers.get("content-type"),
        "image/svg+xml",
      );
      const response = await fetch(url + "/", { method: "HEAD" });
      assert.equal(response.status, 200);
      assert.equal(await response.text(), "");
    },
  );
});
