import { ImmutableMap } from "typings";
import Immutable from "immutable";

export const LOAD_ROLE_BINDINGS_PENDING = "LOAD_ROLE_BINDINGS_PENDING";
export const LOAD_ROLE_BINDINGS_FAILED = "LOAD_ROLE_BINDINGS_FAILED";
export const LOAD_ROLE_BINDINGS_FULFILLED = "LOAD_ROLE_BINDINGS_FULFILLED";

export const CREATE_ROLE_BINDINGS_PENDING = "CREATE_ROLE_BINDINGS_PENDING";
export const CREATE_ROLE_BINDINGS_FAILED = "CREATE_ROLE_BINDINGS_FAILED";
export const CREATE_ROLE_BINDINGS_FULFILLED = "CREATE_ROLE_BINDINGS_FULFILLED";

export const DELETE_ROLE_BINDINGS_PENDING = "DELETE_ROLE_BINDINGS_PENDING";
export const DELETE_ROLE_BINDINGS_FAILED = "DELETE_ROLE_BINDINGS_FAILED";
export const DELETE_ROLE_BINDINGS_FULFILLED = "DELETE_ROLE_BINDINGS_FULFILLED";

export interface RoleBindingContent {
  name: string;
  namespace: string;
  subject: string;
  role: string;
  expiredAtTimestamp: number;
}

export type RoleBinding = ImmutableMap<RoleBindingContent>;

export const newEmptyRoleBindingContent = (): RoleBindingContent => {
  return {
    name: "",
    namespace: "",
    subject: "",
    role: "",
    expiredAtTimestamp: 1,
  };
};

export interface RoleBindingRequestStatusAction {
  type:
    | typeof LOAD_ROLE_BINDINGS_PENDING
    | typeof LOAD_ROLE_BINDINGS_FAILED
    | typeof CREATE_ROLE_BINDINGS_PENDING
    | typeof CREATE_ROLE_BINDINGS_FAILED
    | typeof DELETE_ROLE_BINDINGS_PENDING
    | typeof DELETE_ROLE_BINDINGS_FAILED;
}

export interface LoadRoleBindingsAction {
  type: typeof LOAD_ROLE_BINDINGS_FULFILLED;
  payload: {
    roleBindings: Immutable.List<RoleBinding>;
  };
}

export interface CreateRoleBindingAction {
  type: typeof CREATE_ROLE_BINDINGS_FULFILLED;
}

export interface DeleteRoleBindingAction {
  type: typeof DELETE_ROLE_BINDINGS_FULFILLED;
}

export type RoleBindingsActions =
  | LoadRoleBindingsAction
  | CreateRoleBindingAction
  | DeleteRoleBindingAction
  | RoleBindingRequestStatusAction;
