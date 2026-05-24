import { useState } from "react";
import { methods } from "./lib/core-client/client.js";
import type { PingResult } from "./lib/core-client/types.js";

export default function App(): JSX.Element {
  const [result, setResult] = useState<PingResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handlePing(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const res = await methods.ping();
      setResult(res);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ fontFamily: "monospace", padding: "2rem" }}>
      <h1>Astraler Skillbox</h1>
      <button onClick={handlePing} disabled={loading}>
        {loading ? "Pinging…" : "Ping Go"}
      </button>
      {result && (
        <pre style={{ marginTop: "1rem" }}>{JSON.stringify(result, null, 2)}</pre>
      )}
      {error && (
        <pre style={{ marginTop: "1rem", color: "red" }}>{error}</pre>
      )}
    </div>
  );
}
