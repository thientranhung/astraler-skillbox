import { invoke } from "./client.js";
import type {
  HostChooseRequest,
  HostChooseResponse,
  HostScanRequest,
  HostScanResponse,
  SkillListRequest,
  SkillListResponse,
  SettingsGetResponse,
  ProjectAddRequest,
  ProjectAddResponse,
  ProjectListResponse,
  ProjectGetRequest,
  ProjectGetResponse,
  ProjectScanRequest,
  ProjectScanResponse,
  ProjectRemoveRequest,
  ProjectRemoveResponse,
} from "@contracts/index.js";

export const methods = {
  openHostFolder: () =>
    invoke<{ path: string | null }>("dialog.openHostFolder", {}),

  openProjectFolder: () =>
    invoke<{ path: string | null }>("dialog.openProjectFolder", {}),

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

  addProject: (req: ProjectAddRequest) =>
    invoke<ProjectAddResponse>("project.add", req),

  listProjects: () =>
    invoke<ProjectListResponse>("project.list", {}),

  getProject: (req: ProjectGetRequest) =>
    invoke<ProjectGetResponse>("project.get", req),

  scanProject: (req: ProjectScanRequest) =>
    invoke<ProjectScanResponse>("project.scan", req),

  removeProject: (req: ProjectRemoveRequest) =>
    invoke<ProjectRemoveResponse>("project.remove", req),

  openPath: (path: string) =>
    invoke<{ opened: boolean }>("dialog.openPath", { path }),
};
