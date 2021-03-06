import appLoaded from "./signals/appLoaded";
import navbarClicked from "./signals/navbarClicked";
import homeRouted from "./signals/homeRouted";
import modelInformationsRequest from "./signals/modelInformationsRequest";
import predictURLChanged from "./signals/predictURLChanged";
import batchSizeChanged from "./signals/batchSizeChanged";
import deviceChanged from "./signals/deviceChanged";
import traceLevelChanged from "./signals/traceLevelChanged";
import agentChanged from "./signals/agentChanged";
import predictInputsSet from "./signals/predictInputsSet";
import predictURLAdded from "./signals/predictURLAdded";
import inferenceButtonClicked from "./signals/inferenceButtonClicked";
import modelsRouted from "./signals/modelsRouted";
import frameworksRouted from "./signals/frameworksRouted";
import agentRouted from "./signals/agentRouted";
import agentsRouted from "./signals/agentsRouted";
import aboutRouted from "./signals/aboutRouted";
import openTutorial from "./signals/openTutorial";
import closeTutorial from "./signals/closeTutorial";
import aboutPageRouted from "./signals/aboutPageRouted";

export default {
  state: {
    name: "CarML",
    error: null,
    currentPage: "Home",
    status: {
      isInfering: false,
      isBusy: false,
      isLoaded: false,
      isPredicting: false,
      isLoadingModel: false,
      isLoadingFrameworkAgents: false,
      isLoadingFrameworkManifests: false,
      isLoadingModelAgents: false,
      isLoadingModelManifests: false
    },
    predictInputs: [],
    predictURL: "http://ww4.hdnux.com/photos/41/15/35/8705883/4/920x920.jpg",
    batchSize: 1,
    device: "GPU",
    traceLevel: "FULL_TRACE",
    models: {},
    frameworks: {
      data: []
    },
    selectedAgent: null
  },
  signals: {
    appLoaded,
    homeRouted,
    navbarClicked,
    predictInputsSet,
    predictURLChanged,
    predictURLAdded,
    batchSizeChanged,
    deviceChanged,
    traceLevelChanged,
    agentChanged,
    modelInformationsRequest,
    inferenceButtonClicked,
    modelsRouted,
    frameworksRouted,
    agentRouted,
    agentsRouted,
    aboutRouted,
    openTutorial,
    closeTutorial,
    aboutPageRouted
  }
};
