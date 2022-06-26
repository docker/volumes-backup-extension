import React, { useContext } from "react";
import {
  Button,
  TextField,
  Grid,
  Backdrop,
  CircularProgress,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function RunContainerDialog({ ...props }) {
  console.log("RunContainerDialog component rendered.");
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);

  const [image, setImage] = React.useState<string>("");
  const [containerPath, setContainerPath] = React.useState<string>("");
  const [containerName, setContainerName] = React.useState<string>(
    `container-from-vol-${context.store.volumeName}`
  );
  const [containerEnvVars, setContainerEnvVars] = React.useState<string>("");
  const [containerPorts, setContainerPorts] = React.useState<string>("");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const runContainer = async () => {
    setActionInProgress(true);
    try {
      console.log(
        `Running a container from volume ${context.store.volumeName}...`
      );

      let args = [
        `--name=${containerName}`,
        "-d",
        "-i",
        `-v=${context.store.volumeName}:${containerPath}`,
      ];

      if (containerPorts !== "") {
        const ports = containerPorts.split(",");
        for (let i = 0; i < ports.length; i++) {
          args.push(`-p=${ports[i]}`);
        }
      }

      if (containerEnvVars !== "") {
        const envVars = containerEnvVars.split(",");
        for (let i = 0; i < envVars.length; i++) {
          args.push(`-e=${envVars[i]}`);
        }
      }

      args.push(image);

      console.log(args.join(" "));

      const runOutput = await ddClient.docker.cli.exec("run", args);
      console.log(runOutput);
      if (runOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(runOutput.stderr);
        return;
      }

      ddClient.desktopUI.toast.success(
        `Container ${containerName} is running from volume ${context.store.volumeName}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to run container volume ${context.store.volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Run a container from a volume</DialogTitle>
      <DialogContent>
        <Backdrop
          sx={{
            backgroundColor: "rgba(245,244,244,0.4)",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
          open={actionInProgress}
        >
          <CircularProgress color="info" />
        </Backdrop>
        <DialogContentText>
          Use a volume from a backup and attach it to a new container.
        </DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <TextField
              required
              autoFocus
              margin="dense"
              id="image-name"
              label="Image"
              fullWidth
              variant="standard"
              placeholder="redis:latest"
              spellCheck={false}
              onChange={(e) => {
                setImage(e.target.value);
              }}
            />
          </Grid>
          <Grid item>
            <TextField
              required
              margin="dense"
              id="container-name"
              label="Container name"
              fullWidth
              variant="standard"
              defaultValue={containerName}
              spellCheck={false}
              onChange={(e) => {
                setContainerName(e.target.value);
              }}
            />
          </Grid>
          <Grid item>
            <TextField
              required
              margin="dense"
              id="container-path"
              label="Container path"
              fullWidth
              variant="standard"
              placeholder="/data"
              spellCheck={false}
              onChange={(e) => {
                setContainerPath(e.target.value);
              }}
            />
          </Grid>

          <Grid item>
            <TextField
              margin="dense"
              id="container-ports"
              label="Port(s)"
              fullWidth
              variant="standard"
              placeholder="8080:80,8081:81"
              spellCheck={false}
              onChange={(e) => {
                setContainerPorts(e.target.value);
              }}
            />
          </Grid>
          <Grid item>
            <TextField
              margin="dense"
              id="container-env-vars"
              label="Environment variables"
              fullWidth
              variant="standard"
              placeholder="KEY=VALUE,FOO=BAR"
              spellCheck={false}
              onChange={(e) => {
                setContainerEnvVars(e.target.value);
              }}
            />
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Cancel</Button>
        <Button
          onClick={runContainer}
          disabled={containerPath === "" || containerPath === ""}
        >
          Run
        </Button>
      </DialogActions>
    </Dialog>
  );
}
