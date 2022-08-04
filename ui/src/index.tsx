import React, { useState } from "react";
import ReactDOM from "react-dom";
import CssBaseline from "@mui/material/CssBaseline";
import { DockerMuiThemeProvider } from "@docker/docker-mui-theme";

import { App } from "./App";

interface IAppContext {
  store: {
    volumeName: string;
  };
  actions: {
    setVolumeName(v: string): void;
  };
}

export const MyContext = React.createContext<IAppContext>(null);

const AppProvider = (props) => {
  const [store, setStore] = useState({
    volumeName: "",
  });

  const actions = {
    setVolumeName: (value: string) =>
      setStore((oldStore) => ({ ...oldStore, volumeName: value })),
  };

  return (
    <MyContext.Provider value={{ actions, store }}>
      {props.children}
    </MyContext.Provider>
  );
};

ReactDOM.render(
  <React.StrictMode>
    {/*
      If you eject from MUI (which we don't recommend!), you should add
      the `dockerDesktopTheme` class to your root <html> element to get
      some minimal Docker theming.
    */}
    <DockerMuiThemeProvider>
      <CssBaseline />
      <AppProvider>
        <App />
      </AppProvider>
    </DockerMuiThemeProvider>
  </React.StrictMode>,
  document.getElementById("root")
);
