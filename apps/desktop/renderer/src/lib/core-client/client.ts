export class AppClientError extends Error {
  constructor(
    public readonly code: string,
    public readonly userMessage: string,
    public readonly technicalMessage: string,
    public readonly rpcCode?: number,
  ) {
    super(userMessage);
    this.name = "AppClientError";
  }
}

function extractJsonObject(message: string): string | null {
  const start = message.indexOf("{");
  const end = message.lastIndexOf("}");
  if (start === -1 || end === -1 || end <= start) return null;
  return message.slice(start, end + 1);
}

export async function invoke<TRes = unknown>(method: string, params: unknown): Promise<TRes> {
  if (!window?.core) {
    throw new AppClientError("client_error", "Core not available", "window.core is not defined");
  }

  try {
    return (await window.core.invoke(method, params)) as TRes;
  } catch (err) {
    if (err != null && typeof err === "object" && "code" in err && "userMessage" in err) {
      const appErr = err as {
        code?: string;
        rpcCode?: number;
        userMessage?: string;
        technicalMessage?: string;
      };
      throw new AppClientError(
        appErr.code ?? "unknown_error",
        appErr.userMessage ?? "Unknown error",
        appErr.technicalMessage ?? "",
        appErr.rpcCode,
      );
    }

    const msg = err instanceof Error ? err.message : String(err);

    // Try to parse a JSON-RPC error envelope: { code, message, data: AppError }.
    // Electron may prefix rejected IPC errors with "Error invoking remote method...";
    // extract the JSON object when that happens.
    try {
      const parsed = JSON.parse(extractJsonObject(msg) ?? msg) as {
        code?: number;
        message?: string;
        data?: {
          code?: string;
          rpcCode?: number;
          userMessage?: string;
          technicalMessage?: string;
        };
      };
      if (parsed.data?.code) {
        throw new AppClientError(
          parsed.data.code,
          parsed.data.userMessage ?? parsed.message ?? "Unknown error",
          parsed.data.technicalMessage ?? "",
          parsed.data.rpcCode,
        );
      }
    } catch (inner) {
      if (inner instanceof AppClientError) throw inner;
      // Not parseable as structured error — fall through
    }

    throw new AppClientError("client_error", msg, msg);
  }
}
