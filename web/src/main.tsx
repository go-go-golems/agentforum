// main.tsx — browser entry point (no SSR in this milestone; makeStore is
// SSR-safe if that changes, see design §6.1).
import React from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Provider } from "react-redux";
import { makeStore } from "./store/store";
import { App } from "./App";
import "./index.css";

const store = makeStore();
const root = document.getElementById("root")!;

createRoot(root).render(
  <React.StrictMode>
    <Provider store={store}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </Provider>
  </React.StrictMode>
);
