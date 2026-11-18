// Bentuk envelope respons dari backend Go (internal/response).
export interface ApiEnvelope<T> {
  success: boolean;
  code: number;
  message: string;
  data?: T;
  errors?: unknown;
}

export interface Plan {
  id: string;
  code: string;
  name: string;
  priceIdr: number;
  durationDays: number;
  maxMemes: number; // 0 = unlimited
}

export interface SubscriptionStatus {
  subscribed: boolean;
  planCode?: string;
  planName?: string;
  endsAt?: string;
  maxMemes: number;
  memesUsed: number;
}

export interface Meme {
  id: string;
  title: string;
  baseUrl: string;
  resultUrl?: string;
  createdAt: string;
}

export interface UploadResponse {
  id: string;
  uploadUrl: string;
  baseUrl: string;
  contentType: string;
}

export interface CheckoutResponse {
  subscription: { id: string; status: string };
  payment: {
    id: string;
    subscriptionId: string;
    amountIdr: number;
    status: string;
    reference: string;
  };
  mockGatewayUrl: string;
}

export interface ConfirmResponse {
  subscription: {
    id: string;
    userId: string;
    planId: string;
    status: string;
    startsAt?: string;
    endsAt?: string;
  };
  payment: {
    id: string;
    subscriptionId: string;
    amountIdr: number;
    status: string;
    reference: string;
  };
}
