import React from "react";
import { connect } from "react-redux";
import { RootState } from "reducers";
import { TDispatchProp } from "types";

const mapStateToProps = (state: RootState) => {
  const permissionMethods = state.get("auth").get("permissionMethods");
  const impersonation = state.get("auth").get("impersonation");
  const authToken = state.get("auth").get("token");
  return { authToken, impersonation, ...permissionMethods };
};

export interface WithUserAuthProps extends ReturnType<typeof mapStateToProps>, TDispatchProp {}

export const withUserAuth = (WrappedComponent: React.ComponentType<any>) => {
  const HOC: React.ComponentType<WithUserAuthProps> = class extends React.PureComponent<WithUserAuthProps> {
    render() {
      return <WrappedComponent {...this.props} />;
    }
  };

  HOC.displayName = `withUserAuth(${getDisplayName(WrappedComponent)})`;

  return connect(mapStateToProps)(HOC);
};

function getDisplayName(WrappedComponent: React.ComponentType<any>) {
  return WrappedComponent.displayName || WrappedComponent.name || "Component";
}
