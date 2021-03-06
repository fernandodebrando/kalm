import { Box, Button, createStyles, Theme, withStyles, WithStyles } from "@material-ui/core";
import { Expansion } from "forms/Route/expansion";
import { withComponent, WithComponentProp } from "hoc/withComponent";
import { withRoutesData, WithRoutesDataProps } from "hoc/withRoutesData";
import { ApplicationSidebar } from "pages/Application/ApplicationSidebar";
import { BasePage } from "pages/BasePage";
import { ComponentBasicInfo } from "pages/Components/BasicInfo";
import { PodsTable } from "pages/Components/PodsTable";
import { RouteWidgets } from "pages/Route/Widget";
import React from "react";
import { connect } from "react-redux";
import { Link } from "react-router-dom";
import { RootState } from "reducers";
import { WorkloadType } from "types/componentTemplate";
import { Body, H6 } from "widgets/Label";
import { Namespaces } from "widgets/Namespaces";
import { VerticalHeadTable } from "widgets/VerticalHeadTable";
import { withUserAuth, WithUserAuthProps } from "hoc/withUserAuth";

const styles = (theme: Theme) =>
  createStyles({
    secondHeaderRight: {
      height: "100%",
      width: "100%",
      display: "flex",
      alignItems: "center",
    },
    secondHeaderRightItem: {
      marginLeft: theme.spacing(2),
      marginRight: theme.spacing(2),
    },
  });

const mapStateToProps = (state: RootState) => {
  return {};
};

interface Props
  extends WithStyles<typeof styles>,
    ReturnType<typeof mapStateToProps>,
    WithComponentProp,
    WithUserAuthProps,
    WithRoutesDataProps {}

interface State {}

class ComponentShowRaw extends React.PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {};
  }
  private renderNetwork() {
    const { component, activeNamespaceName } = this.props;
    const hasService = component.ports && component.ports!.length > 0;
    return (
      <Expansion title={"Networking"} defaultUnfold>
        <Box pb={2}>
          <Body>
            Cluster FQDN DNS:{" "}
            <strong>{hasService ? `${component.name}.${activeNamespaceName}.svc.cluster.local` : "none"}</strong>
          </Body>
          <Body>
            Cluster DNS: <strong>{hasService ? `${component.name}.${activeNamespaceName}` : "none"}</strong>
          </Body>
          <Body>
            Namespace DNS: <strong>{hasService ? `${component.name}` : "none"}</strong>
          </Body>
        </Box>
        {component.ports && (
          <VerticalHeadTable
            items={component.ports?.map((port) => ({
              name: "Exposed port: " + port.protocol,
              content: (
                <span>
                  Expose port <strong>{port.containerPort}</strong> to cluster port{" "}
                  <strong>{port.servicePort || port.containerPort}</strong>
                </span>
              ),
            }))}
          />
        )}
      </Expansion>
    );
  }

  private renderRoutes() {
    const { httpRoutes, component, activeNamespaceName, canEditNamespace } = this.props;

    const serviceName = `${component.name}`;

    const routes = httpRoutes.filter(
      (route) =>
        route.destinations.filter((destination) => destination.host.startsWith(serviceName + "." + activeNamespaceName))
          .length > 0,
    );
    return (
      <Expansion title={"Routes"} defaultUnfold>
        <RouteWidgets routes={routes} canEdit={canEditNamespace(activeNamespaceName)} />
      </Expansion>
    );
  }
  private renderPods() {
    const { component, activeNamespaceName, canEditNamespace } = this.props;

    return (
      <Expansion title="pods" defaultUnfold>
        <PodsTable
          activeNamespaceName={activeNamespaceName}
          pods={component.pods}
          workloadType={component.workloadType as WorkloadType}
          canEdit={canEditNamespace(activeNamespaceName)}
        />
      </Expansion>
    );
  }

  private renderSecondHeaderRight() {
    const { classes, component, activeNamespaceName, canEditNamespace } = this.props;

    return (
      <div className={classes.secondHeaderRight}>
        <H6 className={classes.secondHeaderRightItem}>Component {component.name}</H6>
        {canEditNamespace(activeNamespaceName) && (
          <Button
            tutorial-anchor-id="edit-component"
            component={Link}
            color="primary"
            size="small"
            variant="outlined"
            to={`/applications/${activeNamespaceName}/components/${component.name}/edit`}
          >
            Edit
          </Button>
        )}
      </div>
    );
  }

  public render() {
    const { component, activeNamespaceName } = this.props;
    return (
      <BasePage
        secondHeaderRight={this.renderSecondHeaderRight()}
        secondHeaderLeft={<Namespaces />}
        leftDrawer={<ApplicationSidebar />}
      >
        <Box p={2}>
          <Expansion title={"Basic"} defaultUnfold>
            <ComponentBasicInfo component={component} activeNamespaceName={activeNamespaceName} />
          </Expansion>
          {this.renderPods()}
          {this.renderNetwork()}
          {this.renderRoutes()}
        </Box>
      </BasePage>
    );
  }
}

export const ComponentShowPage = withStyles(styles)(
  withUserAuth(connect(mapStateToProps)(withRoutesData(withComponent(ComponentShowRaw)))),
);
