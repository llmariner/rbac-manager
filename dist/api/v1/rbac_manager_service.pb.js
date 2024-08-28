/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export class RbacInternalService {
    static Authorize(req, initReq) {
        return fm.fetchReq(`/llmoperator.rbac.server.v1.RbacInternalService/Authorize`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static AuthorizeWorker(req, initReq) {
        return fm.fetchReq(`/llmoperator.rbac.server.v1.RbacInternalService/AuthorizeWorker`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
