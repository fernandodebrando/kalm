import { Box, Button, createStyles, Theme, Typography, WithStyles, withStyles } from "@material-ui/core";
import { indigo } from "@material-ui/core/colors";
import { deleteDeployAccessTokenAction } from "actions/deployAccessToken";
import { blinkTopProgressAction } from "actions/settings";
import { withDeployAccessTokens, WithDeployAccessTokensProps } from "hoc/withDeployAccessTokens";
import { BasePage } from "pages/BasePage";
import React from "react";
import { connect } from "react-redux";
import { Link } from "react-router-dom";
import { RootState } from "reducers";
import {
  DeployAccessToken,
  DeployAccessTokenScopeCluster,
  DeployAccessTokenScopeComponent,
  DeployAccessTokenScopeNamespace,
} from "types/deployAccessToken";
import sc from "utils/stringConstants";
import { BlankTargetLink } from "widgets/BlankTargetLink";
import { CustomizedButton } from "widgets/Button";
import { EmptyInfoBox } from "widgets/EmptyInfoBox";
import { CIIcon, KalmDetailsIcon } from "widgets/Icon";
import { IconLinkWithToolTip } from "widgets/IconButtonWithTooltip";
import { DeleteButtonWithConfirmPopover } from "widgets/IconWithPopover";
import { InfoBox } from "widgets/InfoBox";
import { KRTable } from "widgets/KRTable";
import { Loading } from "widgets/Loading";
import { withUserAuth, WithUserAuthProps } from "hoc/withUserAuth";
import { withNamespace, WithNamespaceProps } from "hoc/withNamespace";

const styles = (theme: Theme) =>
  createStyles({
    root: {},
  });

const mapStateToProps = (state: RootState) => {
  return {};
};

interface Props
  extends WithStyles<typeof styles>,
    ReturnType<typeof mapStateToProps>,
    WithDeployAccessTokensProps,
    WithNamespaceProps,
    WithUserAuthProps {}

interface State {}

class CIPageRaw extends React.PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {};
  }

  private renderEmpty() {
    return (
      <EmptyInfoBox
        image={<CIIcon style={{ height: 120, width: 120, color: indigo[200] }} />}
        title={sc.EMPTY_CI_TITLE}
        content={sc.EMPTY_CI_SUBTITLE}
        button={
          this.canEdit() ? (
            <CustomizedButton component={Link} variant="contained" to="/ci/keys/new" color="primary">
              New Deploy Key
            </CustomizedButton>
          ) : null
        }
      />
    );
  }

  private renderMemo = (rowData: DeployAccessToken) => {
    return <Typography variant="subtitle2">{rowData.memo}</Typography>;
  };

  private renderScope = (rowData: DeployAccessToken) => {
    switch (rowData.scope) {
      case DeployAccessTokenScopeCluster: {
        return "Cluster";
      }
      case DeployAccessTokenScopeNamespace: {
        return "Specific Applications";
      }
      case DeployAccessTokenScopeComponent: {
        return "Specific Components";
      }
    }
  };

  private renderResources = (rowData: DeployAccessToken) => {
    const resoureces = rowData.resources;

    switch (rowData.scope) {
      case DeployAccessTokenScopeCluster: {
        return "-";
      }
      case DeployAccessTokenScopeNamespace: {
        return resoureces.map((r) => <Box key={r}>{r}</Box>);
      }
      case DeployAccessTokenScopeComponent: {
        return resoureces.map((r) => <Box key={r}>{r}</Box>);
      }
    }
  };

  private renderActions = (rowData: DeployAccessToken) => {
    const { dispatch } = this.props;
    return this.canEdit() ? (
      <>
        <IconLinkWithToolTip
          onClick={() => {
            blinkTopProgressAction();
          }}
          // size="small"
          tooltipTitle="Details"
          to={`/ci/keys/${rowData.name}`}
        >
          <KalmDetailsIcon />
        </IconLinkWithToolTip>
        <DeleteButtonWithConfirmPopover
          popupId="delete-ci-popup"
          popupTitle="DELETE CI?"
          confirmedAction={() => dispatch(deleteDeployAccessTokenAction(rowData))}
        />
      </>
    ) : null;
  };

  private renderInfoBox() {
    return (
      <Box mt={2}>
        <InfoBox
          title={sc.CI_INFO_BOX_TEXT}
          options={[
            {
              title: <BlankTargetLink href="https://kalm.dev">Use webhook (fix the link)</BlankTargetLink>,
              content: "",
            },
            {
              title: <BlankTargetLink href="https://kalm.dev">Use Kalm orb in CircleCI (fix the link)</BlankTargetLink>,
              content: "",
            },
            {
              title: (
                <BlankTargetLink href="https://kalm.dev">
                  Use Kalm action in Github Actions (fix the link)
                </BlankTargetLink>
              ),
              content: "",
            },
          ]}
        />
      </Box>
    );
  }

  private getKRTableColumns() {
    return [
      {
        Header: "Memo",
        accessor: "memo",
      },
      {
        Header: "Scope",
        accessor: "scope",
      },
      {
        Header: "Resources",
        accessor: "resources",
      },
      {
        Header: "Action",
        accessor: "actions",
      },
    ];
  }

  private getKRTableData() {
    const { deployAccessTokens } = this.props;
    const data: any[] = [];

    deployAccessTokens &&
      deployAccessTokens.forEach((deployAccessToken, index) => {
        const rowData = deployAccessToken;
        data.push({
          memo: this.renderMemo(rowData),
          scope: this.renderScope(rowData),
          resources: this.renderResources(rowData),
          actions: this.renderActions(rowData),
        });
      });

    return data;
  }

  private renderKRTable() {
    return <KRTable showTitle={true} title="CI/CD" columns={this.getKRTableColumns()} data={this.getKRTableData()} />;
  }

  private renderContent = () => {
    const { deployAccessTokens, isLoading, loaded } = this.props;

    if (!loaded && isLoading) {
      return (
        <Box p={2}>
          <Loading />
        </Box>
      );
    }

    return <Box p={2}>{deployAccessTokens.length === 0 ? this.renderEmpty() : this.renderKRTable()}</Box>;
  };

  private canEdit() {
    const { canEditNamespace, canEditCluster, activeNamespaceName } = this.props;
    return canEditNamespace(activeNamespaceName) || canEditCluster();
  }

  public render() {
    return (
      <BasePage
        secondHeaderRight={
          this.canEdit() ? (
            <>
              <Button component={Link} color="primary" variant="outlined" size="small" to="/ci/keys/new">
                New Deploy Token
              </Button>
            </>
          ) : null
        }
      >
        {this.renderContent()}
      </BasePage>
    );
  }
}

export const CIPage = withNamespace(
  withUserAuth(withStyles(styles)(withDeployAccessTokens(connect(mapStateToProps)(CIPageRaw)))),
);
