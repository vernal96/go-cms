#!/usr/bin/env python3
"""Check a running starter, optionally exercising an isolated deployment."""
import argparse
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import uuid


def base_url():
    if os.getenv("BASE_URL"):
        return os.environ["BASE_URL"].rstrip("/")
    port = os.getenv("SERVER_PORT")
    if port is None and Path(".env").is_file():
        for line in Path(".env").read_text().splitlines():
            if line.startswith("SERVER_PORT="):
                port = line.split("=", 1)[1].strip().strip("\"'")
    return "http://localhost:" + (port or "8080")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--exercise", action="store_true", help="create data in an isolated test deployment")
    mode.add_argument("--verify", action="store_true", help="verify data after container recreation")
    parser.add_argument("--state", type=Path)
    args = parser.parse_args()
    if (args.exercise or args.verify) and args.state is None:
        parser.error("--state is required with --exercise or --verify")
    base = base_url()

    def request(path, *, token=None, payload=None, body=None, content_type="application/json", expected=200):
        headers = {"Content-Type": content_type}
        if token:
            headers["Authorization"] = "Bearer " + token
        if payload is not None:
            body = json.dumps(payload).encode()
        req = urllib.request.Request(base + path, data=body, headers=headers)
        try:
            response = urllib.request.urlopen(req, timeout=20)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            data = response.read()
            status = response.status
            kind = response.headers.get("Content-Type", "")
        if status != expected:
            raise RuntimeError(f"{path}: HTTP {status}, expected {expected}: {data.decode(errors='replace')[:500]}")
        print(f"PASS {path}: HTTP {status}")
        return json.loads(data) if "application/json" in kind else data

    request("/healthz")
    token = request("/api/auth/login", payload={
        "identifier": os.getenv("CMS_LOGIN", "admin"),
        "password": os.getenv("CMS_PASSWORD", "admin-dev-only-2026"),
    })["access_token"]
    for path in ["/api/admin/session", "/api/admin/navigation", "/api/sites", "/api/files/disks"]:
        request(path, token=token)
    request("/api/admin/session", expected=401)
    request("/api/files/disks", expected=401)
    if not (args.exercise or args.verify):
        print("PASS CMS smoke: health, login, admin session, navigation, sites and disks")
        return
    if args.exercise:
        sites = request("/api/sites", token=token)["items"]
        site_id = next(site["id"] for site in sites if site["domain"] == "localhost")
        name = "smoke-" + uuid.uuid4().hex[:12]
        resource = request(f"/api/sites/{site_id}/resources", token=token, expected=201, payload={
            "type": "page", "title": name, "slug": name, "fields": {}, "type_settings": {},
        })
        state = {"name": name, "site_id": site_id, "resource_id": resource["id"], "files": []}
        for disk in ["public", "private"]:
            folder = request("/api/files/folders", token=token, expected=201, payload={"disk": disk, "name": name})
            boundary = "cms-smoke-" + uuid.uuid4().hex
            data = ""
            for key, value in [("disk", disk), ("folder_id", folder["id"])]:
                data += f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n'
            content = f"{name} {disk}\n"
            data += f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="smoke.txt"\r\nContent-Type: text/plain\r\n\r\n{content}\r\n--{boundary}--\r\n'
            uploaded = request("/api/files/uploads", token=token, expected=201, body=data.encode(), content_type="multipart/form-data; boundary=" + boundary)
            state["files"].append({"id": uploaded["id"], "content": content})
        args.state.write_text(json.dumps(state))
    else:
        state = json.loads(args.state.read_text())
    resource = request(f'/api/sites/{state["site_id"]}/resources/{state["resource_id"]}', token=token)["resource"]
    if resource["title"] != state["name"]:
        raise RuntimeError("resource did not survive restart")
    request("/" + state["name"], token=token)
    # The starter deliberately requires explicit guest grants for public content.
    request("/" + state["name"], expected=403)
    for item in state["files"]:
        path = f'/api/files/{item["id"]}/download'
        if request(path, token=token) != item["content"].encode():
            raise RuntimeError("stored file bytes differ")
        request(path, expected=401)
    print("PASS CMS deployment: resource and public/private files persisted; guest access remains restricted")


if __name__ == "__main__":
    main()
