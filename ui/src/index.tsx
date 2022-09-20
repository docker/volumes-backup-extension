import React, { useState } from "react";
import ReactDOM from "react-dom";
import CssBaseline from "@mui/material/CssBaseline";
import { DockerMuiThemeProvider } from "@docker/docker-mui-theme";

import { App } from "./App";
import type { IVolumeRow } from "./hooks/useGetVolumes";
import { NotificationProvider } from "./NotificationContext";
import { LicenseInfo } from "@mui/x-data-grid-pro";

LicenseInfo.setLicenseKey(process.env["REACT_APP_MUI_LICENSE_KEY"]);

interface IAppContext {
  store: {
    volume: IVolumeRow | null;
  };
  actions: {
    setVolume(v: IVolumeRow | null): void;
  };
}

export const MyContext = React.createContext<IAppContext>(null);

const AppProvider: React.FC = (props) => {
  const [store, setStore] = useState({
    volume: null,
  });

  const actions = {
    setVolume: (value: IVolumeRow | null) =>
      setStore((oldStore) => ({ ...oldStore, volume: value })),
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
        <NotificationProvider>
          <App />
        </NotificationProvider>
      </AppProvider>
    </DockerMuiThemeProvider>
  </React.StrictMode>,
  document.getElementById("root")
);
