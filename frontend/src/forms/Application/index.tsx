import { Button, createStyles, Grid, WithStyles, withStyles } from "@material-ui/core";
import { Theme } from "@material-ui/core/styles";
import Immutable from "immutable";
import React from "react";
import { connect, DispatchProp } from "react-redux";
import { InjectedFormProps } from "redux-form";
import { Field, formValueSelector, getFormValues, reduxForm } from "redux-form/immutable";
import { RootState } from "../../reducers";
import { Application, SharedEnv } from "../../types/application";
import { ComponentTemplate } from "../../types/componentTemplate";
import { H3 } from "../../widgets/Label";
import { TextField } from "../Basic/text";
import { ValidatorName, ValidatorRequired } from "../validator";
import { SharedEnvs } from "./SharedEnvs";
import { Plugins } from "./Plugins";

const styles = (theme: Theme) =>
  createStyles({
    root: {
      // height: "100%",
      // overflow: "hidden",
      background: "#fff",
      padding: 20
    },
    formSection: {
      padding: theme.spacing(2),
      margin: theme.spacing(3)
    },
    formSectionTable: {
      padding: theme.spacing(0),
      margin: theme.spacing(3)
    },
    formSectionContainer: {
      margin: "0",
      width: "auto"
    },
    formSectionItem: {
      padding: "0px !important"
    },
    displayNone: {
      display: "none"
    },
    buttons: {
      margin: "20px 0 0"
    },
    submitButton: {
      marginRight: 20
    }
  });

const mapStateToProps = (state: RootState) => {
  const selector = formValueSelector("application");
  const formComponents: ComponentTemplate[] = selector(state, "components");
  const sharedEnvs: Immutable.List<SharedEnv> = selector(state, "sharedEnvs");
  const values = getFormValues("application")(state) as Application;

  return {
    sharedEnvs,
    formComponents,
    values
  };
};

export interface Props {
  isEdit?: boolean;
  currentTab: "basic" | "sharedEnvs" | "applicationPlugins";
}

class ApplicationFormRaw extends React.PureComponent<
  Props &
    InjectedFormProps<Application, Props> &
    ReturnType<typeof mapStateToProps> &
    WithStyles<typeof styles> &
    DispatchProp
> {
  private renderBasic() {
    const { isEdit } = this.props;
    return (
      <>
        <Grid container spacing={2}>
          <Grid item md={12}>
            <Field
              name="name"
              label="Name"
              disabled={isEdit}
              component={TextField}
              validate={[ValidatorRequired, ValidatorName]}
              helperText={
                isEdit
                  ? "Can't modify name"
                  : 'The characters allowed in names are: digits (0-9), lower case letters (a-z), "-", and ".". Max length is 180.'
              }
              placeholder="Please type the component name"
            />
          </Grid>
        </Grid>
        {/* <Grid container spacing={2}>
            <Grid item md={12}>
              <BoldBody>Status</BoldBody>

              <Field
                name="isActive"
                formControlLabelProps={{
                  label: "Active this application after creation"
                }}
                component={CheckboxField}
                normalizer={NormalizeBoolean}
              />
            </Grid>
          </Grid> */}
      </>
    );
  }

  // private renderComponent() {
  //   return (
  //     <>
  //       <Components />
  //     </>
  //   );
  // }

  private renderSharedEnvs() {
    return (
      <>
        <SharedEnvs />
      </>
    );
  }

  private renderPlugins() {
    return (
      <>
        <Plugins />
      </>
    );
  }

  public render() {
    const { handleSubmit, change, classes, currentTab } = this.props;

    return (
      <form onSubmit={handleSubmit} className={classes.root}>
        <Grid
          container
          spacing={2}
          className={`${classes.formSectionContainer} ${currentTab === "basic" ? "" : classes.displayNone}`}>
          <Grid className={classes.formSectionItem} item xs={12} sm={6} md={8}>
            <H3>Application Basic</H3>
            {this.renderBasic()}
          </Grid>
        </Grid>

        <div className={`${classes.formSectionContainer} ${currentTab === "sharedEnvs" ? "" : classes.displayNone}`}>
          <H3>Shared Environments</H3>
          {this.renderSharedEnvs()}
        </div>

        <div
          className={`${classes.formSectionContainer} ${
            currentTab === "applicationPlugins" ? "" : classes.displayNone
          }`}>
          <H3>Application Plugins</H3>
          {this.renderPlugins()}
        </div>

        <div className={`${currentTab === "basic" ? classes.buttons : classes.displayNone}`}>
          <Button
            variant="contained"
            color="primary"
            className={`${currentTab === "basic" ? classes.submitButton : classes.displayNone}`}
            onClick={event => {
              change("isActive", true);
              setTimeout(() => {
                handleSubmit(event);
              }, 300);
            }}>
            Publish
          </Button>
          <Button
            variant="contained"
            className={`${currentTab === "basic" ? "" : classes.displayNone}`}
            onClick={event => {
              change("isActive", false);
              setTimeout(() => {
                handleSubmit(event);
              }, 300);
            }}>
            Save (publish later)
          </Button>
        </div>
      </form>
    );
  }
}

export const applicationInitialValues: Application = Immutable.fromJS({
  name: "",
  sharedEnvs: [],
  components: []
});

export default reduxForm<Application, Props>({
  form: "application",
  initialValues: applicationInitialValues,
  onSubmitFail: (...args) => {
    console.log("submit failed", args);
  }
})(connect(mapStateToProps)(withStyles(styles)(ApplicationFormRaw)));
