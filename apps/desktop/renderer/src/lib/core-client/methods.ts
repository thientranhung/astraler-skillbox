import { invoke } from "./client.js";
import type {
  HostChooseRequest,
  HostChooseResponse,
  HostScanRequest,
  HostScanResponse,
  SkillListRequest,
  SkillListResponse,
  SkillGetRequest,
  SkillGetResponse,
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
  InstallSkillRequest,
  InstallSkillResponse,
  RemoveSkillRequest,
  RemoveSkillResponse,
  DashboardGetResponse,
  GlobalScanResponse,
  GlobalListResponse,
  ProviderListResponse,
  ProviderUpdatePathsRequest,
  ProviderUpdatePathsResponse,
  ProviderResetPathsRequest,
  ProviderResetPathsResponse,
  ProviderSetEnabledRequest,
  ProviderSetEnabledResponse,
  ProviderPluginScanGlobalResponse,
  ProviderPluginScanProjectRequest,
  ProviderPluginScanProjectResponse,
  ProviderPluginListResponse,
  ProviderPluginSetEnabledRequest,
  ProviderPluginSetEnabledResponse,
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

  getSkill: (req: SkillGetRequest) =>
    invoke<SkillGetResponse>("skill.get", req),

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

  installSkill: (req: InstallSkillRequest) =>
    invoke<InstallSkillResponse>("install.skill", req),

  removeSkill: (req: RemoveSkillRequest) =>
    invoke<RemoveSkillResponse>("remove.skill", req),

  openPath: (path: string) =>
    invoke<{ opened: boolean }>("dialog.openPath", { path }),

  openTerminal: (path: string) =>
    invoke<{ opened: boolean }>("dialog.openTerminal", { path }),

  getDashboard: () => invoke<DashboardGetResponse>("dashboard.get", {}),

  scanGlobal: () => invoke<GlobalScanResponse>("global.scan", {}),

  listGlobal: () => invoke<GlobalListResponse>("global.list", {}),

  listProviders: () => invoke<ProviderListResponse>("provider.list", {}),

  updateProviderPaths: (req: ProviderUpdatePathsRequest) =>
    invoke<ProviderUpdatePathsResponse>("provider.updatePaths", req),

  resetProviderPaths: (req: ProviderResetPathsRequest) =>
    invoke<ProviderResetPathsResponse>("provider.resetPaths", req),

  setProviderEnabled: (req: ProviderSetEnabledRequest) =>
    invoke<ProviderSetEnabledResponse>("provider.setEnabled", req),

  scanProviderPluginsGlobal: () =>
    invoke<ProviderPluginScanGlobalResponse>("providerPlugin.scanGlobal", {}),

  scanProviderPluginsProject: (req: ProviderPluginScanProjectRequest) =>
    invoke<ProviderPluginScanProjectResponse>("providerPlugin.scanProject", req),

  listProviderPlugins: () =>
    invoke<ProviderPluginListResponse>("providerPlugin.list", {}),

  setProviderPluginEnabled: (req: ProviderPluginSetEnabledRequest) =>
    invoke<ProviderPluginSetEnabledResponse>("providerPlugin.setEnabled", req),
};
