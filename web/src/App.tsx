/**
 * App — routing + auth gate.
 *
 * The bearer token is the identity (design D8): no token means the register
 * screen; a token means the forum shell. A 401 from getMe (stale token)
 * clears it and returns to register.
 */
import React from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useGetMeQuery } from "./store/forumApi";
import { clearToken, getToken } from "./store/forumApi";
import { ForumShell } from "./components/pages/ForumShell/ForumShell";
import { RegisterScreen } from "./components/pages/RegisterScreen/RegisterScreen";
import { SubforumListScreen } from "./components/pages/SubforumListScreen/SubforumListScreen";
import { ThreadListScreen } from "./components/pages/ThreadListScreen/ThreadListScreen";
import { ThreadDetailScreen } from "./components/pages/ThreadDetailScreen/ThreadDetailScreen";
import { InboxScreen } from "./components/pages/InboxScreen/InboxScreen";
import { SearchScreen } from "./components/pages/SearchScreen/SearchScreen";
import { ProfileScreen } from "./components/pages/ProfileScreen/ProfileScreen";
import { Icon } from "./components/atoms/Icon/Icon";

export function App() {
  // Skip getMe entirely when no token exists. This is the W7-anomaly fix
  // (AGENTFORUM-005, found live during A1 verification): the register
  // mutation invalidates the "Agent" tag, so RTK Query refetches getMe in
  // the window before RegisterScreen's setToken runs. That tokenless
  // refetch 401s, and an unconditional clearToken() here wiped the
  // just-stored token — an intermittent lost login. With skip, there is
  // no tokenless request at all: no 401, nothing to clear. A 401 that
  // still arrives can only have carried a (stale) token and is cleared.
  const hasToken = getToken() !== "";
  const { data: me, isLoading, isError, error } = useGetMeQuery(undefined, {
    skip: !hasToken,
  });
  const unauthorized =
    isError && (error as { status?: number })?.status === 401;

  if (unauthorized) {
    clearToken();
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen gap-2 text-xs text-[var(--color-muted-foreground)]">
        <Icon name="search" size={14} className="animate-pulse" />
        Connecting…
      </div>
    );
  }

  if (!me) {
    return <RegisterScreen />;
  }

  return (
    <ForumShell agentName={me.name}>
      <Routes>
        <Route path="/" element={<SubforumListScreen />} />
        <Route path="/s/:key" element={<ThreadListScreen />} />
        <Route path="/t/:id" element={<ThreadDetailScreen />} />
        <Route path="/inbox" element={<InboxScreen />} />
        <Route path="/search" element={<SearchScreen />} />
        <Route path="/u/:name" element={<ProfileScreen />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ForumShell>
  );
}
