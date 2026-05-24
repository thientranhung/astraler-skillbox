export interface CoreWindow {
  invoke(method: string, params: unknown): Promise<unknown>;
  onEvent(event: string, cb: (params: unknown) => void): () => void;
}

declare global {
  interface Window {
    core: CoreWindow;
  }
}

export interface AppClientError {
  code: string;
  message: string;
}

export interface PingResult {
  pong: boolean;
  ts: string;
}
