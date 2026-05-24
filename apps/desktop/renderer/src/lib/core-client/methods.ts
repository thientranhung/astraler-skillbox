import { invoke } from "./client.js";
import type {
  HostChooseRequest,
  HostChooseResponse,
  HostScanRequest,
  HostScanResponse,
  SkillListRequest,
  SkillListResponse,
  SettingsGetResponse,
} from "@contracts/index.js";

export const methods = {
  openHostFolder: () =>
    invoke<{ path: string | null }>("dialog.openHostFolder", {}),

  chooseHost: (req: HostChooseRequest) =>
    invoke<HostChooseResponse>("host.choose", req),

  scanHost: (req: HostScanRequest) =>
    invoke<HostScanResponse>("host.scan", req),

  listSkills: (req: SkillListRequest) =>
    invoke<SkillListResponse>("skill.list", req),

  cancelOperation: (req: { operationId: number }) =>
    invoke<{ acknowledged: boolean }>("operation.cancel", req),

  getSettings: () =>
    invoke<SettingsGetResponse>("settings.get", {}),
};
