import { createStyles, List, ListItem, ListItemIcon, ListItemText, Theme } from "@material-ui/core";
import AppsIcon from "@material-ui/icons/Apps";
import { WithStyles, withStyles } from "@material-ui/styles";
import React from "react";
import { connect } from "react-redux";
import { NavLink, RouteComponentProps, withRouter } from "react-router-dom";
import { RootState } from "reducers";
import { TDispatch } from "types";
import { blinkTopProgressAction } from "actions/settings";
import { DashboardIcon, KalmComponentsIcon, PeopleIcon } from "widgets/Icon";
import sc from "utils/stringConstants";
import { withUserAuth, WithUserAuthProps } from "hoc/withUserAuth";

const mapStateToProps = (state: RootState) => {
  return {
    activeNamespaceName: state.namespaces.active,
  };
};

const styles = (theme: Theme) =>
  createStyles({
    listItem: {
      height: 40,

      "& > .MuiListItemIcon-root": {
        minWidth: 40,
        marginLeft: -4,
      },
      borderLeft: `4px solid transparent`,
    },
    listItemSelected: {
      borderLeft: `4px solid ${
        theme.palette.type === "light" ? theme.palette.primary.dark : theme.palette.primary.light
      }`,
    },
    listSubHeader: {
      textTransform: "uppercase",
    },
    listItemText: {
      "font-size": theme.typography.subtitle1.fontSize,
    },
  });

interface Props
  extends WithStyles<typeof styles>,
    WithUserAuthProps,
    ReturnType<typeof mapStateToProps>,
    RouteComponentProps<{ applicationName: string }> {
  dispatch: TDispatch;
  // canEdit?: boolean;
}

interface State {}

class ApplicationViewDrawerRaw extends React.PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {};
  }

  private getMenuData() {
    const { activeNamespaceName, canManageNamespace, canEditCluster } = this.props;
    const menus = [];
    menus.push({
      text: "Components",
      to: "/applications/" + activeNamespaceName + "/components",
      icon: <KalmComponentsIcon />,
    });
    if (canManageNamespace(activeNamespaceName) || canEditCluster()) {
      menus.push({
        text: sc.APP_MEMBERS_PAGE_NAME,
        to: "/applications/" + activeNamespaceName + "/members",
        icon: <PeopleIcon />,
      });
    }
    menus.push({
      text: sc.APP_DASHBOARD_PAGE_NAME,
      to: "/applications/" + activeNamespaceName + "/metrics",
      highlightWhenExact: true,
      icon: <DashboardIcon />,
    });
    return menus;
  }

  render() {
    const {
      classes,
      location: { pathname },
    } = this.props;
    const menuData = this.getMenuData();

    return (
      <List style={{ width: "100%" }}>
        {menuData.map((item, index) => (
          <ListItem
            onClick={() => blinkTopProgressAction()}
            className={classes.listItem}
            classes={{
              selected: classes.listItemSelected,
            }}
            button
            component={NavLink}
            to={item.to}
            key={item.text}
            selected={item.highlightWhenExact ? pathname === item.to : pathname.startsWith(item.to.split("?")[0])}
          >
            <ListItemIcon>{item.icon ? item.icon : <AppsIcon />}</ListItemIcon>
            <ListItemText classes={{ primary: classes.listItemText }} primary={item.text} />
          </ListItem>
        ))}
      </List>
    );
  }
}

export const ApplicationSidebar = withUserAuth(
  withRouter(connect(mapStateToProps)(withStyles(styles)(ApplicationViewDrawerRaw))),
);
