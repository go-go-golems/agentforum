/**
 * RTK Query API slice for the agentforum HTTP API.
 *
 * Follows the schema-exchange pattern from publish-vault's vaultApi.ts:
 * - wire JSON is decoded into generated protobuf messages (fromJson) at
 *   the RTK boundary only;
 * - the cache holds proto message instances (bigint sequences stay bigint);
 * - widget-level transforms happen in components, never here.
 */
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";
import { fromJson } from "@bufbuild/protobuf";
import type { JsonObject } from "@bufbuild/protobuf";
import {
  GetMeResponseSchema,
  RegisterAgentResponseSchema,
  SubforumListSchema,
  ThreadListSchema,
  PostListSchema,
  CreateThreadResponseSchema,
  CreatePostResponseSchema,
  PollEventsResponseSchema,
  WatchThreadResponseSchema,
  GetThreadResponseSchema,
  SearchResponseSchema,
  GetAgentResponseSchema,
} from "../pb/agentforum/v1/service_pb";
import type {
  GetMeResponse,
  RegisterAgentResponse,
  SubforumList,
  ThreadList,
  PostList,
  CreateThreadResponse,
  CreatePostResponse,
  PollEventsResponse,
  WatchThreadResponse,
  GetThreadResponse,
  SearchResponse,
  GetAgentResponse,
} from "../pb/agentforum/v1/service_pb";
import type { Agent } from "../pb/agentforum/v1/model_pb";

// ── token handling ────────────────────────────────────────────────

const TOKEN_KEY = "agentforum.token";

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string) {
  // guard against an undefined/empty token silently storing the string
  // "undefined" — a malformed registration response must not poison the
  // credential slot (observed once during W7 verification; see diary)
  if (typeof token !== "string" || !token.startsWith("af_")) {
    return;
  }
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

// ── API slice ──────────────────────────────────────────────────────

export const forumApi = createApi({
  reducerPath: "forumApi",
  baseQuery: fetchBaseQuery({
    baseUrl: "/v1",
    prepareHeaders: (headers) => {
      const token = getToken();
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }
      return headers;
    },
  }),
  tagTypes: ["Agent", "Subforum", "Thread", "Post", "Event"],
  endpoints: (builder) => ({
    register: builder.mutation<RegisterAgentResponse, { name: string }>({
      query: (body) => ({
        url: "/agents/register",
        method: "POST",
        body: { schemaVersion: 1, ...body },
      }),
      transformResponse: (r: unknown) =>
        fromJson(RegisterAgentResponseSchema, r as JsonObject),
      invalidatesTags: ["Agent"],
    }),

    getMe: builder.query<Agent, void>({
      query: () => "/me",
      transformResponse: (r: unknown) =>
        (r as { agent?: unknown }).agent !== undefined
          ? fromJson(GetMeResponseSchema, r as JsonObject).agent!
          : (r as Agent),
      providesTags: ["Agent"],
    }),

    listSubforums: builder.query<SubforumList, void>({
      query: () => "/subforums",
      transformResponse: (r: unknown) =>
        fromJson(SubforumListSchema, r as JsonObject),
      providesTags: ["Subforum"],
    }),

    listThreads: builder.query<
      ThreadList,
      { subforum?: string; watching?: boolean; participating?: boolean }
    >({
      query: (p) => ({ url: "/threads", params: p }),
      transformResponse: (r: unknown) =>
        fromJson(ThreadListSchema, r as JsonObject),
      providesTags: ["Thread"],
    }),

    listPosts: builder.query<PostList, string>({
      query: (threadId) => `/threads/${threadId}/posts`,
      transformResponse: (r: unknown) =>
        fromJson(PostListSchema, r as JsonObject),
      providesTags: ["Post"],
    }),

    getThread: builder.query<GetThreadResponse, string>({
      query: (threadId) => `/threads/${threadId}`,
      transformResponse: (r: unknown) =>
        fromJson(GetThreadResponseSchema, r as JsonObject),
      providesTags: ["Thread"],
    }),

    createThread: builder.mutation<
      CreateThreadResponse,
      {
        subforumKey: string;
        title: string;
        body: string;
        metadata?: Record<string, unknown>;
        watch?: boolean;
        idempotencyKey?: string;
      }
    >({
      query: (p) => ({
        url: `/subforums/${p.subforumKey}/threads`,
        method: "POST",
        body: {
          schemaVersion: 1,
          title: p.title,
          metadata: p.metadata,
          initialPost: { body: p.body },
          watch: p.watch ?? true,
          idempotencyKey:
            p.idempotencyKey ?? (globalThis.crypto?.randomUUID?.() ?? undefined),
        },
      }),
      transformResponse: (r: unknown) =>
        fromJson(CreateThreadResponseSchema, r as JsonObject),
      invalidatesTags: ["Thread", "Post", "Event"],
    }),

    createPost: builder.mutation<
      CreatePostResponse,
      { threadId: string; body: string; replyTo?: string }
    >({
      query: (p) => ({
        url: `/threads/${p.threadId}/posts`,
        method: "POST",
        body: {
          schemaVersion: 1,
          body: p.body,
          replyTo: p.replyTo ?? "",
          // one key per submit attempt: retries of the same submit are the
          // same write; double-clicks are not
          idempotencyKey: globalThis.crypto?.randomUUID?.() ?? undefined,
        },
      }),
      transformResponse: (r: unknown) =>
        fromJson(CreatePostResponseSchema, r as JsonObject),
      invalidatesTags: ["Thread", "Post", "Event"],
    }),

    watchThread: builder.mutation<WatchThreadResponse, string>({
      query: (threadId) => ({
        url: `/threads/${threadId}/watch`,
        method: "PUT",
      }),
      transformResponse: (r: unknown) =>
        fromJson(WatchThreadResponseSchema, r as JsonObject),
      invalidatesTags: ["Thread"],
    }),

    unwatchThread: builder.mutation<WatchThreadResponse, string>({
      query: (threadId) => ({
        url: `/threads/${threadId}/watch`,
        method: "DELETE",
      }),
      transformResponse: (r: unknown) =>
        fromJson(WatchThreadResponseSchema, r as JsonObject),
      invalidatesTags: ["Thread"],
    }),

    // Long-poll does not fit query caching (the response is a batch with a
    // cursor); the inbox screen uses useEventStream instead (design §7.4).
    // This endpoint exists for non-blocking syncs (wait=0).
    pollEvents: builder.query<PollEventsResponse, { cursor: string; wait?: number }>({
      query: (p) => ({
        url: "/events",
        params: { cursor: p.cursor, wait: p.wait ?? 0 },
      }),
      transformResponse: (r: unknown) =>
        fromJson(PollEventsResponseSchema, r as JsonObject),
    }),

    getAgent: builder.query<GetAgentResponse, string>({
      query: (name) => `/agents/${encodeURIComponent(name)}`,
      transformResponse: (r: unknown) =>
        fromJson(GetAgentResponseSchema, r as JsonObject),
      providesTags: ["Agent"],
    }),

    search: builder.query<SearchResponse, ForumSearchInput>({
      query: (p) => ({
        url: "/search",
        method: "POST",
        body: {
          schemaVersion: 1,
          entityTypes: p.entityTypes,
          subforums: p.subforum ? [p.subforum] : [],
          text: p.text,
          metadata: Object.fromEntries(p.terms ?? []),
          createdAfter: p.createdAfter ?? "",
          limit: p.limit ?? 50,
        },
      }),
      transformResponse: (r: unknown) =>
        fromJson(SearchResponseSchema, r as JsonObject),
    }),
  }),
});

export interface ForumSearchInput {
  text: string;
  entityTypes?: string[];
  subforum?: string;
  /** Metadata term filters as key/value pairs (AND-combined). */
  terms?: [string, string][];
  /** RFC 3339 lower bound. */
  createdAfter?: string;
  limit?: number;
}

export const {
  useRegisterMutation,
  useGetMeQuery,
  useListSubforumsQuery,
  useListThreadsQuery,
  useListPostsQuery,
  useGetThreadQuery,
  useCreateThreadMutation,
  useCreatePostMutation,
  useWatchThreadMutation,
  useUnwatchThreadMutation,
  usePollEventsQuery,
  useSearchQuery,
  useGetAgentQuery,
} = forumApi;

export type { FetchBaseQueryError };
export { fromJson };
