import { StatusFailure, ThunkResult } from "../types";
import {
  Application,
  ApplicationDetails,
  ApplicationDetailsList,
  CREATE_APPLICATION,
  DELETE_APPLICATION,
  DUPLICATE_APPLICATION,
  LOAD_APPLICATIONS_FAILED,
  LOAD_APPLICATIONS_FULFILLED,
  LOAD_APPLICATIONS_PENDING,
  LOAD_APPLICATION_FAILED,
  LOAD_APPLICATION_FULFILLED,
  LOAD_APPLICATION_PENDING,
  SetIsSubmittingApplication,
  SetIsSubmittingApplicationComponent,
  SET_IS_SUBMITTING_APPLICATION,
  SET_IS_SUBMITTING_APPLICATION_COMPONENT,
  UPDATE_APPLICATION,
  ApplicationComponent,
  ApplicationComponentDetails,
  CREATE_COMPONENT,
  UPDATE_COMPONENT,
  DELETE_COMPONENT,
  LOAD_APPLICATION_PLUGINS_FULFILLED,
  LOAD_COMPONENT_PLUGINS_FULFILLED
} from "../types/application";
import {
  createKappApplication,
  deleteKappApplication,
  getKappApplication,
  getKappApplicationList,
  updateKappApplication,
  getKappApplicationComponentList,
  createKappApplicationComponent,
  updateKappApplicationComponent,
  deleteKappApplicationComponent,
  getKappApplicationPlugins,
  getKappComponentPlugins
} from "./kubernetesApi";
import { setErrorNotificationAction, setSuccessNotificationAction } from "./notification";
import { SubmissionError } from "redux-form";
import { push } from "connected-react-router";
import { resErrorsToSubmitErrors } from "../utils";
import Immutable from "immutable";
import { setCurrentNamespaceAction } from "./namespaces";

export const createComponentAction = (
  componentValues: ApplicationComponent,
  applicationName?: string
): ThunkResult<Promise<void>> => {
  return async (dispatch, getState) => {
    if (!applicationName) {
      applicationName = getState()
        .get("namespaces")
        .get("active");
    }

    let component: ApplicationComponentDetails;
    try {
      component = await createKappApplicationComponent(applicationName, componentValues);
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch(loadApplicationAction(applicationName));
    dispatch({
      type: CREATE_COMPONENT,
      payload: { applicationName, component }
    });
    dispatch(setSuccessNotificationAction("Create component successfully"));
  };
};

export const updateComponentAction = (
  componentValues: ApplicationComponent,
  applicationName?: string
): ThunkResult<Promise<void>> => {
  return async (dispatch, getState) => {
    if (!applicationName) {
      applicationName = getState()
        .get("namespaces")
        .get("active");
    }

    let component: ApplicationComponentDetails;
    try {
      component = await updateKappApplicationComponent(applicationName, componentValues);
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch(loadApplicationAction(applicationName));
    dispatch({
      type: UPDATE_COMPONENT,
      payload: { applicationName, component }
    });
    dispatch(setSuccessNotificationAction("Update component successfully"));
  };
};

export const deleteComponentAction = (componentName: string, applicationName?: string): ThunkResult<Promise<void>> => {
  return async (dispatch, getState) => {
    if (!applicationName) {
      applicationName = getState()
        .get("namespaces")
        .get("active");
    }

    try {
      await deleteKappApplicationComponent(applicationName, componentName);
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch(loadApplicationAction(applicationName));
    dispatch({
      type: DELETE_COMPONENT,
      payload: { applicationName, componentName }
    });
    dispatch(setSuccessNotificationAction("Delete component successfully"));
  };
};

export const createApplicationAction = (applicationValues: Application): ThunkResult<Promise<void>> => {
  return async dispatch => {
    dispatch(setIsSubmittingApplication(true));
    let application: ApplicationDetails;

    try {
      applicationValues = applicationValues.set("namespace", applicationValues.get("name"));
      application = await createKappApplication(applicationValues);
      // const applicationComponents = Immutable.List(
      //   await Promise.all(
      //     applicationValues.get("components").map(async component => {
      //       const applicationComponent = await createKappApplicationComponent(application.get("name"), component);
      //       return applicationComponent;
      //     })
      //   )
      // );
      // application = application.set("components", applicationComponents);
    } catch (e) {
      console.log(e);
      if (e.response && e.response.data.errors && e.response.data.errors.length > 0) {
        const submitErrors = resErrorsToSubmitErrors(e.response.data.errors);
        throw new SubmissionError(submitErrors);
      } else if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    } finally {
      setTimeout(() => {
        dispatch(setIsSubmittingApplication(false));
      }, 2000);
    }

    dispatch(loadApplicationsAction());
    dispatch({
      type: CREATE_APPLICATION,
      payload: { application }
    });
    dispatch(setSuccessNotificationAction("Create application successfully"));
    dispatch(push("/applications"));
  };
};

export const updateApplicationAction = (applicationRaw: Application): ThunkResult<Promise<void>> => {
  return async dispatch => {
    // const testErrors = [
    //   {
    //     key: ".name",
    //     message: "name errors"
    //   },
    //   {
    //     key: ".components[1].name",
    //     message: "components name errors"
    //   },
    //   {
    //     key: ".components[1].ports",
    //     message: "components ports errors"
    //   }
    // ];
    // const submitErrors = resErrorsToSubmitErrors(testErrors);
    // console.log("throw", submitErrors);
    // throw new SubmissionError(submitErrors);

    dispatch(setIsSubmittingApplication(true));
    let application: ApplicationDetails;

    try {
      application = await updateKappApplication(applicationRaw);
    } catch (e) {
      if (e.response && e.response.data.errors && e.response.data.errors.length > 0) {
        const submitErrors = resErrorsToSubmitErrors(e.response.data.errors);
        throw new SubmissionError(submitErrors);
      } else if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }

      return;
    } finally {
      setTimeout(() => {
        dispatch(setIsSubmittingApplication(false));
      }, 2000);
    }

    dispatch(loadApplicationsAction());
    dispatch({
      type: UPDATE_APPLICATION,
      payload: { application }
    });
    dispatch(setSuccessNotificationAction("Edit application successfully"));
    dispatch(push("/applications"));
  };
};

export const duplicateApplicationAction = (duplicatedApplication: Application): ThunkResult<Promise<void>> => {
  return async dispatch => {
    let application: ApplicationDetails;
    try {
      application = await createKappApplication(duplicatedApplication);
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch(loadApplicationsAction());
    dispatch({
      type: DUPLICATE_APPLICATION,
      payload: { application }
    });
  };
};

export const deleteApplicationAction = (name: string): ThunkResult<Promise<void>> => {
  return async dispatch => {
    try {
      await deleteKappApplication(name);
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch({
      type: DELETE_APPLICATION,
      payload: { applicationName: name }
    });
  };
};

export const loadApplicationAction = (name: string): ThunkResult<Promise<void>> => {
  return async dispatch => {
    dispatch({ type: LOAD_APPLICATION_PENDING });

    let application: ApplicationDetails;
    try {
      application = await getKappApplication(name);
      const applicationComponents = await getKappApplicationComponentList(application.get("name"));
      application = application.set("components", applicationComponents);
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      dispatch({ type: LOAD_APPLICATION_FAILED });
      return;
    }

    dispatch({
      type: LOAD_APPLICATION_FULFILLED,
      payload: {
        application
      }
    });
  };
};

export const loadApplicationsAction = (): ThunkResult<Promise<void>> => {
  return async (dispatch, getState) => {
    dispatch({ type: LOAD_APPLICATIONS_PENDING });

    let applicationList: ApplicationDetailsList;
    try {
      applicationList = await getKappApplicationList();

      applicationList = Immutable.List(
        await Promise.all(
          applicationList.map(async application => {
            const components = await getKappApplicationComponentList(application.get("name"));
            application = application.set("components", components);
            return application;
          })
        )
      );
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      dispatch({ type: LOAD_APPLICATIONS_FAILED });
      return;
    }

    const activeNamespace = getState()
      .get("namespaces")
      .get("active");
    const firstNamespace = applicationList.get(0);
    if (!activeNamespace && applicationList.size > 0 && firstNamespace != null) {
      dispatch(setCurrentNamespaceAction(firstNamespace?.get("name"), false));
    }

    dispatch({
      type: LOAD_APPLICATIONS_FULFILLED,
      payload: {
        applicationList
      }
    });
  };
};

export const loadApplicationPluginsAction = (): ThunkResult<Promise<void>> => {
  return async dispatch => {
    let applicationPlugins;
    try {
      applicationPlugins = await getKappApplicationPlugins();
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch({
      type: LOAD_APPLICATION_PLUGINS_FULFILLED,
      payload: {
        applicationPlugins
      }
    });
  };
};

export const loadComponentPluginsAction = (): ThunkResult<Promise<void>> => {
  return async dispatch => {
    let componentPlugins;
    try {
      componentPlugins = await getKappComponentPlugins();
    } catch (e) {
      if (e.response && e.response.data.status === StatusFailure) {
        dispatch(setErrorNotificationAction(e.response.data.message));
      } else {
        dispatch(setErrorNotificationAction());
      }
      return;
    }

    dispatch({
      type: LOAD_COMPONENT_PLUGINS_FULFILLED,
      payload: {
        componentPlugins
      }
    });
  };
};

export const setIsSubmittingApplication = (isSubmittingApplication: boolean): SetIsSubmittingApplication => {
  return {
    type: SET_IS_SUBMITTING_APPLICATION,
    payload: {
      isSubmittingApplication
    }
  };
};

export const setIsSubmittingApplicationComponent = (
  isSubmittingApplicationComponent: boolean
): SetIsSubmittingApplicationComponent => {
  return {
    type: SET_IS_SUBMITTING_APPLICATION_COMPONENT,
    payload: {
      isSubmittingApplicationComponent
    }
  };
};
