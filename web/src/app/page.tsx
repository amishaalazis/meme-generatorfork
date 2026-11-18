"use client";

// Halaman depan: tampil untuk semua user (login maupun tidak).
// Tidak ada paksa-redirect — link ke /studio dan /pricing sudah cukup.
// Guard per-halaman (studio, dll.) masing-masing menangani kasus belum login.
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useMe } from "@/lib/auth";

export default function Home() {
  const me = useMe();

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-8 p-8">
      <h1
        className="text-6xl font-bold tracking-tight text-center"
        style={{ fontFamily: "var(--font-impact)" }}
      >
        MEME PORTAL
      </h1>
      <p className="text-muted-foreground text-center max-w-md">
        Meme generator berlangganan. Upload gambar dasar, tulis teks dengan
        font Impact, dan bagikan hasilnya.
      </p>
      <div className="flex gap-4">
        <Button asChild>
          <Link href="/studio">Buat Meme</Link>
        </Button>
        <Button asChild variant="outline">
          <Link href="/pricing">Lihat Paket</Link>
        </Button>
        {!me.isLoading && !me.data && (
          <Button asChild variant="ghost">
            <Link href="/login">Masuk</Link>
          </Button>
        )}
      </div>
    </main>
  );
}
