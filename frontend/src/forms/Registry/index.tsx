import { Box, createStyles, Grid, WithStyles, withStyles } from "@material-ui/core";
import { Theme } from "@material-ui/core/styles";
import { FormikProps, withFormik, Field, Form } from "formik";
import React from "react";
import { connect, DispatchProp } from "react-redux";
import { RootState } from "reducers";
import { RegistryFormType } from "types/registry";
import sc from "utils/stringConstants";
import { CustomizedButton } from "widgets/Button";
import { KPanel } from "widgets/KPanel";
import { Prompt } from "widgets/Prompt";
import { KRenderThrottleFormikTextField } from "../Basic/textfield";
import { RequireNoSuffix, RequirePrefix, ValidatorName, ValidatorRequired } from "../validator";

const styles = (theme: Theme) =>
  createStyles({
    root: {
      padding: theme.spacing(2),
    },
  });

const mapStateToProps = (state: RootState) => {
  return {
    isSubmittingRegistry: state.registries.isSubmittingRegistry,
  };
};

export interface Props {
  isEdit?: boolean;
  onSubmit: any;
  initial: RegistryFormType;
}

const validateHost = (value: any, _allValues?: any, _props?: any, _name?: any) => {
  if (!value) return undefined;

  return RequirePrefix("https://")(value) || RequireNoSuffix("/")(value);
};

class RegistryFormRaw extends React.PureComponent<
  Props & FormikProps<RegistryFormType> & ReturnType<typeof mapStateToProps> & WithStyles<typeof styles> & DispatchProp
> {
  public render() {
    const { classes, isEdit, dirty, values, isSubmittingRegistry, isSubmitting } = this.props;

    return (
      <Form className={classes.root} id="registry-form">
        <Prompt when={dirty && !isSubmitting} message={sc.CONFIRM_LEAVE_WITHOUT_SAVING} />
        <KPanel
          content={
            <Box p={2}>
              <Grid container spacing={2}>
                <Grid item md={12}>
                  <Field
                    name="name"
                    label="Name"
                    disabled={isEdit}
                    component={KRenderThrottleFormikTextField}
                    validate={ValidatorName}
                    helperText={isEdit ? "Can't modify name" : sc.NAME_RULE}
                  />
                </Grid>
                <Grid item md={12}>
                  <Field
                    name="username"
                    label="Username"
                    autoComplete="off"
                    component={KRenderThrottleFormikTextField}
                    validate={ValidatorRequired}
                  />
                </Grid>
                <Grid item md={12}>
                  <Field
                    type="password"
                    name="password"
                    label="Password"
                    autoComplete="off"
                    component={KRenderThrottleFormikTextField}
                    validate={ValidatorRequired}
                  />
                </Grid>
                <Grid item md={12}>
                  <Field
                    name="host"
                    label="Host"
                    component={KRenderThrottleFormikTextField}
                    validate={validateHost}
                    placeholder="E.g. https://registry.kalm.dev"
                    helperText={<span>Leave blank for private docker hub registry</span>}
                  />
                </Grid>
              </Grid>

              {process.env.REACT_APP_DEBUG === "true" ? (
                <pre style={{ maxWidth: 1500, background: "#eee" }}>{JSON.stringify(values, undefined, 2)}</pre>
              ) : null}
            </Box>
          }
        />
        <Box pt={2}>
          <CustomizedButton disabled={isSubmittingRegistry} type="submit" color="primary" variant="contained">
            Save
          </CustomizedButton>
        </Box>
      </Form>
    );
  }
}

const connectedForm = connect(mapStateToProps)(withStyles(styles)(RegistryFormRaw));

export const RegistryForm = withFormik<Props, RegistryFormType>({
  mapPropsToValues: (props) => {
    return props.initial;
  },
  validate: (values: RegistryFormType) => {
    let errors = {};
    return errors;
  },
  handleSubmit: async (formValues, { props: { onSubmit } }) => {
    await onSubmit(formValues);
  },
})(connectedForm);
