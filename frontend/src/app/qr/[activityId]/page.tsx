"use client";

// WP-3.3 (RC-10): the poster QR deep-link landing page. Scanning a poster
// QR opens /qr/<activityId>, which:
//   1. records the pilot scan (fire-and-forget; offline scans are simply
//      not counted until the device reconnects — the page never invents one),
//   2. warms the offline cache by fetching the activity modules + journey
//      through fetchWithCache, so the poster's content works with no signal
//      (the offline demo kit),
//   3. marks the scan as "started" when the learner clicks through — the
//      pilot's honest first-session drop-off signal.
import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { QrCode, Download, ArrowRight, CheckCircle2, WifiOff, Loader2 } from "lucide-react";
import { fetchWithCache } from "@/lib/api";
import SkeletonLoader from "@/components/SkeletonLoader";

type ScanState = "recording" | "recorded" | "offline" | "failed";

export default function QRLanding() {
  const router = useRouter();
  const params = useParams();
  const activityId = (params?.activityId as string) || "";
  const [state, setState] = useState<ScanState>("recording");
  const [title, setTitle] = useState("");
  const [modulesReady, setModulesReady] = useState(false);
  const scanIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (!activityId) {
      setState("failed");
      return;
    }

    const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:6101/api/v1";
    const token = localStorage.getItem("log_token");

    // 1. Record the scan. No auth required by design (posters work before
    //    login), so an absent token is fine. Failures are honest: offline or
    //    an error means the scan just isn't counted.
    (async () => {
      try {
        const res = await fetch(`${apiBase}/pilot/scans`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ poster_id: activityId, source: "qr" }),
        });
        if (res.ok) {
          const body = await res.json().catch(() => ({}));
          scanIdRef.current = body?.scan_id ?? null;
          setState("recorded");
        } else {
          setState("failed");
        }
      } catch {
        setState("offline");
      }
    })();

    // 2. Warm the offline cache through the normal fetchWithCache seam, so
    //    the scanned activity is usable with no signal afterwards.
    Promise.all([
      fetchWithCache(`/activities/${activityId}/modules`).catch(() => null),
      fetchWithCache("/learning-journey", {
        headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      }).catch(() => null),
    ]).then(([mods]) => {
      const m = mods as { activity?: { title?: string }; modules?: unknown[] } | null;
      if (m?.activity?.title) setTitle(m.activity.title);
      setModulesReady(Array.isArray(m?.modules) && (m.modules?.length ?? 0) > 0);
    });
  }, [activityId]);

  const markStarted = () => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:6101/api/v1";
    if (scanIdRef.current != null) {
      fetch(`${apiBase}/pilot/scans/${scanIdRef.current}/start`, { method: "POST" }).catch(() => {});
    }
    router.push(`/learning/${activityId}`);
  };

  if (!activityId) {
    return (
      <div className="max-w-xl mx-auto w-full text-center py-24 space-y-6">
        <h1 className="text-2xl font-bold text-white">This poster is not linked to a lesson</h1>
        <Link href="/learning" className="btn-primary px-8 py-3 font-bold inline-block">
          Browse the Learning Journey
        </Link>
      </div>
    );
  }

  if (state === "recording") {
    return (
      <div className="max-w-xl mx-auto w-full space-y-6">
        <SkeletonLoader type="card" count={2} />
      </div>
    );
  }

  const statusCopy =
    state === "recorded"
      ? "Scan recorded — this poster is being counted in the pilot."
      : state === "offline"
        ? "You are offline, so this scan is not counted yet. Your lesson is still being saved for offline use."
        : "This scan could not be recorded. The lesson below still works normally.";

  return (
    <div className="max-w-xl mx-auto w-full">
      <div className="card-glow bg-black/40 backdrop-blur-3xl border border-white/10 rounded-3xl p-8 text-center">
        <span className="inline-flex p-4 rounded-full bg-brand-dark text-brand-neon mb-6">
          <QrCode className="w-8 h-8" />
        </span>
        <h1 className="text-3xl font-bold text-white tracking-tight mb-2">
          {title || "Poster Lesson"}
        </h1>
        <p className="text-white/60 mb-2">
          Scanned from a classroom poster. Your lesson is ready — even without internet.
        </p>
        <p className="text-sm text-brand-muted mb-8">
          {statusCopy}
        </p>

        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <button onClick={markStarted} className="btn-primary px-8 py-3 font-bold flex items-center justify-center gap-2">
            {modulesReady ? (
              <>
                Start the lesson <ArrowRight className="w-4 h-4" />
              </>
            ) : (
              <>
                <Loader2 className="w-4 h-4 animate-spin" /> Saving for offline…
              </>
            )}
          </button>
          <Link href="/learning" className="px-6 py-3 rounded-full text-sm font-bold bg-white/5 border border-white/10 text-white/70 hover:bg-white/10 hover:text-white transition-all flex items-center justify-center gap-2">
            Browse all lessons
          </Link>
        </div>

        <div className="mt-8 pt-6 border-t border-white/10 text-xs text-brand-muted flex items-center justify-center gap-2">
          {state === "recorded" ? (
            <><CheckCircle2 className="w-4 h-4 text-brand-teal" /> Pilot scan recorded</>
          ) : state === "offline" ? (
            <><WifiOff className="w-4 h-4" /> Offline — scan will not be counted</>
          ) : (
            <><Download className="w-4 h-4" /> Lesson cached for offline use</>
          )}
        </div>
      </div>
    </div>
  );
}