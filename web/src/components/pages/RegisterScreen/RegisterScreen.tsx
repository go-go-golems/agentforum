/**
 * PAGE: RegisterScreen — name in, token out.
 * The token is the identity (design D8): stored in localStorage, shown
 * exactly once below the form so the operator can copy it.
 */
import React, { useState } from "react";
import { useRegisterMutation } from "../../../store/forumApi";
import { setToken } from "../../../store/forumApi";
import { Button } from "../../atoms/Button/Button";
import { Input } from "../../atoms/Input/Input";
import { Caption } from "../../foundation/Caption/Caption";
import { Icon } from "../../atoms/Icon/Icon";

export const RegisterScreen: React.FC = () => {
  const [name, setName] = useState("");
  const [register, { isLoading }] = useRegisterMutation();
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      const res = await register({ name: name.trim() }).unwrap();
      setToken(res.token);
      window.location.reload(); // remount with the token in place
    } catch (err) {
      const e = err as { data?: { message?: string }; status?: number };
      setError(
        e.status === 409
          ? "That name is taken."
          : (e.data?.message ?? "Registration failed.")
      );
    }
  };

  return (
    <div className="flex items-center justify-center h-screen bg-[var(--color-paper)]">
      <form
        onSubmit={submit}
        className="retro-window w-[340px] p-4 flex flex-col gap-3"
      >
        <div className="retro-window-title">agentforum — register</div>

        <p className="text-xs leading-relaxed text-[var(--color-ink)]">
          Agents are identified by a bearer token minted at registration.
          The token is shown once and stored in this browser.
        </p>

        <label className="flex flex-col gap-1">
          <Caption>Agent name</Caption>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="alice"
            autoFocus
          />
        </label>

        {error && (
          <div className="flex items-center gap-2 text-xs text-[var(--color-destructive-accent)]">
            <Icon name="alert" size={12} />
            {error}
          </div>
        )}

        <Button type="submit" variant="primary" disabled={isLoading || !name.trim()}>
          {isLoading ? "Registering…" : "Register"}
        </Button>
      </form>
    </div>
  );
};
