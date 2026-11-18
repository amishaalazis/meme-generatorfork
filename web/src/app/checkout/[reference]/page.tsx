"use client";

// Halaman simulasi pembayaran (mock payment gateway).
// Di produksi halaman ini digantikan halaman payment gateway sungguhan
// yang memanggil webhook /confirm dengan verifikasi signature.

import { useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { queryKeys, useConfirmPayment } from "@/lib/queries";
import { describeError } from "@/lib/errors";

function formatIdr(n: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(n);
}

export default function CheckoutPage() {
  const { reference } = useParams<{ reference: string }>();
  const router = useRouter();
  const confirm = useConfirmPayment();
  const qc = useQueryClient();
  const [payError, setPayError] = useState("");
  const refStr = typeof reference === "string" ? reference : Array.isArray(reference) ? reference[0] : "";

  const onPay = async () => {
    if (!refStr || confirm.isPending || !!confirm.data) return;
    setPayError("");
    try {
      await confirm.mutateAsync(refStr);
      await qc.invalidateQueries({ queryKey: queryKeys.subscription });
      await qc.refetchQueries({ queryKey: queryKeys.subscription });
      // Beri jeda agar pesan sukses dan struk pembayaran sempat dibaca user sebelum berpindah
      setTimeout(() => router.push("/pricing"), 1500);
    } catch (err) {
      setPayError(describeError(err));
    }
  };

  return (
    <main className="mx-auto flex min-h-[70vh] max-w-md items-center p-8">
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Simulasi Pembayaran</CardTitle>
          <CardDescription>
            Payment gateway mock — tidak ada uang sungguhan.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">Referensi Transaksi</span>
            <span className="font-mono">{refStr}</span>
          </div>

          {confirm.data ? (
            <div className="space-y-3 rounded-lg border border-green-200 bg-green-50/60 p-4 dark:border-green-900 dark:bg-green-950/20">
              <p className="text-center text-sm font-semibold text-green-600 dark:text-green-400">
                ✅ Pembayaran Berhasil! Langganan Anda Aktif.
              </p>
              <div className="space-y-1.5 border-t border-green-200/60 pt-2 text-xs text-muted-foreground dark:border-green-900/60">
                <div className="flex justify-between">
                  <span>Status Pembayaran:</span>
                  <span className="font-bold text-green-600 dark:text-green-400">LUNAS</span>
                </div>
                {confirm.data.payment?.amountIdr ? (
                  <div className="flex justify-between">
                    <span>Total Tagihan:</span>
                    <span className="font-semibold text-foreground">
                      {formatIdr(confirm.data.payment.amountIdr)}
                    </span>
                  </div>
                ) : null}
                <div className="flex justify-between">
                  <span>No. Struk:</span>
                  <span className="font-mono text-foreground">{refStr}</span>
                </div>
              </div>
              <p className="text-center text-[11px] text-muted-foreground">
                Mengalihkan kembali ke halaman paket…
              </p>
            </div>
          ) : null}

          {payError && (
            <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive">
              <p className="font-semibold">Pembayaran Gagal</p>
              <p className="text-xs mt-0.5">{payError}</p>
            </div>
          )}
        </CardContent>
        <CardFooter className="flex gap-2">
          <Button
            className="flex-1"
            onClick={onPay}
            disabled={confirm.isPending || !!confirm.data}
          >
            {confirm.isPending
              ? "Memproses…"
              : confirm.data
                ? "Lunas"
                : "Bayar Sekarang (Simulasi)"}
          </Button>
          <Button asChild variant="ghost" disabled={confirm.isPending}>
            <Link href="/pricing">Batal</Link>
          </Button>
        </CardFooter>
      </Card>
    </main>
  );
}
