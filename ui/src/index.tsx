import React, { useEffect, useState } from "react";
import ReactDOM from "react-dom";
import CssBaseline from "@mui/material/CssBaseline";
import { DockerMuiThemeProvider } from "@docker/docker-mui-theme";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { App } from "./App";
import type { IVolumeRow } from "./hooks/useGetVolumes";

const ddClient = createDockerDesktopClient();

interface IAppContext {
  store: {
    volume: IVolumeRow | null;
    sdkVersion: string;
  };
  actions: {
    setVolume(v: IVolumeRow | null): void;
  };
}

export const MyContext = React.createContext<IAppContext>(null);

const AppProvider = (props) => {
  const [store, setStore] = useState({
    volume: null,
    sdkVersion: "",
  });

  const actions = {
    setVolume: (value: IVolumeRow | null) =>
      setStore((oldStore) => ({ ...oldStore, volume: value })),
  };

  useEffect(() => {
    ddClient.docker.cli
      .exec("extension version", [])
      .then((output) => {
        const sdkVersion = output.lines()[1].split(": ")[1];
        setStore((oldStore) => ({ ...oldStore, sdkVersion }));
      })
      .catch((err) => {
        console.error(err);
        setStore((oldstore) => ({ ...oldstore, sdkVersion: "" }));
      });
  }, []);

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
