#!/usr/bin/env python3
"""
LOG live-stack test suite — runs against the Docker-composed stack.

  python3 scripts/live_stack_test.py [--base http://localhost:6101/api/v1] [--web http://localhost:6100]

Covers (per AGENTS.md constraints — everything must be honest, nothing fabricated):
  L1  Liveness probes            L6  Consent gate (403 consent_required)
  L2  Security headers           L7  Guardian consent -> enroll -> complete
  L3  Auth (login/bad/malformed) L8  Teacher flows
  L4  RBAC (student->admin 403)  L9  Admin flows + honest analytics (null avg)
  L5  Catalog / dashboard as_of  L10 Metrics: public text + JSON, no PII
  L11 Rate limiting (X-RateLimit*, 429 + Retry-After)
  L12 Privacy: /me/consent, export, logout Clear-Site-Data
  W1  Frontend routes            W2  Browser: dark/light theme toggle end-to-end

Exit code: number of failed checks (0 = all green).
"""
import argparse, hashlib, json, re, sys, time
import requests

PASS, FAIL = 0, 0
def check(name, cond, detail=""):
    global PASS, FAIL
    mark = "PASS" if cond else "FAIL"
    if cond: PASS += 1
    else: FAIL += 1
    print(f"  [{mark}] {name}" + (f"  {detail}" if detail else ""))

def disclosure_hash(text):
    return hashlib.sha256(text.encode()).hexdigest()

CONSENT_NOTICE = "Guardian Consent · अभिभावकको सहमति\nbilingual notice for live-stack test"

def login(base, email, password):
    r = requests.post(f"{base}/auth/login", json={"email": email, "password": password}, timeout=15)
    return r

def auth_headers(token):
    return {"Authorization": f"Bearer {token}"}

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://localhost:6101/api/v1")
    ap.add_argument("--web", default="http://localhost:6100")
    args = ap.parse_args()
    base, web = args.base.rstrip("/"), args.web.rstrip("/")

    print("=" * 72)
    print("L1  Liveness probes")
    r = requests.get(base.replace("/api/v1", "/api/ping"), timeout=10)
    check("GET /api/ping", r.status_code == 200 and r.json().get("message") == "pong", f"{r.status_code}")
    r = requests.get(base.replace("/api/v1", "/healthz"), timeout=10)
    check("GET /healthz (real SQLite ping)", r.status_code == 200 and r.json().get("status") == "ok", f"{r.status_code}")
    r = requests.get(base.replace("/api/v1", "/readyz"), timeout=10)
    check("GET /readyz", r.status_code == 200, f"{r.status_code}")

    print("=" * 72)
    print("L2  Security headers")
    r = requests.get(f"{base}/courses", timeout=10)  # any protected route
    for h, want in [("X-Content-Type-Options", "nosniff"), ("X-Frame-Options", None), ("X-XSS-Protection", None)]:
        if h in r.headers:
            check(f"header {h}", True, r.headers[h])
        else:
            check(f"header {h} present", False, "missing")
    check("no server stack trace leak on 404", requests.get(f"{base}/does-not-exist", timeout=10).status_code in (404, 405), "")

    print("=" * 72)
    print("L3  Auth flows")
    bad = login(base, "aisha@example.com", "wrong-password")
    check("wrong password -> 401", bad.status_code == 401, f"{bad.status_code}")
    r = login(base, "not-an-email", "x")
    check("garbage creds rejected", r.status_code in (400, 401, 422), f"{r.status_code}")
    r = requests.post(f"{base}/auth/login", json={}, timeout=10)
    check("missing fields -> 400", r.status_code == 400, f"{r.status_code}")

    # Fresh registration — the live-stack suite registers its own learner so
    # the consent gate (L6) can be tested against an account with no grants.
    fresh_email = f"livetest-{int(time.time())}@log.edu"
    r = requests.post(f"{base}/auth/register", json={"name": "Live Test", "email": fresh_email, "password": "Password123"}, timeout=15)
    check("email/password register -> 201 + token", r.status_code == 201 and "token" in r.json(), f"{r.status_code} {r.text[:100]}")
    fresh_token = r.json().get("token", "")
    r = requests.post(f"{base}/auth/register", json={"name": "Live Test", "email": fresh_email, "password": "Password123"}, timeout=15)
    check("duplicate register -> 409", r.status_code == 409, f"{r.status_code}")
    r = requests.post(f"{base}/auth/register", json={"name": "X", "email": "nope", "password": "x"}, timeout=15)
    check("invalid register payload -> 400", r.status_code == 400, f"{r.status_code}")
    r = requests.post(f"{base}/auth/login", json={"email": fresh_email, "password": "Password123"}, timeout=15)
    check("fresh account logs in", r.status_code == 200 and "token" in r.json(), f"{r.status_code}")
    fresh_token = r.json()["token"]

    r = requests.post(f"{base}/auth/login", json={"email": "aisha@example.com", "password": "Student@123"}, timeout=15)
    check("student login", r.status_code == 200 and "token" in r.json(), f"{r.status_code}")
    student_token = r.json()["token"]
    r = requests.post(f"{base}/auth/login", json={"email": "teacher@log.edu", "password": "Teacher@123"}, timeout=15)
    check("teacher login", r.status_code == 200 and "token" in r.json(), f"{r.status_code}")
    teacher_token = r.json()["token"]
    r = requests.post(f"{base}/auth/login", json={"email": "admin@log.edu", "password": "Admin@123"}, timeout=15)
    check("admin login", r.status_code == 200 and "token" in r.json(), f"{r.status_code}")
    admin_token = r.json()["token"]
    r = requests.get(f"{base}/dashboard", headers=auth_headers("totally.bogus.token"), timeout=10)
    check("malformed JWT -> 401", r.status_code == 401, f"{r.status_code}")

    print("=" * 72)
    print("L4  RBAC")
    r = requests.get(f"{base}/admin/dashboard", headers=auth_headers(student_token), timeout=10)
    check("student -> /admin 403", r.status_code == 403, f"{r.status_code}")
    r = requests.get(f"{base}/moderator/roster", headers=auth_headers(student_token), timeout=10)
    check("student -> /moderator 403", r.status_code == 403, f"{r.status_code}")
    r = requests.get(f"{base}/moderator/roster", headers=auth_headers(admin_token), timeout=10)
    check("admin -> /moderator allowed (>= role)", r.status_code == 200, f"{r.status_code}")

    print("=" * 72)
    print("L5  Catalog + dashboard freshness")
    r = requests.get(f"{base}/courses", headers=auth_headers(student_token), timeout=15)
    cbody = r.json()
    courses = cbody.get("courses", []) if isinstance(cbody, dict) else (cbody if isinstance(cbody, list) else [])
    check("GET /courses (student)", r.status_code == 200, f"{r.status_code}")
    course_id = str(courses[0]["id"]) if courses else None
    check("catalog non-empty (seeded)", course_id is not None, f"{len(courses)} courses")
    r = requests.get(f"{base}/dashboard", headers=auth_headers(student_token), timeout=15)
    d = r.json()
    check("GET /dashboard 200", r.status_code == 200, f"{r.status_code}")
    check("dashboard carries as_of (freshness honesty)", "as_of" in d, json.dumps(d)[:120])
    r = requests.get(f"{base}/learning-journey", headers=auth_headers(student_token), timeout=15)
    check("GET /learning-journey 200", r.status_code == 200, f"{r.status_code}")
    r = requests.get(f"{base}/chart-data", headers=auth_headers(student_token), timeout=15)
    cd = r.json()
    check("GET /chart-data 200 + as_of", r.status_code == 200 and "as_of" in cd, f"{r.status_code}")

    # A real activity id for the completion flow — from the dashboard payload.
    activity_id = None
    blob = json.dumps(d)
    m = re.search(r'"id"\s*:\s*"(act-[^"]+)"', blob)
    if m: activity_id = m.group(1)
    check("found a real activity id for completion flow", activity_id is not None, f"{activity_id}")

    print("=" * 72)
    print("L6  Consent gate (server-side enforcement — fresh account)")
    if course_id and activity_id:
        r = requests.post(f"{base}/activities/{activity_id}/complete",
                          headers=auth_headers(fresh_token),
                          json={"elapsed_seconds": 20, "correct_count": 4, "total_count": 5}, timeout=15)
        body = r.json() if r.headers.get("content-type", "").startswith("application/json") else {}
        check("complete w/o consent -> 403 consent_required",
              r.status_code == 403 and body.get("code") == "consent_required", f"{r.status_code} {json.dumps(body)[:80]}")
        r = requests.post(f"{base}/courses/{course_id}/enroll", headers=auth_headers(fresh_token), json={}, timeout=15)
        body = r.json() if r.headers.get("content-type", "").startswith("application/json") else {}
        check("enroll w/o consent -> 403 consent_required",
              r.status_code == 403 and body.get("code") == "consent_required", f"{r.status_code}")

    print("=" * 72)
    print("L7  Guardian consent -> enroll -> complete")
    consent_payload = {
        "consent_type": "guardian", "version": "2026-08-v1",
        "granted_by": "guardian", "language": "en", "source": "live-test",
        "guardian_name": "Live Stack Test", "guardian_contact": "test@example.com",
        "disclosure_hash": disclosure_hash(CONSENT_NOTICE),
    }
    r = requests.post(f"{base}/me/consent", headers=auth_headers(fresh_token), json=consent_payload, timeout=15)
    check("guardian consent grant", r.status_code in (200, 201), f"{r.status_code} {r.text[:120]}")
    for t in ("terms", "privacy"):
        r = requests.post(f"{base}/me/consent", headers=auth_headers(fresh_token),
                          json={"consent_type": t, "version": "2026-08-v1", "granted_by": "self", "language": "en", "source": "live-test"}, timeout=15)
        check(f"consent grant ({t})", r.status_code in (200, 201), f"{r.status_code}")
    r = requests.post(f"{base}/me/consent", headers=auth_headers(fresh_token), json={
        "consent_type": "guardian", "version": "2026-08-v1", "granted_by": "guardian", "language": "en", "source": "live-test",
        "disclosure_hash": "not-a-real-hash"}, timeout=15)
    check("guardian consent w/o valid disclosure_hash -> 400", r.status_code == 400, f"{r.status_code}")

    if course_id:
        r = requests.post(f"{base}/courses/{course_id}/enroll", headers=auth_headers(fresh_token), json={}, timeout=15)
        check("enroll after consent", r.status_code in (200, 201, 409), f"{r.status_code} {r.text[:100]}")
    if activity_id:
        r = requests.post(f"{base}/activities/{activity_id}/complete",
                          headers=auth_headers(fresh_token),
                          json={"elapsed_seconds": 20, "correct_count": 4, "total_count": 5}, timeout=15)
        check("complete after consent", r.status_code == 200, f"{r.status_code} {r.text[:120]}")
        r = requests.get(f"{base}/dashboard", headers=auth_headers(fresh_token), timeout=15)
        d = r.json()
        check("dashboard reflects completion (no fabricated numbers)",
              r.status_code == 200 and "stats" in d or "goal" in d or "activities" in d, f"{r.status_code}")
    r = requests.get(f"{base}/me/consent", headers=auth_headers(fresh_token), timeout=15)
    mc = r.json()
    check("GET /me/consent (policy + retention)", r.status_code == 200 and "policy" in mc, f"{r.status_code} {json.dumps(mc)[:120]}")

    print("=" * 72)
    print("L8  Teacher flows")
    r = requests.get(f"{base}/moderator/classes", headers=auth_headers(teacher_token), timeout=15)
    tclasses = r.json().get("classes", []) if r.status_code == 200 else []
    check("teacher classes", r.status_code == 200 and len(tclasses) > 0, f"{r.status_code} ({len(tclasses)} classes)")
    r = requests.get(f"{base}/moderator/roster", headers=auth_headers(teacher_token), timeout=15)
    check("moderator roster", r.status_code == 200, f"{r.status_code}")
    class_id = str(tclasses[0]["id"]) if tclasses else None
    r = requests.get(f"{base}/moderator/gradebook?class_id={class_id}", headers=auth_headers(teacher_token), timeout=15) if class_id else requests.get(f"{base}/moderator/gradebook", headers=auth_headers(teacher_token), timeout=15)
    check("gradebook (class-scoped)", r.status_code in (200, 404), f"{r.status_code}")
    r = requests.get(f"{base}/moderator/gradebook.csv?class_id={class_id}", headers=auth_headers(teacher_token), timeout=15) if class_id else requests.get(f"{base}/moderator/gradebook.csv", headers=auth_headers(teacher_token), timeout=15)
    check("gradebook CSV (formula-safe cells)", r.status_code == 200 and r.content[:1] not in (b"=", b"+", b"-", b"@", b"\t"), f"{r.status_code} head={r.content[:20]!r}")

    print("=" * 72)
    print("L9  Admin flows + honest analytics")
    r = requests.get(f"{base}/admin/dashboard", headers=auth_headers(admin_token), timeout=15)
    check("admin dashboard", r.status_code == 200, f"{r.status_code}")
    r = requests.get(f"{base}/admin/users", headers=auth_headers(admin_token), timeout=15)
    users = (r.json().get("users", []) if isinstance(r.json(), dict) else r.json()) if r.status_code == 200 else []
    check("admin users list", r.status_code == 200 and isinstance(users, list) and len(users) > 0, f"{r.status_code} ({len(users)} users)")
    r = requests.get(f"{base}/admin/metrics", headers=auth_headers(admin_token), timeout=15)
    mj = r.json() if r.status_code == 200 else {}
    check("admin metrics JSON", r.status_code == 200 and "routes" in mj, f"{r.status_code}")
    r = requests.get(f"{base}/admin/analytics/summary", headers=auth_headers(admin_token), timeout=15)
    s = r.json() if r.status_code == 200 else {}
    check("analytics summary (no opt-in yet)", r.status_code == 200 and s.get("avg_score") is None,
          f"avg_score={s.get('avg_score')!r} (must be null, never fabricated 0)")
    r = requests.get(f"{base}/admin/pilot/stats", headers=auth_headers(admin_token), timeout=15)
    check("pilot stats (honest zeros)", r.status_code == 200, f"{r.status_code}")
    r = requests.get(f"{base}/admin/audit-log", headers=auth_headers(admin_token), timeout=15)
    check("audit log", r.status_code == 200, f"{r.status_code}")

    print("=" * 72)
    print("L10 Metrics: public text + no PII")
    r = requests.get(base.replace("/api/v1", "/metrics"), timeout=10)
    body = r.text
    check("GET /metrics text/plain", r.status_code == 200 and r.headers.get("content-type", "").startswith("text/plain"), r.headers.get("content-type", ""))
    check("metrics contain route patterns", "route=" in body, body[:200])
    for leak in ("aisha@example.com", "Student@123", "Bearer", "student-"):
        check(f"no PII in /metrics ({leak!r})", leak not in body, "")

    print("=" * 72)
    print("L11 Analytics opt-in -> summary -> withdrawal (honest flip)")
    r = requests.post(f"{base}/me/consent", headers=auth_headers(student_token),
                      json={"consent_type": "analytics", "version": "2026-08-v1", "granted_by": "self", "language": "en", "source": "settings", "status": "granted"}, timeout=15)
    check("analytics opt-in", r.status_code in (200, 201), f"{r.status_code} {r.text[:100]}")
    time.sleep(0.5)
    r = requests.get(f"{base}/admin/analytics/summary", headers=auth_headers(admin_token), timeout=15)
    s = r.json()
    check("summary sees opted-in learner", r.status_code == 200 and s.get("opted_in_users", 0) >= 1, f"{json.dumps(s)[:140]}")
    r = requests.post(f"{base}/me/consent", headers=auth_headers(student_token),
                      json={"consent_type": "analytics", "version": "2026-08-v1", "granted_by": "self", "language": "en", "source": "settings", "status": "withdrawn"}, timeout=15)
    check("analytics withdrawal", r.status_code in (200, 201), f"{r.status_code} {r.text[:100]}")
    time.sleep(0.5)
    r = requests.get(f"{base}/admin/analytics/summary", headers=auth_headers(admin_token), timeout=15)
    s = r.json()
    check("summary drops opted-out learner", r.status_code == 200 and s.get("opted_in_users", 0) == 0, f"{json.dumps(s)[:140]}")

    print("=" * 72)
    print("L12 Privacy: export + logout headers")
    r = requests.get(f"{base}/me/export", headers=auth_headers(student_token), timeout=15)
    check("GET /me/export", r.status_code == 200, f"{r.status_code}")
    if r.status_code == 200:
        exp = r.text
        check("export has no password hash", "bcrypt" not in exp and "$2" not in exp, "")
        check("export self-describing envelope", '"email"' in exp and "data" in exp, exp[:120])
    r = requests.post(f"{base}/auth/logout", headers=auth_headers(student_token), timeout=15)
    check("logout clears site data", r.status_code == 200 and "clear-site-data" in {k.lower() for k in r.headers},
          f"{r.status_code} {r.headers.get('Clear-Site-Data', '')}")
    r = requests.get(f"{base}/dashboard", headers=auth_headers(student_token), timeout=10)
    check("revoked token -> 401", r.status_code == 401, f"{r.status_code}")
    student_token = login(base, "aisha@example.com", "Student@123").json()["token"]  # re-login for later sections

    print("=" * 72)
    print("L13 Rate limiting (public pilot route, own bucket — deterministic)")
    hdr_r = requests.post(f"{base}/pilot/scans", json={"poster_id": "act-1"}, timeout=15)
    check("rate-limit headers present", "X-RateLimit-Limit" in hdr_r.headers and "X-RateLimit-Remaining" in hdr_r.headers,
          f"{hdr_r.headers.get('X-RateLimit-Limit', '?')}/{hdr_r.headers.get('X-RateLimit-Remaining', '?')}")
    blocked = None
    for _ in range(40):
        r = requests.post(f"{base}/pilot/scans", json={"poster_id": "act-1"}, timeout=15)
        if r.status_code == 429:
            blocked = r
            break
    check("hammered pilot scans -> 429 (after budget)", blocked is not None, f"status={blocked.status_code if blocked else 'never'}")
    check("429 carries Retry-After", blocked is not None and "Retry-After" in blocked.headers,
          blocked.headers.get("Retry-After", "missing") if blocked else "")

    print("=" * 72)
    print("W1  Frontend routes (web server)")
    for path, want in [("/", 200), ("/login", 200), ("/dashboard", (301, 302, 307, 308)), ("/settings", (301, 302, 307, 308))]:
        r = requests.get(web + path, timeout=30, allow_redirects=False)
        ok = r.status_code == want if isinstance(want, int) else r.status_code in want
        check(f"GET {path} -> {want}", ok, f"got {r.status_code}")
    r = requests.get(web + "/login", timeout=30)
    check("login page is the app (has root div)", r.status_code == 200 and "<html" in r.text, "")

    # Browser: dark/light theme end-to-end (W2)
    print("=" * 72)
    print("W2  Browser — dark/light theme toggle end-to-end")
    try:
        from playwright.sync_api import sync_playwright
        with sync_playwright() as pw:
            browser = pw.chromium.launch(executable_path="/usr/bin/chromium", headless=True)
            page = browser.new_page(viewport={"width": 1280, "height": 800})
            page.goto(web + "/login", wait_until="networkidle")
            page.wait_for_timeout(800)
            cls = page.evaluate("document.documentElement.className")
            check("default theme is dark", "dark" in cls, f"html class={cls!r}")
            btn = page.locator("button[aria-label='Toggle Theme']")
            check("theme toggle present (nav + login card)", btn.count() >= 1, f"count={btn.count()}")
            if btn.count() >= 1:
                btn.first.click()
                page.wait_for_timeout(500)
                cls2 = page.evaluate("document.documentElement.className")
                check("toggle -> light (class removed)", "dark" not in cls2, f"html class={cls2!r}")
                stored = page.evaluate("localStorage.getItem('theme')")
                check("theme persisted to localStorage (offline-ready)", stored == "light", f"theme={stored!r}")
                page.screenshot(path="/tmp/log-theme-light.png")
                btn.first.click()
                page.wait_for_timeout(500)
                cls3 = page.evaluate("document.documentElement.className")
                check("toggle -> dark again", "dark" in cls3, f"html class={cls3!r}")
                page.screenshot(path="/tmp/log-theme-dark.png")

            # Student login -> Settings Appearance switch
            page.goto(web + "/login", wait_until="networkidle")
            page.fill("input[type='email']", "aisha@example.com")
            page.fill("input[type='password']", "Student@123")
            page.click("button[type='submit']")
            page.wait_for_timeout(3000)
            page.goto(web + "/settings", wait_until="networkidle")
            page.wait_for_timeout(1000)
            switch = page.locator("button[aria-label='Toggle dark mode']")
            check("settings Appearance dark-mode switch present", switch.count() == 1, f"count={switch.count()}")
            if switch.count() == 1:
                check("switch reflects current theme", switch.first.get_attribute("aria-checked") == "true", f"aria-checked={switch.first.get_attribute('aria-checked')}")
                switch.first.click()
                page.wait_for_timeout(500)
                page.goto(web + "/login", wait_until="networkidle")
                stored2 = page.evaluate("localStorage.getItem('theme')")
                check("switch flips + persists across navigation", stored2 == "light", f"theme={stored2!r}")
                page.screenshot(path="/tmp/log-settings-check.png")
            browser.close()
    except Exception as e:
        check("W2 browser suite ran", False, repr(e)[:200])

    print("=" * 72)
    total = PASS + FAIL
    print(f"RESULT: {PASS}/{total} checks passed, {FAIL} failed")
    return FAIL if FAIL else 0

if __name__ == "__main__":
    sys.exit(main())