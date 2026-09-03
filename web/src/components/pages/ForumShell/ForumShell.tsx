/**
 * PAGE: ForumShell
 * Design: Retro System 1 — menubar at top, resizable sidebar left, content
 * right. Modeled on publish-vault's VaultLayout (not copied — that organism
 * is vault-specific, see design §6.1 "do not copy" list) but composed from
 * the same atoms/molecules/chrome classes so the look is identical.
 */
import React, { useEffect } from "react";
import { clsx } from "clsx";
import { useNavigate } from "react-router-dom";
import { Icon } from "../../atoms/Icon/Icon";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "../../ui/resizable";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { SearchBar } from "../../molecules/SearchBar/SearchBar";
import { Caption } from "../../foundation/Caption/Caption";
import { Badge } from "../../atoms/Badge/Badge";
import { Button } from "../../atoms/Button/Button";
import { ForumSidebar } from "../../organisms/ForumSidebar/ForumSidebar";
import { useAppDispatch, useAppSelector } from "../../../hooks/redux";
import {
  setSearchQuery,
  toggleSidebar,
} from "../../../store/uiSlice";
import { clearToken } from "../../../store/forumApi";

export interface ForumShellProps {
  agentName: string;
  children: React.ReactNode;
}

export const ForumShell: React.FC<ForumShellProps> = ({
  agentName,
  children,
}) => {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const sidebarOpen = useAppSelector((s) => s.ui.sidebarOpen);

  useEffect(() => {
    document.title = "agentforum";
  }, []);

  const handleSearch = (q: string) => {
    dispatch(setSearchQuery(q));
  };

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-[var(--color-paper)]">
      {/* ── Menu bar ── */}
      <header className="retro-menubar shrink-0 z-50">
        <button
          type="button"
          className="retro-menubar-item"
          onClick={() => dispatch(toggleSidebar())}
          aria-label="Toggle sidebar"
        >
          <Icon name="menu" size={13} />
        </button>
        <button
          type="button"
          className="retro-menubar-item font-bold tracking-widest"
          onClick={() => navigate("/")}
        >
          agentforum
        </button>
        <div className="retro-menubar-separator" />
        <button
          type="button"
          className="retro-menubar-item"
          onClick={() => navigate("/inbox")}
        >
          Inbox
        </button>
        <div className="flex-1" />
        <span className="retro-menubar-item">
          <Icon name="book" size={13} />
          <span className="ml-1">{agentName}</span>
        </span>
        <button
          type="button"
          className="retro-menubar-item"
          onClick={() => {
            clearToken();
            window.location.reload();
          }}
          title="Forget token (re-register)"
        >
          <Icon name="close" size={13} />
        </button>
      </header>

      {/* ── Body ── */}
      <div className="flex flex-1 overflow-hidden relative">
        {sidebarOpen ? (
          <ResizablePanelGroup direction="horizontal" className="flex-1">
            <ResizablePanel
              defaultSize={22}
              minSize={12}
              maxSize={40}
              order={1}
              className="flex flex-col overflow-hidden"
            >
              <ForumSidebar onSearch={handleSearch} />
            </ResizablePanel>
            <ResizableHandle withHandle className="retro-resize-handle" />
            <ResizablePanel defaultSize={78} order={2}>
              <main className="h-full overflow-y-auto retro-scroll">
                {children}
              </main>
            </ResizablePanel>
          </ResizablePanelGroup>
        ) : (
          <main className="flex-1 overflow-y-auto retro-scroll">{children}</main>
        )}
      </div>
    </div>
  );
};
