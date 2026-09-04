/**
 * A3 — W7 register-token anomaly probe (AGENTFORUM-005).
 *
 * During W7 verification one register flow stored an invalid token
 * (getMe 401 -> clearToken). The setToken af_-prefix guard was added
 * blind; these tests prove the property the guard defends: the client
 * cannot manufacture a non-af_ token from any RegisterAgentResponse the
 * decoder will emit — well-formed or malformed.
 *
 * Case analysis pinned here:
 *  1. response without a token field  -> proto3 default ""  -> guard rejects
 *  2. response with a non-string token -> fromJson throws    -> register
 *     rejects, setToken is never called
 *  3. response with a valid af_ token  -> round-trips       -> stored
 *  4. the W7 symptom string "undefined" and a bare "af_" prefix
 *     -> guard rejects (tightened in A3: the old startsWith("af_") check
 *     accepted the bare prefix)
 */
import { beforeAll, describe, expect, it } from "vitest";
import { fromJson } from "@bufbuild/protobuf";
import type { JsonObject } from "@bufbuild/protobuf";
import { RegisterAgentResponseSchema } from "../pb/agentforum/v1/service_pb";
import { getToken, setToken, clearToken } from "./forumApi";

// The API slice reads localStorage inside getToken/setToken only (never at
// import time), so a minimal stub installed before the first call suffices
// in the node test environment.
beforeAll(() => {
  const store = new Map<string, string>();
  const stub: Storage = {
    getItem: (k) => store.get(k) ?? null,
    setItem: (k, v) => void store.set(k, v),
    removeItem: (k) => void store.delete(k),
    clear: () => void store.clear(),
    key: (i) => Array.from(store.keys())[i] ?? null,
    get length() {
      return store.size;
    },
  };
  Object.defineProperty(globalThis, "localStorage", { value: stub, configurable: true });
});

const agentBody = {
  id: "agt_01",
  name: "alice",
  createdAt: "2026-09-04T00:00:00Z",
};

describe("A3: register-token anomaly probe", () => {
  it("decodes a response without a token field to an empty token (proto3 default), never to a fabricated string", () => {
    const resp = fromJson(RegisterAgentResponseSchema, {
      schemaVersion: 1,
      agent: agentBody,
    } as JsonObject);
    expect(resp.token).toBe("");
  });

  it("an empty decoded token is never stored", () => {
    clearToken();
    const resp = fromJson(RegisterAgentResponseSchema, {
      schemaVersion: 1,
      agent: agentBody,
    } as JsonObject);
    setToken(resp.token); // must be a no-op
    expect(getToken()).toBe("");
  });

  it("a response with a non-string token fails decoding, so the register promise rejects and setToken is never reached", () => {
    expect(() =>
      fromJson(RegisterAgentResponseSchema, {
        schemaVersion: 1,
        agent: agentBody,
        token: 12345, // protojson requires a JSON string
      } as unknown as JsonObject)
    ).toThrow();
  });

  it("the W7 symptom string is rejected by the guard", () => {
    clearToken();
    setToken("undefined"); // the pre-guard localStorage.setItem(undefined) symptom
    setToken("af_"); // prefix alone is not a token
    expect(getToken()).toBe("");
  });

  it("a valid af_ token round-trips through decode, setToken, and getToken", () => {
    clearToken();
    const resp = fromJson(RegisterAgentResponseSchema, {
      schemaVersion: 1,
      agent: agentBody,
      token: "af_probe_ok",
    } as JsonObject);
    setToken(resp.token);
    expect(getToken()).toBe("af_probe_ok");
  });
});
