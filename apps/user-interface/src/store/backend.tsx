import React from "react";

declare global {
  interface SparseBackend {
    id: number;
    name: string;
    baseUrl: string;
    modelsServed: string[];
    weight: number;
    inFlight: number;
    total: number;
    breakerState: "open" | "half-open" | "closed" | "disabled";
    enabled: boolean;
  }

  interface Backend {
    id?: number;
    name: string;
    protocol: "h1" | "h2c" | "h2";
    baseUrl: string;
    enabled: boolean;
    modelsServed: string[];
    weight: number;
    maxConcurrent: number;
    kvCacheAwareRouting: boolean;
    metricsUrl: string;
    scrapeInterval: number;
    maxIdleConnectionsPerHost: number;
    idleConnectionTimeout: number;
    dialTimeout: number;
    streamStallTimeout: number;
    responseHeaderTimeout: number;
    failureThreshold: number;
    rollingWindow: number;
    openBase: number;
    openMax: number;
    backoffFactor: number;
    halfOpenProbes: number;
    halfOpenSuccesses: number;
    healthCheckPath: string;
    healthInterval: number;
    verifyTlsCert: boolean;
    description: string;
    labels: string[];
    createdAt: string;
    updatedAt: string;
  }

  type BackendState = {
    BackendList: SparseBackend[];
    CurrentBackend?: Backend;
  }

  interface SetBackendList {
    type: "SET_BACKEND_LIST";
    payload: SparseBackend[];
  }

  interface AddBackend {
    type: "ADD_BACKEND";
    payload: SparseBackend;
  }

  interface RemoveBackend {
    type: "REMOVE_BACKEND";
    payload: number
  }

  interface SetCurrentBackend {
    type: "SET_CURRENT_BACKEND";
    payload: Backend;
  }

  interface UpdateCurrentBackend {
    type: "UPDATE_CURRENT_BACKEND";
    payload: Backend;
  }

  interface RemovedCurrentBackend {
    type: "REMOVED_CURRENT_BACKEND";
  }

  type BackendAction =
    | SetBackendList
    | AddBackend
    | RemoveBackend
    | SetCurrentBackend
    | UpdateCurrentBackend
    | RemovedCurrentBackend;

  type BackendDispatch = (action: BackendAction) => void;
}

const reducer = (state: BackendState, action: BackendAction): BackendState => {
  switch (action.type) {
    case "ADD_BACKEND":
      return {
        ...state,
        BackendList: [...state.BackendList, action.payload],
      }
    case "SET_BACKEND_LIST":
      return {
        ...state,
        BackendList: action.payload,
      };
    case "REMOVE_BACKEND":
      return {
        ...state,
        BackendList: state.BackendList.filter((backend) => backend.id !== action.payload),
      }
    case "SET_CURRENT_BACKEND":
      return {
        ...state,
        CurrentBackend: action.payload,
      }
    case "UPDATE_CURRENT_BACKEND":
      return {
        ...state,
        CurrentBackend: action.payload,
      }
    case "REMOVED_CURRENT_BACKEND":
      return {
        ...state,
        CurrentBackend: undefined,
      }
  }
};

const initialState: BackendState = {
  BackendList: [],
  CurrentBackend: undefined,
};

const BackendContext = React.createContext<{ state: BackendState; dispatch: BackendDispatch } | undefined>(undefined);

const BackendProvider: React.FC<Record<"children", React.ReactNode>> = ({ children }) => {
  const [state, dispatch] = React.useReducer(reducer, initialState);

  const value = { state, dispatch };
  return (
    <BackendContext.Provider value={value}>
      {children}
    </BackendContext.Provider>
  );
};

function useBackend() {
  const context = React.useContext(BackendContext);
  if (context === undefined) {
    throw new Error("useBackend must be used within a Backend Provider");
  }
  return context;
}

export { BackendProvider, useBackend };
