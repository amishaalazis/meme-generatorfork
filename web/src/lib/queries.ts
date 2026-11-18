import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type {
  CheckoutResponse,
  ConfirmResponse,
  Meme,
  Plan,
  SubscriptionStatus,
  UploadResponse,
} from "@/lib/types";

export const queryKeys = {
  plans: ["plans"] as const,
  subscription: ["subscription"] as const,
  memes: (limit: number, offset: number) => ["memes", limit, offset] as const,
};

export function usePlans() {
  return useQuery({
    queryKey: queryKeys.plans,
    queryFn: async () =>
      unwrap<Plan[]>(await api.get("/api/v1/subscription/plans")) ?? [],
  });
}

export function useSubscriptionStatus() {
  return useQuery({
    queryKey: queryKeys.subscription,
    queryFn: async () =>
      unwrap<SubscriptionStatus>(await api.get("/api/v1/subscription/status")),
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    refetchInterval: 60 * 1000,
  });
}

export function useCheckout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ planCode, idempotencyKey }: { planCode: string; idempotencyKey: string }) =>
      unwrap<CheckoutResponse>(
        await api.post("/api/v1/subscription/checkout", { planCode }, {
          headers: {
            // Key dibuat sekali di sisi pemanggil (per intent/klik),
            // bukan per eksekusi mutationFn, agar retry/double-click
            // menghasilkan checkout yang sama dan tidak duplikat.
            "Idempotency-Key": idempotencyKey,
          },
        }),
      ),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.subscription });
    },
  });
}

export function useConfirmPayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (reference: string) =>
      unwrap<ConfirmResponse>(
        await api.post(
          `/api/v1/subscription/payments/${reference}/confirm`,
        ),
      ),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.subscription });
      await qc.refetchQueries({ queryKey: queryKeys.subscription });
    },
  });
}

export function useMemes(limit = 20, offset = 0) {
  return useQuery({
    queryKey: queryKeys.memes(limit, offset),
    queryFn: async () =>
      unwrap<{ items: Meme[]; total: number }>(
        await api.get("/api/v1/memes", { params: { limit, offset } }),
      ),
  });
}

export function usePrepareUpload() {
  return useMutation({
    mutationFn: async (input: { title: string; filename: string; size: number }) =>
      unwrap<UploadResponse>(
        await api.post("/api/v1/memes/upload", input),
      ),
  });
}

export function useCompleteMeme() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { id: string; resultUrl: string }) =>
      unwrap<Meme>(
        await api.post(`/api/v1/memes/${input.id}/complete`, {
          resultUrl: input.resultUrl,
        }),
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["memes"] }),
  });
}

export function useDeleteMeme() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/v1/memes/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["memes"] }),
  });
}
