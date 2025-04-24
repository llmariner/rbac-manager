import * as fm from "../../fetch.pb";
export type AuthorizeRequest = {
    token?: string;
    accessResource?: string;
    capability?: string;
    organizationId?: string;
    projectId?: string;
};
export type AuthorizeResponse = {
    authorized?: boolean;
    user?: User;
    organization?: Organization;
    project?: Project;
    tenantId?: string;
    apiKeyId?: string;
    excludedFromRateLimiting?: boolean;
};
export type AuthorizeWorkerRequest = {
    token?: string;
};
export type AuthorizeWorkerResponse = {
    authorized?: boolean;
    cluster?: Cluster;
    tenantId?: string;
};
export type User = {
    id?: string;
    internalId?: string;
};
export type Organization = {
    id?: string;
    title?: string;
};
export type ProjectAssignedKubernetesEnv = {
    clusterId?: string;
    clusterName?: string;
    namespace?: string;
};
export type Project = {
    id?: string;
    title?: string;
    assignedKubernetesEnvs?: ProjectAssignedKubernetesEnv[];
};
export type Cluster = {
    id?: string;
    name?: string;
};
export declare class RbacInternalService {
    static Authorize(req: AuthorizeRequest, initReq?: fm.InitReq): Promise<AuthorizeResponse>;
    static AuthorizeWorker(req: AuthorizeWorkerRequest, initReq?: fm.InitReq): Promise<AuthorizeWorkerResponse>;
}
