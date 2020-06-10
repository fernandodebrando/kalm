import { Box, Button, Grid, Icon, MenuItem } from "@material-ui/core";
import Immutable from "immutable";
import React from "react";
import { connect, DispatchProp } from "react-redux";
import { arrayPush, WrappedFieldArrayProps } from "redux-form";
import { Field, FieldArray } from "redux-form/immutable";
import { DeleteIcon } from "widgets/Icon";
import { IconButtonWithTooltip } from "widgets/IconButtonWithTooltip";
import { portTypeTCP, portTypeUDP } from "../../types/common";
import { ComponentLikePort } from "../../types/componentTemplate";
import { RenderSelectField } from "../Basic/select";
import { KRenderTextField } from "../Basic/textfield";
import { NormalizePort } from "../normalizer";
import { ValidatorRequired } from "../validator";
import { CodeGenerator } from "@babel/generator";
import { Alert } from "@material-ui/lab";
interface FieldArrayComponentHackType {
  name: any;
  component: any;
  validate: any;
}

interface FieldArrayProps extends DispatchProp {}

interface Props extends WrappedFieldArrayProps<ComponentLikePort>, FieldArrayComponentHackType, FieldArrayProps {}

const ValidatorPorts = (values: Immutable.List<ComponentLikePort>, _allValues?: any, _props?: any, _name?: any) => {
  if (!values) return undefined;
  const names = new Set<string>();
  const protocolServicePorts = new Set<string>();
  for (let i = 0; i < values.size; i++) {
    const port = values.get(i)!;
    const name = port.get("name");

    if (!names.has(name)) {
      names.add(name);
    } else {
      return "Port names should be unique.  " + name + "";
    }

    if (!port.get("servicePort")) {
      continue;
    }

    const protocol = port.get("protocol");
    const servicePort = port.get("servicePort");
    const protocolServicePort = protocol + "-" + servicePort;

    if (!protocolServicePorts.has(protocolServicePort)) {
      protocolServicePorts.add(protocolServicePort);
    } else {
      return "Listening port on a protocol should be unique.  " + protocol + " - " + servicePort;
    }
  }
};

class RenderPorts extends React.PureComponent<Props> {
  public render() {
    const {
      fields,
      dispatch,
      meta: { submitFailed, error, form }
    } = this.props;
    return (
      <>
        <Box mb={2}>
          <Grid item xs>
            <Button
              variant="outlined"
              color="primary"
              startIcon={<Icon>add</Icon>}
              size="small"
              onClick={() =>
                dispatch(
                  arrayPush(
                    form,
                    fields.name,
                    Immutable.Map({
                      name: "",
                      protocol: portTypeTCP,
                      containerPort: ""
                    })
                  )
                )
              }>
              Add
            </Button>

            {/* {submitFailed && error && <span>{error}</span>} */}
            {error ? (
              <Box mt={2}>
                <Alert severity="error">{error}</Alert>
              </Box>
            ) : null}
          </Grid>
        </Box>

        {fields.map((field, index) => {
          return (
            <Grid container spacing={2} key={field}>
              <Grid item xs>
                <Field
                  component={KRenderTextField}
                  name={`${field}.name`}
                  label="Name"
                  validate={[ValidatorRequired]}
                  required
                />
              </Grid>
              <Grid item xs>
                <Field
                  name={`${field}.protocol`}
                  component={RenderSelectField}
                  label="Protocol"
                  validate={[ValidatorRequired]}
                  options={[
                    { value: portTypeTCP, text: portTypeTCP },
                    { value: portTypeUDP, text: portTypeUDP }
                  ]}>
                  <MenuItem value={portTypeUDP}>{portTypeUDP}</MenuItem>
                  <MenuItem value={portTypeTCP}>{portTypeTCP}</MenuItem>
                </Field>
              </Grid>
              <Grid item xs>
                <Field
                  component={KRenderTextField}
                  name={`${field}.containerPort`}
                  label="Publish port"
                  required
                  validate={[ValidatorRequired]}
                  normalize={NormalizePort}
                />
              </Grid>
              <Grid item xs>
                <Field
                  component={KRenderTextField}
                  name={`${field}.servicePort`}
                  required
                  label="On listening port"
                  validate={[ValidatorRequired]}
                  normalize={NormalizePort}
                />
              </Grid>
              <Grid item xs>
                <IconButtonWithTooltip
                  tooltipPlacement="top"
                  tooltipTitle="Delete"
                  aria-label="delete"
                  onClick={() => fields.remove(index)}>
                  <DeleteIcon />
                </IconButtonWithTooltip>
              </Grid>
            </Grid>
          );
        })}
      </>
    );
  }
}

export const Ports = connect()((props: FieldArrayProps) => {
  return <FieldArray name="ports" component={RenderPorts} validate={ValidatorPorts} {...props} />;
});
