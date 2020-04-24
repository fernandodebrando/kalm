import {
  Button,
  ClickAwayListener,
  createStyles,
  Grow,
  MenuItem,
  MenuList,
  Paper,
  Popper,
  Theme,
  withStyles,
  WithStyles
} from "@material-ui/core";
import React from "react";
import { connect } from "react-redux";
import { TDispatchProp } from "types";
import {
  // loadNamespacesAction,
  setCurrentNamespaceAction
} from "actions/namespaces";
import { RootState } from "reducers";

const styles = (theme: Theme) =>
  createStyles({
    root: {
      zIndex: 10
    },
    namespaceButton: {
      color: "#C7E1F5",
      marginLeft: theme.spacing(2),
      background: "#42A5F5",
      border: "0"
    }
  });

const mapStateToProps = (state: RootState) => {
  const applicationsRoot = state.get("applications");
  return {
    applications: applicationsRoot.get("applications"),
    isListFirstLoading: applicationsRoot.get("isListLoading") && !applicationsRoot.get("isListFirstLoaded"),
    active: state.get("namespaces").get("active")
  };
};

interface Props extends WithStyles<typeof styles>, ReturnType<typeof mapStateToProps>, TDispatchProp {}

interface State {
  open: boolean;
}

class NamespacesRaw extends React.PureComponent<Props, State> {
  private anchorRef = React.createRef<HTMLButtonElement>();

  constructor(props: Props) {
    super(props);
    this.state = {
      open: false
    };
  }

  private handleToggle = () => {
    this.setState({
      open: !this.state.open
    });
  };

  private handleClose = () => {
    this.setState({
      open: false
    });
  };

  private handleListKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === "Tab") {
      event.preventDefault();
      this.handleClose();
    }
  };

  componentDidMount() {
    // this.props.dispatch(loadNamespacesAction());
  }

  public render() {
    const { classes, applications, active, dispatch, isListFirstLoading } = this.props;
    const { open } = this.state;
    const activeNamespace = applications.find(application => application.get("name") === active);
    return (
      <div className={classes.root}>
        <Button
          ref={this.anchorRef}
          aria-controls={open ? "menu-list-grow" : undefined}
          aria-haspopup="true"
          variant="outlined"
          size="medium"
          className={classes.namespaceButton}
          onClick={this.handleToggle}>
          {isListFirstLoading ? "Loading..." : activeNamespace ? activeNamespace.get("name") : "Select a namespace"}
        </Button>
        <Popper open={open} anchorEl={this.anchorRef.current} role={undefined} transition disablePortal>
          {({ TransitionProps, placement }) => (
            <Grow
              {...TransitionProps}
              style={{ transformOrigin: placement === "bottom" ? "center top" : "center bottom" }}>
              <Paper>
                <ClickAwayListener onClickAway={this.handleClose}>
                  <MenuList autoFocusItem={open} id="menu-list-grow" onKeyDown={this.handleListKeyDown}>
                    {applications
                      .map(application => (
                        <MenuItem
                          onClick={() => {
                            dispatch(setCurrentNamespaceAction(application.get("name")));
                            this.handleClose();
                          }}
                          key={application.get("name")}>
                          {application.get("name")}
                        </MenuItem>
                      ))
                      .toArray()}
                  </MenuList>
                </ClickAwayListener>
              </Paper>
            </Grow>
          )}
        </Popper>
      </div>
    );
  }
}

export const Namespaces = withStyles(styles)(connect(mapStateToProps)(NamespacesRaw));
