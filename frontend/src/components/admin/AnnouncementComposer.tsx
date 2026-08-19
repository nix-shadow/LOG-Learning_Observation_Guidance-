"use client";
import { useState } from 'react';
import { fetchWithCache } from '@/lib/api';
import { Loader2, Megaphone } from 'lucide-react';
import toast from 'react-hot-toast';

interface AnnouncementComposerProps {
  token: string;
  endpoint: string;
}

// AnnouncementComposer is the shared publish form for both the admin and
// moderator dashboards. The endpoint differs per role; everything else —
// validation, publishing, honest feedback — lives here, tested once.
export default function AnnouncementComposer({ token, endpoint }: AnnouncementComposerProps) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [isPublishing, setIsPublishing] = useState(false);

  const handlePublish = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title || !body) {
      toast.error('Title and message are required');
      return;
    }
    setIsPublishing(true);
    try {
      const res = await fetchWithCache(endpoint, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ title, body }),
      });
      // F8: queued writes are pending, not published.
      if (res && res.queued) {
        toast.success('Announcement saved offline. Will sync when back online.', { icon: '💾' });
      } else {
        toast.success('Announcement published');
      }
      setTitle('');
      setBody('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to publish announcement');
    } finally {
      setIsPublishing(false);
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-3">
        <Megaphone className="w-6 h-6 text-brand-amber" />
        <h2 className="text-xl font-bold text-white tracking-tight">Publish Announcement</h2>
      </div>
      <form onSubmit={handlePublish} className="space-y-4">
        <input aria-label="Announcement title" value={title} onChange={e => setTitle(e.target.value)}
          className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30"
          placeholder="Announcement title" />
        <textarea aria-label="Announcement message" value={body} onChange={e => setBody(e.target.value)} rows={4}
          className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-xl text-white focus:ring-2 focus:ring-brand-neon/50 outline-none placeholder-white/30 resize-none"
          placeholder="Message for all students & teachers…" />
        <button type="submit" disabled={isPublishing} className="btn-primary w-full flex items-center justify-center gap-2 disabled:opacity-40">
          {isPublishing && <Loader2 className="w-4 h-4 animate-spin" />}
          {isPublishing ? 'Publishing...' : 'Publish'}
        </button>
      </form>
    </div>
  );
}