import React from "react";
import { createStyles, Theme, withStyles, WithStyles } from "@material-ui/core";
import { TDispatchProp } from "types";
import { connect } from "react-redux";
import { RootState } from "reducers";
import { VerticalHeadTable } from "widgets/VerticalHeadTable";
import { ApplicationComponentDetails } from "types/application";
import { getComponentCreatedAtString } from "utils/application";
import { SmallCPULineChart, SmallMemoryLineChart } from "widgets/SmallLineChart";

const styles = (theme: Theme) =>
  createStyles({
    root: {},
  });

const mapStateToProps = (state: RootState) => {
  return {
    // xxx: state.get("xxx").get("xxx"),
  };
};

interface Props extends WithStyles<typeof styles>, ReturnType<typeof mapStateToProps>, TDispatchProp {
  activeNamespaceName: string;
  component: ApplicationComponentDetails;
}

interface State {}

class ComponentBasicInfoRaw extends React.PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {};
  }

  private renderCreatedAt = () => {
    const { component } = this.props;
    return getComponentCreatedAtString(component);
  };

  private renderComponentStatus = () => {
    const { component } = this.props;

    let running = 0;
    let pending = 0;
    let error = 0;

    component.get("pods").forEach((pod) => {
      if (pod.get("isTerminating")) {
        pending = pending + 1;
      } else {
        switch (pod.get("status")) {
          case "Pending": {
            pending = pending + 1;
            break;
          }
          case "Failed": {
            error = error + 1;
            break;
          }
          case "Running":
          case "Succeeded": {
            running = running + 1;
            break;
          }
        }
      }
    });

    return `Running: ${running}, Pending: ${pending}, Error: ${error}`;
  };

  private renderComponentCPU = () => {
    const { component } = this.props;
    return <SmallCPULineChart data={component.get("metrics").get("cpu")!} />;
  };

  private renderComponentMemory = () => {
    const { component } = this.props;
    return <SmallMemoryLineChart data={component.get("metrics").get("memory")!} />;
  };

  public render() {
    const { component, activeNamespaceName } = this.props;
    return (
      <VerticalHeadTable
        items={[
          { name: "Created At", content: this.renderCreatedAt() },
          { name: "Name", content: component.get("name") },
          { name: "Namespace", content: activeNamespaceName },
          { name: "Image", content: component.get("image") },
          { name: "Workload Type", content: component.get("workloadType") },
          { name: "Update Strategy", content: component.get("restartStrategy") },
          { name: "Pod Status", content: this.renderComponentStatus() },
          { name: "CPU", content: this.renderComponentCPU() },
          { name: "Memory", content: this.renderComponentMemory() },
        ]}
      />
    );
  }
}

export const ComponentBasicInfo = withStyles(styles)(connect(mapStateToProps)(ComponentBasicInfoRaw));