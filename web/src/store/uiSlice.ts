import { createSlice, PayloadAction } from "@reduxjs/toolkit";

interface UIState {
  sidebarOpen: boolean;
  rightPanelOpen: boolean;
  searchQuery: string;
  activeThread: string | null;
}

// Keep the initial state independent of browser-only viewport data so the
// first render is stable (and SSR-safe if a server entry is added later).
const initialState: UIState = {
  sidebarOpen: true,
  rightPanelOpen: true,
  searchQuery: "",
  activeThread: null,
};

// Adapted from publish-vault's uiSlice (design §6.1): same shell fields,
// forum domain instead of note domain.
const uiSlice = createSlice({
  name: "ui",
  initialState,
  reducers: {
    toggleSidebar(state) {
      state.sidebarOpen = !state.sidebarOpen;
    },
    setSidebarOpen(state, action: PayloadAction<boolean>) {
      state.sidebarOpen = action.payload;
    },
    toggleRightPanel(state) {
      state.rightPanelOpen = !state.rightPanelOpen;
    },
    setRightPanelOpen(state, action: PayloadAction<boolean>) {
      state.rightPanelOpen = action.payload;
    },
    setSearchQuery(state, action: PayloadAction<string>) {
      state.searchQuery = action.payload;
    },
    setActiveThread(state, action: PayloadAction<string | null>) {
      state.activeThread = action.payload;
    },
  },
});

export const {
  toggleSidebar,
  setSidebarOpen,
  toggleRightPanel,
  setRightPanelOpen,
  setSearchQuery,
  setActiveThread,
} = uiSlice.actions;

export default uiSlice.reducer;
