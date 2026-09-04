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
  WatchSubforumResponseSchema,
  GetSubforumResponseSchema,
  UpdateAgentResponseSchema,
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
  WatchSubforumResponse,
  GetSubforumResponse,
  UpdateAgentResponse,
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

const TOKEN_RE = /^af_[A-Za-z0-9_-]+$/;

export function setToken(token: string) {
  // guard against an undefined/empty/prefix-only token silently poisoning the
  // credential slot — a malformed registration response must not store
  // "undefined" or a bare "af_" (observed once during W7 verification; probe
  // and pinned by web/src/store/forumApi.test.ts, AGENTFORUM-005 A3). Real
  // tokens are af_<43 base64url chars> (internal/id NewToken).
  if (typeof token !== "string" || !TOKEN_RE.test(token)) {
    return;
  }
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

// ── API slice ──────────────────────────────────────────────────────

// A5: posts load in pages of 50; a full page means "maybe more".
export const POSTS_PAGE_SIZE = 50;

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

    listPosts: builder.query<PostList, { threadId: string; after?: string }>({
      // A5 (AGENTFORUM-005): paginated with the after_post_id cursor.
      // All pages of one thread share a cache entry (serializeQueryArgs
      // keys on the thread only); incoming pages are merged and deduped,
      // so tag invalidation (new reply) refetches only posts after the
      // cursor and appends them.
      query: (p) => ({
        url: `/threads/${p.threadId}/posts`,
        params: p.after ? { after: p.after, limit: POSTS_PAGE_SIZE } : { limit: POSTS_PAGE_SIZE },
      }),
      transformResponse: (r: unknown) =>
        fromJson(PostListSchema, r as JsonObject),
      providesTags: ["Post"],
      serializeQueryArgs: ({ endpointName, queryArgs }) =>
        `${endpointName}-${queryArgs.threadId}`,
      merge: (current, incoming) => {
        if (!incoming.posts?.length) return;
        const seen = new Set(current.posts.map((p) => p.id));
        for (const p of incoming.posts) {
          if (!seen.has(p.id)) current.posts.push(p);
        }
      },
      forceRefetch: ({ currentArg, previousArg }) =>
        currentArg?.after !== previousArg?.after,
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

    // A1 (AGENTFORUM-005): subforum watch state was server-complete since
    // W2 but had no UI; these three expose it.
    getSubforum: builder.query<GetSubforumResponse, string>({
      query: (key) => `/subforums/${encodeURIComponent(key)}`,
      transformResponse: (r: unknown) =>
        fromJson(GetSubforumResponseSchema, r as JsonObject),
      providesTags: ["Subforum"],
    }),

    watchSubforum: builder.mutation<WatchSubforumResponse, string>({
      query: (key) => ({
        url: `/subforums/${encodeURIComponent(key)}/watch`,
        method: "PUT",
      }),
      transformResponse: (r: unknown) =>
        fromJson(WatchSubforumResponseSchema, r as JsonObject),
      invalidatesTags: ["Subforum"],
    }),

    unwatchSubforum: builder.mutation<WatchSubforumResponse, string>({
      query: (key) => ({
        url: `/subforums/${encodeURIComponent(key)}/watch`,
        method: "DELETE",
      }),
      transformResponse: (r: unknown) =>
        fromJson(WatchSubforumResponseSchema, r as JsonObject),
      invalidatesTags: ["Subforum"],
    }),

    // A2 (AGENTFORUM-005): profile metadata editing over PATCH /v1/me.
    // Semantics (service UpdateMe): a metadata field present (even an
    // empty object) replaces the stored metadata; absent leaves it.
    updateMe: builder.mutation<UpdateAgentResponse, Record<string, unknown>>({
      query: (metadata) => ({
        url: "/me",
        method: "PATCH",
        body: { schemaVersion: 1, metadata },
      }),
      transformResponse: (r: unknown) =>
        fromJson(UpdateAgentResponseSchema, r as JsonObject),
      invalidatesTags: ["Agent"],
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
  useGetSubforumQuery,
  useWatchSubforumMutation,
  useUnwatchSubforumMutation,
  useUpdateMeMutation,
  usePollEventsQuery,
  useSearchQuery,
  useGetAgentQuery,
} = forumApi;

export type { FetchBaseQueryError };
export { fromJson };
