/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
export type AuthorizeRequest = {
  token?: string
  accessResource?: string
  capability?: string
  organizationId?: string
  projectId?: string
}

export type AuthorizeResponse = {
  authorized?: boolean
  user?: User
  organization?: Organization
  project?: Project
  tenantId?: string
}

export type AuthorizeWorkerRequest = {
  token?: string
}

export type AuthorizeWorkerResponse = {
  authorized?: boolean
  cluster?: Cluster
  tenantId?: string
}

export type User = {
  id?: string
}

export type Organization = {
  id?: string
}

export type ProjectAssignedKubernetesEnv = {
  clusterId?: string
  namespace?: string
}

export type Project = {
  id?: string
  assignedKubernetesEnvs?: ProjectAssignedKubernetesEnv[]
}

export type Cluster = {
  id?: string
}

export class RbacInternalService {
  static Authorize(req: AuthorizeRequest, initReq?: fm.InitReq): Promise<AuthorizeResponse> {
    return fm.fetchReq<AuthorizeRequest, AuthorizeResponse>(`/llmariner.rbac.server.v1.RbacInternalService/Authorize`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static AuthorizeWorker(req: AuthorizeWorkerRequest, initReq?: fm.InitReq): Promise<AuthorizeWorkerResponse> {
    return fm.fetchReq<AuthorizeWorkerRequest, AuthorizeWorkerResponse>(`/llmariner.rbac.server.v1.RbacInternalService/AuthorizeWorker`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}