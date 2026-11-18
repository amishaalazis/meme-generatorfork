import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { ApiEnvelope } from "@/lib/types";

export interface AuthUser {
  id: string;
  email: string;
  name: string;
}

export interface AuthResponse {
  user: AuthUser;
}

interface AuthInput {
  email: string;
  password: string;
  name?: string;
}

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: async () => unwrap<AuthUser>(await api.get("/api/v1/auth/me")),
    retry: false,
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: async (input: AuthInput) => {
      const res = await api.post<ApiEnvelope<AuthResponse>>(
        "/api/v1/auth/register",
        { name: input.name, email: input.email, password: input.password },
      );
      return res.data;
    },
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: Omit<AuthInput, "name">) => {
      const res = await api.post<ApiEnvelope<AuthResponse>>(
        "/api/v1/auth/login",
        input,
      );
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => api.post("/api/v1/auth/logout"),
    onSuccess: () => {
      // Hapus seluruh cache React Query agar data user lama tidak tersisa,
      // lalu arahkan ke /login — bukan "/" — untuk menghindari redirect loop
      // jika "/" masih mencoba mendeteksi sesi yang baru saja dihapus.
      qc.clear();
      window.location.href = "/login";
    },
    onError: () => {
      // Logout gagal di server (mis. Redis down) tapi cookie sudah dikirim
      // untuk dihapus oleh backend. Tetap clear cache dan arahkan ke /login
      // agar user tidak terjebak di halaman yang memerlukan autentikasi.
      qc.clear();
      window.location.href = "/login";
    },
  });
}
