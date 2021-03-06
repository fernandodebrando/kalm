import {
  Box,
  createStyles,
  Grid,
  MenuItem,
  StandardTextFieldProps,
  TextField,
  Theme,
  Typography,
  withStyles,
} from "@material-ui/core";
import { WithStyles } from "@material-ui/styles";
import { FastField, FieldProps, getIn } from "formik";
import { FormikNormalizePositiveNumber } from "forms/normalizer";
import React from "react";
import { connect, DispatchProp } from "react-redux";
import { PortProtocolHTTP, PortProtocolTCP, Probe, ComponentLikePort } from "types/componentTemplate";
import sc from "../../utils/stringConstants";
import { makeSelectOption, SelectField } from "../Basic/select";
import { ValidatorOneof, ValidatorRequired, ValidateHost } from "../validator";

interface FieldComponentHackType {
  component: any;
}

const ValidatorScheme = ValidatorOneof(/^https?$/i);

interface ProbeProps extends DispatchProp {}

interface Props extends FieldProps, FieldComponentHackType, ProbeProps, WithStyles<typeof styles> {}

const styles = (theme: Theme) =>
  createStyles({
    input: {
      padding: 2,
      textAlign: "center",
      width: 60,
      "&::placeholder": {
        textAlign: "center",
      },
    },
    code: {
      fontFamily: "Hack, monospace",
      "& input": {
        fontFamily: "Hack, monospace",
      },
    },
  });

class RenderProbe extends React.PureComponent<Props> {
  private renderNestedTextfield = ({
    field: { name, value },
    form: { setFieldValue, values, errors, touched },
    placeholder,
    style,
    select,
    children,
    normalize,
    type,
  }: FieldProps & StandardTextFieldProps & { style?: any; type?: string; normalize?: any }) => {
    const { classes } = this.props;
    return (
      <TextField
        error={!!getIn(errors, name)}
        helperText={getIn(errors, name)}
        InputProps={{ classes: { input: classes.input } }}
        onChange={(e) => {
          if (normalize) {
            setFieldValue(name, normalize(e));
          } else {
            setFieldValue(name, e.target.value);
          }
        }}
        value={value}
        size="small"
        type={type}
        select={select}
        placeholder={placeholder}
        inputProps={{ style }}
      >
        {children}
      </TextField>
    );
  };

  private renderHttpGet() {
    const name = this.props.field.name;
    const { classes } = this.props;
    return (
      <Box p={1}>
        <Typography component="div">
          After initial
          <FastField
            name={`${name}.initialDelaySeconds`}
            component={this.renderNestedTextfield}
            normalize={FormikNormalizePositiveNumber}
            placeholder="10"
            type="number"
            min="1"
            style={{ width: 60 }}
          />
          seconds delay, Request{" "}
          <Box className={classes.code} display="inline-block">
            <FastField
              name={`${name}.httpGet.scheme`}
              component={this.renderNestedTextfield}
              validate={ValidatorScheme}
              placeholder="http"
              select
              style={{ width: 60 }}
            >
              <MenuItem key={"http"} value={"HTTP"}>
                http
              </MenuItem>
              <MenuItem key={"http"} value={"HTTPS"}>
                https
              </MenuItem>
            </FastField>
            ://
            <FastField
              name={`${name}.httpGet.host`}
              component={this.renderNestedTextfield}
              placeholder="0.0.0.0"
              validate={ValidateHost}
              style={{ width: 80 }}
            />
            :
            <FastField
              name={`${name}.httpGet.port`}
              component={this.renderNestedTextfield}
              placeholder="8080"
              normalize={FormikNormalizePositiveNumber}
              validate={ValidatorRequired}
              style={{ width: 60 }}
            />
            <FastField
              name={`${name}.httpGet.path`}
              component={this.renderNestedTextfield}
              placeholder="/healthy"
              validate={ValidatorRequired}
              style={{ width: 80, textAlign: "left" }}
            />
          </Box>{" "}
          will be triggered every{" "}
          <FastField
            name={`${name}.periodSeconds`}
            component={this.renderNestedTextfield}
            normalize={FormikNormalizePositiveNumber}
            placeholder="10"
            type="number"
            min="1"
            style={{ width: 60 }}
          />{" "}
          seconds.
        </Typography>
      </Box>
    );
  }

  private renderExec() {
    const name = this.props.field.name;
    const { classes } = this.props;
    return (
      <Box p={1}>
        <Typography component="div">
          After initial
          <FastField
            name={`${name}.initialDelaySeconds`}
            component={this.renderNestedTextfield}
            placeholder="10"
            normalize={FormikNormalizePositiveNumber}
            type="number"
            min="1"
            style={{ width: 60 }}
          />
          seconds delay, Command{" "}
          <Box className={classes.code} display="inline-block">
            <FastField
              name={`${name}.exec.command[0]`}
              component={this.renderNestedTextfield}
              validate={ValidatorRequired}
              placeholder="command"
              style={{ width: 300 }}
            />
          </Box>{" "}
          will be executed every{" "}
          <FastField
            name={`${name}.periodSeconds`}
            component={this.renderNestedTextfield}
            normalize={FormikNormalizePositiveNumber}
            placeholder="10"
            type="number"
            min="1"
            style={{ width: 60 }}
          />{" "}
          seconds.
        </Typography>
      </Box>
    );
  }

  private renderTcpSocket() {
    const name = this.props.field.name;
    const { classes } = this.props;
    return (
      <Box p={1}>
        <Typography component="div">
          After initial
          <FastField
            name={`${name}.initialDelaySeconds`}
            component={this.renderNestedTextfield}
            normalize={FormikNormalizePositiveNumber}
            placeholder="10"
            type="number"
            min="1"
            style={{ width: 60 }}
          />
          seconds delay, TCP socket connection to{" "}
          <Box className={classes.code} display="inline-block">
            <FastField
              name={`${name}.tcpSocket.host`}
              component={this.renderNestedTextfield}
              placeholder="0.0.0.0"
              style={{ width: 200 }}
            />
            :
            <FastField
              name={`${name}.tcpSocket.port`}
              component={this.renderNestedTextfield}
              validate={ValidatorRequired}
              normalize={FormikNormalizePositiveNumber}
              placeholder="8080"
              style={{ width: 60 }}
            />
          </Box>{" "}
          will be established every{" "}
          <FastField
            name={`${name}.periodSeconds`}
            component={this.renderNestedTextfield}
            normalize={FormikNormalizePositiveNumber}
            placeholder="10"
            type="number"
            min="1"
            style={{ width: 60 }}
          />{" "}
          seconds.
        </Typography>
      </Box>
    );
  }

  private renderCommon() {
    const name = this.props.field.name;
    const type = this.getProbeType();

    return (
      <Box p={1}>
        <Typography component="div">
          If there is no response within{" "}
          <FastField
            name={`${name}.timeoutSeconds`}
            component={this.renderNestedTextfield}
            placeholder="1"
            normalize={FormikNormalizePositiveNumber}
            style={{ width: 60 }}
            type="number"
            min="1"
          />{" "}
          seconds or{" "}
          {type === "httpGet"
            ? "an error response (http status code >= 400) is returned"
            : type === "exec"
            ? "the command exits with a non-zero code"
            : "the TCP connection is failed"}
          , the current round of testing is considered to have failed. Otherwise, the it is considered successful.
        </Typography>
        <br />
        <Typography component="div">
          {name === "livenessProbe" ? (
            "One successful testing"
          ) : (
            <>
              <FastField
                name={`${name}.successThreshold`}
                component={this.renderNestedTextfield}
                placeholder="1"
                normalize={FormikNormalizePositiveNumber}
                style={{ width: 60 }}
                type="number"
                min="1"
              />
              consecutive successful tesings
            </>
          )}{" "}
          will make the probe ready.{" "}
          <FastField
            name={`${name}.failureThreshold`}
            component={this.renderNestedTextfield}
            placeholder="3"
            normalize={FormikNormalizePositiveNumber}
            type="number"
            min="1"
            style={{ width: 60 }}
          />{" "}
          consecutive failed tesings will make the probe faild.
        </Typography>
      </Box>
    );
  }

  private getProbeObject = () => {
    const {
      field: { value },
    } = this.props;

    let probe: Probe | undefined = value;

    return probe;
  };

  private handleChangeType(type: string) {
    const {
      field: { name },
      form: { setFieldValue, values },
    } = this.props;

    const ports: ComponentLikePort[] | undefined = values.ports;

    if (type === "httpGet") {
      const potentialPort = ports ? ports.find((x) => x.protocol === PortProtocolHTTP && !!x.containerPort) : null;
      setFieldValue(name, {
        httpGet: {
          scheme: "HTTP",
          host: "0.0.0.0",
          path: "/health",
          port: potentialPort ? potentialPort.containerPort : 8080,
        },
        failureThreshold: 3,
        periodSeconds: 10,
        successThreshold: 1,
        timeoutSeconds: 1,
        initialDelaySeconds: 10,
      });
    } else if (type === "exec") {
      setFieldValue(name, {
        exec: {
          command: [""],
        },
        failureThreshold: 3,
        periodSeconds: 10,
        successThreshold: 1,
        timeoutSeconds: 1,
        initialDelaySeconds: 10,
      });
    } else if (type === "tcpSocket") {
      const potentialPort = ports ? ports.find((x: any) => x.protocol === PortProtocolTCP && !!x.containerPort) : null;
      setFieldValue(name, {
        tcpSocket: {
          port: potentialPort ? potentialPort.containerPort : 8080,
          host: "0.0.0.0",
        },
        failureThreshold: 3,
        periodSeconds: 10,
        successThreshold: 1,
        timeoutSeconds: 1,
        initialDelaySeconds: 10,
      });
    } else {
      setFieldValue(name, null);
    }
  }

  private getProbeType = () => {
    const probe = this.getProbeObject();

    return !probe
      ? "none"
      : !!probe.httpGet
      ? "httpGet"
      : !!probe.exec
      ? "exec"
      : !!probe.tcpSocket
      ? "tcpSocket"
      : "none";
  };

  public render() {
    const type = this.getProbeType();
    return (
      <Grid container spacing={2}>
        <Grid item xs={6}>
          <SelectField
            label="Type"
            value={type}
            onChange={(event: any) => {
              this.handleChangeType(event.target.value);
            }}
            meta={{
              touched: true,
              error: undefined,
            }}
            options={[
              makeSelectOption("none", "None", sc.PROBE_NONE_OPTION),
              makeSelectOption("httpGet", "HTTP", sc.PROBE_HTTP_OPTION),
              makeSelectOption("exec", "Command", sc.PROBE_COMMAND_OPTION),
              makeSelectOption("tcpSocket", "TCP", sc.PROBE_TCP_OPTION),
            ]}
          />
        </Grid>

        {type === "httpGet" && this.renderHttpGet()}
        {type === "exec" && this.renderExec()}
        {type === "tcpSocket" && this.renderTcpSocket()}
        {type !== "none" && this.renderCommon()}
      </Grid>
    );
  }
}

export const LivenessProbe = connect()((props: ProbeProps) => {
  return <FastField name="livenessProbe" component={withStyles(styles)(RenderProbe)} {...props} />;
});

export const ReadinessProbe = connect()((props: ProbeProps) => {
  return <FastField name="readinessProbe" component={withStyles(styles)(RenderProbe)} {...props} />;
});
