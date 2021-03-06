import { Box, Button, Collapse, Grid } from "@material-ui/core";
import Typography from "@material-ui/core/Typography";
import Warning from "@material-ui/icons/Warning";
import { Alert, AlertTitle } from "@material-ui/lab";
import { Field, FieldArray, getIn } from "formik";
import { KAutoCompleteOption, KFormikAutoCompleteSingleValue } from "forms/Basic/autoComplete";
import { KFormikRenderSlider } from "forms/Basic/slider";
import React from "react";
import { connect } from "react-redux";
import { RootState } from "reducers";
import {
  PortProtocolGRPC,
  PortProtocolGRPCWEB,
  PortProtocolHTTP,
  PortProtocolHTTP2,
  PortProtocolHTTPS,
} from "types/componentTemplate";
import { HttpRouteDestination } from "types/route";
import { AddIcon, DeleteIcon } from "widgets/Icon";
import { IconButtonWithTooltip } from "widgets/IconButtonWithTooltip";
import { ValidatorRequired } from "../validator";

interface FieldArrayComponentHackType {
  destinations: HttpRouteDestination[];
  errors: any;
  touched: any;
}

const mapStateToProps = (state: RootState) => {
  return {
    activeNamespace: state.namespaces.active,
    services: state.services.services,
  };
};

interface Props extends FieldArrayComponentHackType, ReturnType<typeof mapStateToProps> {}

class RenderHttpRouteDestinationsRaw extends React.PureComponent<Props> {
  private renderRows() {
    const { services, activeNamespace, destinations } = this.props;

    const options: KAutoCompleteOption[] = [];
    services
      .filter((x) => {
        const ns = x.namespace;

        // TODO should we ignore the system namespaces??
        return (
          ns !== "default" &&
          ns !== "kalm-operator" &&
          ns !== "kalm-imgconv" &&
          ns !== "kube-system" &&
          ns !== "istio-system" &&
          ns !== "cert-manager" &&
          ns !== "istio-operator"
        );
      })
      .forEach((svc) => {
        svc.ports
          .filter((p) => {
            const appProtocol = p.appProtocol;
            return (
              appProtocol === PortProtocolHTTP ||
              appProtocol === PortProtocolHTTP2 ||
              appProtocol === PortProtocolHTTPS ||
              appProtocol === PortProtocolGRPC ||
              appProtocol === PortProtocolGRPCWEB
            );
          })
          .forEach((port) => {
            options.push({
              value: `${svc.name}.${svc.namespace}.svc.cluster.local:${port.port}`,
              label: svc.name + ":" + port.port + `(${port.appProtocol})`,
              group: svc.namespace === activeNamespace ? `${svc.namespace} (Current)` : svc.namespace,
            });
          });
      });

    return (
      <FieldArray
        name="destinations"
        render={(arrayHelpers) => (
          <div>
            <Box mt={2} mr={2} mb={2}>
              <Button
                variant="outlined"
                color="primary"
                startIcon={<AddIcon />}
                size="small"
                id="add-target-button"
                onClick={() =>
                  arrayHelpers.push({
                    host: "",
                    weight: 1,
                  })
                }
              >
                Add a target
              </Button>
            </Box>
            <Collapse in={destinations.length > 1}>
              <Alert className="alert" severity="info">
                There are more than one target, traffic will be forwarded to each target by weight.
              </Alert>
            </Collapse>
            {destinations &&
              destinations.map((destination, index) => (
                <Grid container spacing={2} key={index} alignItems="center">
                  <Grid item xs={8} sm={8} md={6} lg={4} xl={4}>
                    <Field
                      name={`destinations.${index}.host`}
                      component={KFormikAutoCompleteSingleValue}
                      label="Choose a target"
                      validate={ValidatorRequired}
                      options={options}
                      noOptionsText={
                        <Alert severity="warning">
                          <AlertTitle>No valid targets found.</AlertTitle>
                          <Typography>
                            If you can't find the target you want, please check if you have configured ports on the
                            component. Only components that have ports will appear in the options.
                          </Typography>
                        </Alert>
                      }
                    />
                  </Grid>
                  {destinations.length > 1 ? (
                    <Grid item md={2}>
                      <Field
                        name={`destinations.${index}.weight`}
                        component={KFormikRenderSlider}
                        label="Weight"
                        step={1}
                        min={0}
                        max={10}
                        disabled={destinations.length <= 1}
                      />
                    </Grid>
                  ) : null}
                  <Grid item md={1}>
                    <IconButtonWithTooltip
                      tooltipPlacement="top"
                      tooltipTitle="Delete"
                      aria-label="delete"
                      onClick={() => arrayHelpers.remove(index)}
                    >
                      <DeleteIcon />
                    </IconButtonWithTooltip>
                  </Grid>
                  {destination.weight === 0 ? (
                    <Grid item md={3}>
                      <Warning /> Requests won't go into this target since it has 0 weight.
                    </Grid>
                  ) : null}
                </Grid>
              ))}{" "}
          </div>
        )}
      />
    );
  }

  public render() {
    const { destinations, touched } = this.props;
    return (
      <div>
        {getIn(touched, "destinations") && destinations && destinations.length === 0 ? (
          <Box>
            <Alert severity="error">{"You should at least configure one Targets."}</Alert>
          </Box>
        ) : null}
        {this.renderRows()}
      </div>
    );
  }
}

export const RenderHttpRouteDestinations = connect(mapStateToProps)(RenderHttpRouteDestinationsRaw);
